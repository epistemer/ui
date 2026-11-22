// <ui-menu>: the APG menu button pattern, in light DOM so ui's CSS applies
// (ADR 0005). The popover opens natively (popovertarget), which also
// anchors it to its trigger; this element keeps aria-expanded in step,
// moves focus into the menu, handles the keys (arrows, Home, End, Tab,
// typeahead), closes after an item is chosen, and places the popover
// itself where anchor positioning is missing.

// The fallback can be forced for testing with <html data-ui-no-anchor>.
const anchored = CSS.supports("position-area", "block-end") && !document.documentElement.hasAttribute("data-ui-no-anchor");

const items = (menu) =>
	[...menu.querySelectorAll('[role^="menuitem"]')].filter((el) => !el.disabled && el.getAttribute("aria-disabled") !== "true");

class UIMenu extends HTMLElement {
	// Private fields: an element's own properties (popover, hidden, title…)
	// reflect attributes, and assigning one would rewrite the attribute.
	#panel;
	#trigger;
	#focusLast = false;
	#typed = "";
	#typedAt = 0;

	connectedCallback() {
		this.#panel = this.querySelector("[popover]");
		if (!this.#panel) return;
		this.#trigger = this.querySelector(`[popovertarget="${CSS.escape(this.#panel.id)}"]`);
		this.#trigger?.setAttribute("aria-expanded", String(this.#panel.matches(":popover-open")));
		this.#panel.addEventListener("toggle", this);
		this.#panel.addEventListener("keydown", this);
		this.#panel.addEventListener("click", this);
		this.#trigger?.addEventListener("keydown", this);
	}

	disconnectedCallback() {
		this.#panel?.removeEventListener("toggle", this);
		this.#panel?.removeEventListener("keydown", this);
		this.#panel?.removeEventListener("click", this);
		this.#trigger?.removeEventListener("keydown", this);
		this.#unfollow();
	}

	handleEvent(event) {
		switch (event.type) {
			case "toggle": {
				const open = event.newState === "open";
				this.#trigger?.setAttribute("aria-expanded", String(open));
				if (open) {
					this.#place();
					const list = items(this.#panel);
					(this.#focusLast ? list[list.length - 1] : list[0])?.focus();
				} else {
					this.#unfollow();
				}
				this.#focusLast = false;
				break;
			}
			case "keydown":
				if (event.currentTarget === this.#trigger) this.#triggerKey(event);
				else this.#menuKey(event);
				break;
			case "click":
				// Choosing an item closes the menu; focus goes back to the
				// trigger, natively. A disabled item does nothing.
				{
					const item = event.target.closest('[role^="menuitem"]');
					if (item && !item.disabled && item.getAttribute("aria-disabled") !== "true") this.#panel.hidePopover();
				}
				break;
			case "scroll":
			case "resize":
				this.#place();
				break;
		}
	}

	// ArrowDown and ArrowUp on the trigger open the menu at the first or
	// last item (APG).
	#triggerKey(event) {
		if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
		event.preventDefault();
		this.#focusLast = event.key === "ArrowUp";
		this.#panel.showPopover();
	}

	#menuKey(event) {
		const list = items(this.#panel);
		if (list.length === 0) return;
		const i = list.indexOf(document.activeElement);
		let next;
		switch (event.key) {
			case "ArrowDown":
				next = list[i < 0 ? 0 : (i + 1) % list.length];
				break;
			case "ArrowUp":
				next = list[i < 0 ? list.length - 1 : (i - 1 + list.length) % list.length];
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
				// Typeahead: printable characters move to the next item whose
				// text starts with what was typed in the last half second.
				if (event.key.length !== 1 || event.ctrlKey || event.metaKey || event.altKey) return;
				if (event.timeStamp - this.#typedAt > 500) this.#typed = "";
				this.#typed += event.key.toLocaleLowerCase();
				this.#typedAt = event.timeStamp;
				for (let k = 1; k <= list.length; k++) {
					const candidate = list[(Math.max(i, 0) + (this.#typed.length === 1 ? k : k - 1)) % list.length];
					if (candidate.textContent.trim().toLocaleLowerCase().startsWith(this.#typed)) {
						next = candidate;
						break;
					}
				}
				if (!next) return;
		}
		event.preventDefault();
		next.focus();
	}

	// Where the browser can't anchor the popover to its trigger (Firefox),
	// place it ourselves, through CSSOM (a strict CSP allows it): below the
	// trigger at its start edge, above it when there's no room below, and
	// clamped inside the viewport. It follows scrolling and resizing while
	// open. In RTL the start edge is the right one.
	#place() {
		if (anchored || !this.#trigger || !this.#panel.matches(":popover-open")) return;
		const gap = 4;
		const r = this.#trigger.getBoundingClientRect();
		const style = this.#panel.style;
		style.position = "fixed";
		style.margin = "0";
		style.inset = "auto";
		const p = this.#panel.getBoundingClientRect();
		const vw = document.documentElement.clientWidth;
		const vh = document.documentElement.clientHeight;
		const rtl = getComputedStyle(this).direction === "rtl";
		let top = r.bottom + gap;
		if (top + p.height > vh && r.top - gap - p.height >= 0) top = r.top - gap - p.height;
		let left = rtl ? r.right - p.width : r.left;
		left = Math.min(Math.max(left, gap), Math.max(gap, vw - p.width - gap));
		top = Math.min(Math.max(top, gap), Math.max(gap, vh - p.height - gap));
		style.top = `${top}px`;
		style.left = `${left}px`;
		window.addEventListener("scroll", this, { capture: true, passive: true });
		window.addEventListener("resize", this, { passive: true });
	}

	#unfollow() {
		window.removeEventListener("scroll", this, { capture: true });
		window.removeEventListener("resize", this);
	}
}

if (!customElements.get("ui-menu")) customElements.define("ui-menu", UIMenu);
