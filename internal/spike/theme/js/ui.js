// ui's client script (ADR 0005), D4 spike. Plain modules, no framework.

// Invoker commands (commandfor/command) are Baseline only since December
// 2025. Where the browser lacks them, do what it would: nothing else runs
// where it has them, so a command never fires twice.
const hasCommands = "command" in HTMLButtonElement.prototype;

if (!hasCommands) {
	document.addEventListener("click", (event) => {
		const button = event.target instanceof Element ? event.target.closest("button[commandfor]") : null;
		if (!button || button.disabled) return;
		const target = document.getElementById(button.getAttribute("commandfor"));
		if (!target) return;
		event.preventDefault();
		switch (button.getAttribute("command")) {
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
				target.showPopover();
				break;
			case "hide-popover":
				target.hidePopover();
				break;
		}
	});
}
