// ui's client script (ADR 0005). Plain modules, no framework.

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
	// What a form submits: no buttons, fieldsets or outputs.
	const submits = (el) =>
		el.name &&
		(el instanceof HTMLSelectElement ||
			el instanceof HTMLTextAreaElement ||
			(el instanceof HTMLInputElement && !["button", "submit", "reset", "image"].includes(el.type)));
	const controls = (name) => [...form.elements].filter((el) => el.name === name && submits(el));
	return {
		kind(name) {
			return controls(name).some((el) => el.type === "checkbox") ? "checkbox" : "text";
		},
		values(name) {
			const out = [];
			for (const el of controls(name)) {
				if ((el.type === "checkbox" || el.type === "radio") && !el.checked) continue;
				if (el instanceof HTMLSelectElement) {
					for (const o of el.selectedOptions) if (!o.disabled) out.push(o.value);
				} else if (el.type === "file") {
					for (const f of el.files) out.push(f.name); // Filled reads presence
				} else {
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

// A wrapper enables only what was disabled for a condition: the marker
// data-ui-when-off, which the server sets on what it renders disabled for
// one and this script on what it disables. A control the app disabled
// itself stays disabled. A fieldset wrapper is a control too. Wrappers
// come in document order, so an outer one is settled before the ones
// inside it.
function refresh(form) {
	const env = formEnv(form);
	for (const wrapper of form.querySelectorAll("[data-ui-when]")) {
		const outerOff = wrapper.parentElement.closest("[data-ui-when][hidden]");
		const on = !outerOff && evaluate(JSON.parse(wrapper.dataset.uiWhen), env);
		wrapper.hidden = !on;
		const controls = wrapper.querySelectorAll("input, select, textarea, button, fieldset");
		for (const el of wrapper instanceof HTMLFieldSetElement ? [wrapper, ...controls] : controls) {
			if (!on && !el.disabled) {
				el.disabled = true;
				el.dataset.uiWhenOff = "";
			} else if (on && "uiWhenOff" in el.dataset && !el.parentElement.closest("[data-ui-when][hidden]")) {
				el.disabled = false;
				delete el.dataset.uiWhenOff;
			}
		}
	}
}

// After an htmx swap: the forms inside what was swapped, and the form
// around it when a field was swapped inside a form.
function refreshAll(root) {
	if (!(root instanceof Element || root instanceof Document)) return;
	const forms = new Set(root.querySelectorAll("form"));
	if (root instanceof Element) {
		const around = root.closest("form");
		if (around) forms.add(around);
	}
	for (const form of forms) {
		if (form.querySelector("[data-ui-when]")) refresh(form);
	}
}

document.addEventListener("input", (e) => e.target.form && refresh(e.target.form));
document.addEventListener("change", (e) => e.target.form && refresh(e.target.form));
// reset fires before the values are reset.
document.addEventListener("reset", (e) => setTimeout(() => refresh(e.target), 0));
document.addEventListener("htmx:afterSwap", (e) => refreshAll(e.target));
if (document.readyState === "loading") {
	document.addEventListener("DOMContentLoaded", () => refreshAll(document));
} else {
	refreshAll(document);
}

// Tooltips (§5.3, WCAG 1.4.13): Esc hides a tooltip showing for the
// hovered or keyboard-focused trigger. The key stops here, so a dialog or
// popover around the trigger stays open; but an open menu or popover that
// doesn't hold the trigger gets the key, since Esc is meant for it. The
// tooltip comes back when its trigger is entered or focused again.
const showing = (tip) => tip.matches(":hover, :has(:focus-visible)");

document.addEventListener(
	"keydown",
	(event) => {
		if (event.key !== "Escape") return;
		const open = [...document.querySelectorAll(":popover-open, dialog:modal")];
		let hid = false;
		for (const tip of document.querySelectorAll(".ui-tooltip")) {
			const text = tip.querySelector(":scope > .ui-tooltip-text");
			if (!text || text.hidden || !showing(tip)) continue;
			if (open.length && !open.some((layer) => layer.contains(tip))) continue;
			text.hidden = true;
			hid = true;
			const back = () => {
				text.hidden = false;
				tip.removeEventListener("pointerenter", back);
				tip.removeEventListener("focusin", back);
			};
			// Leaving and coming back, or focus arriving again (a dialog
			// reopened), shows it again.
			tip.addEventListener("pointerleave", () => tip.addEventListener("pointerenter", back), { once: true });
			tip.addEventListener("focusout", () => tip.addEventListener("focusin", back), { once: true });
		}
		if (hid) {
			event.preventDefault();
			event.stopImmediatePropagation();
		}
	},
	true,
);

// The script has run (§10.1): CSS shows what needs it.
document.documentElement.dataset.uiReady = "";
