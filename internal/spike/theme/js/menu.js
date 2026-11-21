// <ui-menu>: the APG menu button pattern, in light DOM so ui's CSS applies
// (ADR 0005). The popover opens natively (popovertarget); this element
// keeps aria-expanded in step, moves focus to the first item, handles the
// arrow keys, Home and End, closes after an item is chosen, and places the
// popover next to its trigger where anchor positioning is missing.

const anchored = CSS.supports("position-area", "block-end");

const items = (menu) =>
	[...menu.querySelectorAll('[role^="menuitem"]')].filter((el) => el.getAttribute("aria-disabled") !== "true");

class UIMenu extends HTMLElement {
	// Private fields: an element's own properties (popover, hidden, title…)
	// reflect attributes, and assigning one would rewrite the attribute.
	#panel;
	#trigger;

	connectedCallback() {
		this.#panel = this.querySelector("[popover]");
		if (!this.#panel) return;
		this.#trigger = this.querySelector(`[popovertarget="${CSS.escape(this.#panel.id)}"]`);
		this.#trigger?.setAttribute("aria-expanded", "false");
		this.#panel.addEventListener("toggle", this);
		this.#panel.addEventListener("keydown", this);
		this.#panel.addEventListener("click", this);
	}

	disconnectedCallback() {
		this.#panel?.removeEventListener("toggle", this);
		this.#panel?.removeEventListener("keydown", this);
		this.#panel?.removeEventListener("click", this);
	}

	handleEvent(event) {
		switch (event.type) {
			case "toggle": {
				const open = event.newState === "open";
				this.#trigger?.setAttribute("aria-expanded", String(open));
				if (open) {
					this.place();
					items(this.#panel)[0]?.focus();
				}
				break;
			}
			case "keydown":
				this.key(event);
				break;
			case "click":
				// Choosing an item closes the menu; focus goes back to the
				// trigger, natively. A confirmation keeps it open until
				// answered (confirm.js closes it).
				if (event.target.closest('[role^="menuitem"]') && !event.defaultPrevented) this.#panel.hidePopover();
				break;
		}
	}

	key(event) {
		const list = items(this.#panel);
		const i = list.indexOf(document.activeElement);
		let next;
		switch (event.key) {
			case "ArrowDown":
				next = list[(i + 1) % list.length];
				break;
			case "ArrowUp":
				next = list[(i - 1 + list.length) % list.length];
				break;
			case "Home":
				next = list[0];
				break;
			case "End":
				next = list[list.length - 1];
				break;
			case "Tab":
				this.#panel.hidePopover();
				return;
			default:
				return;
		}
		event.preventDefault();
		next?.focus();
	}

	// Where the browser can't anchor the popover to its trigger, place it
	// below the trigger's start edge ourselves. CSSOM writes, which a strict
	// CSP allows. In RTL the start edge is the right one.
	place() {
		if (anchored || !this.#trigger) return;
		const r = this.#trigger.getBoundingClientRect();
		const rtl = getComputedStyle(this).direction === "rtl";
		const style = this.#panel.style;
		style.position = "fixed";
		style.margin = "0";
		style.top = `${r.bottom + 4}px`;
		style.left = rtl ? "auto" : `${r.left}px`;
		style.right = rtl ? `${document.documentElement.clientWidth - r.right}px` : "auto";
		style.bottom = "auto";
	}
}

if (!customElements.get("ui-menu")) customElements.define("ui-menu", UIMenu);
