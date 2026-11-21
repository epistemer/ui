// ui's client script (ADR 0005), D4 spike. Plain modules, no framework.

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
