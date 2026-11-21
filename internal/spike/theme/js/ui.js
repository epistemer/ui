// ui's client script (ADR 0005), D4 spike. Plain modules, no framework.

import { evaluate } from "./cond.js";
import "./menu.js";
import "./confirm.js";

// A disabled link (aria-disabled, no href) and anything inside it: no
// click reaches htmx or any other listener. A capturing listener on the
// document runs before the target's own.
document.addEventListener(
	"click",
	(event) => {
		const off = event.target instanceof Element ? event.target.closest('[aria-disabled="true"]') : null;
		if (off) {
			event.preventDefault();
			event.stopImmediatePropagation();
		}
	},
	true,
);

// Invoker commands (commandfor/command) are Baseline only since December
// 2025. Where the browser lacks them, do what it would: nothing else runs
// where it has them, so a command never fires twice. Like the browser, do
// nothing when the target can't take the command.
const hasCommands = "command" in HTMLButtonElement.prototype;

const dialogCommands = new Set(["show-modal", "close", "request-close"]);
const popoverCommands = new Set(["toggle-popover", "show-popover", "hide-popover"]);

if (!hasCommands) {
	document.addEventListener("click", (event) => {
		const button = event.target instanceof Element ? event.target.closest("button[commandfor]") : null;
		if (!button || button.disabled) return;
		const target = document.getElementById(button.getAttribute("commandfor"));
		const command = button.getAttribute("command");
		if (!target) return;
		if (dialogCommands.has(command) && !(target instanceof HTMLDialogElement)) return;
		if (popoverCommands.has(command) && !target.hasAttribute("popover")) return;
		switch (command) {
			case "show-modal":
				if (!target.open) target.showModal();
				break;
			case "close":
				if (target.open) target.close(button.value);
				break;
			case "request-close":
				if (target.open) target.requestClose ? target.requestClose(button.value) : target.close(button.value);
				break;
			case "toggle-popover":
				target.togglePopover();
				break;
			case "show-popover":
				if (!target.matches(":popover-open")) target.showPopover();
				break;
			case "hide-popover":
				if (target.matches(":popover-open")) target.hidePopover();
				break;
			default:
				return; // a custom command: leave it to its listener
		}
		event.preventDefault();
	});
}

// Conditional fields (design §6). A wrapper with data-ui-when holds a
// forms condition; while it doesn't hold, the wrapper is hidden and its
// controls are disabled, so they neither show nor submit, as the server
// clears an inactive field. Nested wrappers chain, as groups do.
function formEnv(form) {
	const controls = (name) => [...form.elements].filter((el) => el.name === name);
	return {
		kind(name) {
			return controls(name).some((el) => el.type === "checkbox") ? "checkbox" : "text";
		},
		values(name) {
			const out = [];
			for (const el of controls(name)) {
				if ((el.type === "checkbox" || el.type === "radio") && !el.checked) continue;
				if (el instanceof HTMLSelectElement) {
					for (const o of el.selectedOptions) out.push(o.value);
				} else if (el.type !== "file") {
					out.push(el.value);
				}
			}
			return out;
		},
		when(name) {
			const el = controls(name)[0];
			const conds = [];
			for (let node = el && el.closest("[data-ui-when]"); node && form.contains(node); node = node.parentElement && node.parentElement.closest("[data-ui-when]")) {
				conds.push(JSON.parse(node.dataset.uiWhen));
			}
			return conds;
		},
	};
}

function refresh(form) {
	const env = formEnv(form);
	for (const wrapper of form.querySelectorAll("[data-ui-when]")) {
		const on = evaluate({ typ: "l", op: "and", args: [JSON.parse(wrapper.dataset.uiWhen)] }, env) && !wrapper.parentElement.closest("[data-ui-when][hidden]");
		wrapper.hidden = !on;
		for (const el of wrapper.querySelectorAll("input, select, textarea, button")) el.disabled = !on;
	}
}

function refreshAll(root) {
	for (const form of root.querySelectorAll("form")) {
		if (form.querySelector("[data-ui-when]")) refresh(form);
	}
}

document.addEventListener("input", (e) => e.target.form && refresh(e.target.form));
document.addEventListener("change", (e) => e.target.form && refresh(e.target.form));
document.addEventListener("htmx:afterSwap", (e) => refreshAll(e.target));
if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", () => refreshAll(document));
} else {
	refreshAll(document);
}
