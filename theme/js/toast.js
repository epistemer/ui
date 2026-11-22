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
	connectedCallback() {
		this.addEventListener("click", this);
		this.sync();
	}

	handleEvent(event) {
		const close = event.target instanceof Element ? event.target.closest(".ui-toast-close") : null;
		if (close) this.dismiss(close.closest("ui-toast"));
	}

	// open shows the region, so it is in the accessibility tree before a
	// toast goes into it: a live region that appears with its content
	// isn't announced.
	open() {
		if (!this.matches(":popover-open")) this.showPopover();
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

// topModal is the open modal dialog people can reach: the one holding
// focus, else the last opened (a confirmation over a dialog).
function topModal() {
	const modals = [...document.querySelectorAll("dialog")].filter((d) => d.matches(":modal"));
	return document.activeElement?.closest("dialog:modal") ?? modals.at(-1) ?? null;
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
		const back = [...inner.children];
		inner.remove();
		if (back.length) {
			region.open();
			setTimeout(() => region.append(...back), 50);
		} else {
			region.sync();
		}
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
	const inDialog = dialog?.querySelector("[autofocus], button, [href], input, select, textarea");
	if (hadFocus) (next?.querySelector(".ui-toast-close") ?? inDialog)?.focus();
}

class UIToast extends HTMLElement {
	#left = 0;
	#timer = 0;
	#last = 0;

	connectedCallback() {
		const region = document.getElementById("ui-toasts");
		if (!region) return; // no region: stays where it was rendered
		const modal = topModal();
		const home = modal ? dialogRegion(modal, region) : region;
		const shown = home === region ? region.matches(":popover-open") : home.isConnected && home.childElementCount > 0;
		if (this.parentElement !== home || !shown) {
			// Show the region first, then add the toast in a later task, so
			// the live region is there to announce it.
			const from = this.parentElement;
			this.remove();
			if (from === region) region.sync();
			if (home === region) region.open();
			setTimeout(() => home.append(this), 50);
			return;
		}
		if (this.#timer) return; // moved between regions: keep its timeout
		const ms = Number(this.dataset.uiTimeout);
		if (ms > 0) {
			this.#left = ms;
			this.#last = performance.now();
			this.#timer = setInterval(() => this.#tick(), 250);
		}
	}

	disconnectedCallback() {
		// Moving between regions disconnects and connects again; only a
		// toast that stays out is stopped.
		setTimeout(() => {
			if (!this.isConnected) {
				clearInterval(this.#timer);
				this.#timer = 0;
			}
		}, 100);
	}

	// The timeout pauses while the pointer is over the toasts or focus is
	// in them, read from the page each tick, so no event can leave it stuck.
	#tick() {
		const now = performance.now();
		const holder = this.parentElement;
		if (!holder?.matches(":hover, :focus-within")) this.#left -= now - this.#last;
		this.#last = now;
		if (this.#left <= 0) {
			clearInterval(this.#timer);
			this.#timer = 0;
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

// A modal dialog opening makes the page's region inert: its toasts move
// into the dialog.
new MutationObserver((records) => {
	if (!records.some((r) => r.target instanceof HTMLDialogElement && r.target.open)) return;
	const region = document.getElementById("ui-toasts");
	const modal = topModal();
	if (!region || !modal || !region.childElementCount) return;
	const moving = [...region.children];
	region.replaceChildren();
	region.sync();
	const inner = dialogRegion(modal, region);
	setTimeout(() => inner.append(...moving), 50);
}).observe(document.documentElement, { attributes: true, attributeFilter: ["open"], subtree: true });

if (!customElements.get("ui-toasts")) customElements.define("ui-toasts", UIToasts);
if (!customElements.get("ui-toast")) customElements.define("ui-toast", UIToast);
