// <ui-toasts> and <ui-toast> (visual-interface.md §7.5, §10.5): light-DOM
// custom elements. A toast moves itself into the region when it connects
// outside it, so a toast rendered anywhere in a page is announced like one
// that arrives by htmx. The region is a manual popover: shown while it
// holds a toast, and shown again when one arrives.
//
// A modal dialog makes the rest of the page inert, a popover shown above it
// included, so a toast there could be seen but not reached or announced
// (checked in V5). While a modal dialog is open, toasts go into a region
// inside it, and back to the page's region when it closes.

class UIToasts extends HTMLElement {
	#paused = false;

	connectedCallback() {
		this.addEventListener("pointerenter", this);
		this.addEventListener("pointerleave", this);
		this.addEventListener("focusin", this);
		this.addEventListener("focusout", this);
		this.addEventListener("click", this);
		this.sync();
	}

	get paused() {
		return this.#paused;
	}

	handleEvent(event) {
		switch (event.type) {
			case "pointerenter":
			case "focusin":
				this.#paused = true;
				break;
			case "pointerleave":
				this.#paused = this.matches(":focus-within");
				break;
			case "focusout":
				this.#paused = this.matches(":hover") || this.contains(event.relatedTarget);
				break;
			case "click": {
				const close = event.target instanceof Element ? event.target.closest(".ui-toast-close") : null;
				if (close) this.dismiss(close.closest("ui-toast"));
				break;
			}
		}
	}

	// raise shows the region again, on top of anything opened since.
	raise() {
		if (this.matches(":popover-open")) this.hidePopover();
		if (this.childElementCount) this.showPopover();
	}

	sync() {
		const open = this.matches(":popover-open");
		if (this.childElementCount && !open) this.showPopover();
		if (!this.childElementCount && open) this.hidePopover();
	}

	// dismiss removes a toast; focus goes to the next toast's close button,
	// else the previous one's, else the page's main content.
	dismiss(toast) {
		if (!toast) return;
		const hadFocus = toast.contains(document.activeElement);
		const next = toast.nextElementSibling ?? toast.previousElementSibling;
		toast.remove();
		this.sync();
		if (hadFocus) (next?.querySelector(".ui-toast-close") ?? document.getElementById("ui-main"))?.focus();
	}
}

// dialogRegion is the region inside an open modal dialog, made on demand;
// when the dialog closes, its toasts go back to the page's region.
function dialogRegion(dialog, region) {
	let inner = dialog.querySelector(":scope > .ui-toasts[data-ui-in-dialog]");
	if (inner) return inner;
	inner = document.createElement("div");
	inner.className = "ui-toasts";
	inner.dataset.uiInDialog = "";
	inner.setAttribute("aria-live", "polite");
	dialog.append(inner);
	// The open attribute goes as the dialog closes, in the same task; the
	// close event waits for rendering, which a page in the background holds
	// back (design finding 13).
	const watch = new MutationObserver(() => {
		if (dialog.open) return;
		watch.disconnect();
		for (const t of [...inner.children]) region.append(t);
		inner.remove();
	});
	watch.observe(dialog, { attributes: true, attributeFilter: ["open"] });
	return inner;
}

// dismissToast removes a toast; focus goes to the next toast's close
// button, else the previous one's, else the dialog it sat in, else the
// page's main content.
function dismissToast(toast) {
	const region = toast.closest("ui-toasts");
	if (region) return region.dismiss(toast);
	const hadFocus = toast.contains(document.activeElement);
	const next = toast.nextElementSibling ?? toast.previousElementSibling;
	const dialog = toast.closest("dialog");
	toast.remove();
	if (hadFocus) (next?.querySelector(".ui-toast-close") ?? dialog)?.focus();
}

class UIToast extends HTMLElement {
	#left = 0;
	#timer = 0;
	#last = 0;

	connectedCallback() {
		const region = document.getElementById("ui-toasts");
		if (!region) return; // no region: stays where it was rendered
		const modal = [...document.querySelectorAll("dialog")].find((d) => d.matches(":modal"));
		const home = modal ? dialogRegion(modal, region) : region;
		if (this.parentElement !== home) {
			home.append(this); // connects again, in its region
			return;
		}
		if (home === region) region.raise?.();
		if (this.#timer) return; // moved back from a dialog: keep its timeout
		const ms = Number(this.dataset.uiTimeout);
		if (ms > 0) {
			this.#left = ms;
			this.#last = performance.now();
			this.#timer = setInterval(() => this.#tick(), 250);
		}
	}

	disconnectedCallback() {
		// Moving between regions disconnects and connects again in one task;
		// only a toast that stays out is stopped.
		queueMicrotask(() => {
			if (!this.isConnected) clearInterval(this.#timer);
		});
	}

	#tick() {
		const now = performance.now();
		const holder = this.parentElement;
		const paused = holder?.paused ?? holder?.matches(":hover, :focus-within");
		if (!paused) this.#left -= now - this.#last;
		this.#last = now;
		if (this.#left <= 0) {
			clearInterval(this.#timer);
			dismissToast(this);
		}
	}
}

// Close buttons of toasts inside a dialog's region: the page's region
// handles its own.
document.addEventListener("click", (event) => {
	const close = event.target instanceof Element ? event.target.closest(".ui-toasts[data-ui-in-dialog] .ui-toast-close") : null;
	if (close) dismissToast(close.closest("ui-toast"));
});

if (!customElements.get("ui-toasts")) customElements.define("ui-toasts", UIToasts);
if (!customElements.get("ui-toast")) customElements.define("ui-toast", UIToast);
