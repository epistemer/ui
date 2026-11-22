// The spike's script (design §7): <ui-table>, the action dialog, a 422
// that swaps, the refresh event, and menus whose arrow keys stop on a
// disabled item that can be reached (S21).

// htmx 2 swaps no 4xx: a 422 is a form with its errors, so it swaps.
document.addEventListener("htmx:beforeSwap", (e) => {
	if (e.detail.xhr.status === 422) {
		e.detail.shouldSwap = true;
		e.detail.isError = false;
	}
});

// The action dialog opens when content arrives in its body, and closes
// when an answer says so.
const actionDialog = () => document.getElementById("ui-action-dialog");

document.addEventListener("htmx:afterSwap", (e) => {
	const d = actionDialog();
	if (!d || e.detail.target?.id !== "ui-action-dialog-body" || !e.detail.target.childElementCount) return;
	// The loaded title names the dialog; aria-labelledby wins over its
	// generic aria-label.
	if (document.getElementById("ui-action-dialog-title")) d.setAttribute("aria-labelledby", "ui-action-dialog-title");
	else d.removeAttribute("aria-labelledby");
	if (!d.open) d.showModal();
	d.querySelector("#ui-action-dialog-body :is([autofocus], input:not([type=hidden]), select, textarea)")?.focus();
});

// What opened the dialog, by a keyed id: the row, whose trigger takes
// focus back when the dialog closes, even if a refresh replaced it. The
// opener itself may be gone, and a menu's trigger has a generated id.
let opener = null;

document.addEventListener("htmx:beforeRequest", (e) => {
	if (e.detail.target?.id !== "ui-action-dialog-body" || actionDialog()?.open) return;
	const row = e.detail.elt.closest?.("tr[id]");
	opener = row ? { row: row.id } : { id: e.detail.elt.id || "" };
});

function closeActionDialog() {
	const d = actionDialog();
	if (!d?.open) return;
	d.close();
	document.getElementById("ui-action-dialog-body")?.replaceChildren();
	d.removeAttribute("aria-labelledby");
	// Focus goes back once the answer's changes have settled: now, and
	// again after a refresh that replaces the opener (review 0014).
	pending = opener;
	opener = null;
	refocus();
}

let pending = null;

function refocus() {
	if (!pending) return;
	const back = pending.row
		? document.getElementById(pending.row)?.querySelector("ui-menu button")
		: pending.id && document.getElementById(pending.id);
	if (back && (document.activeElement === document.body || !document.activeElement)) back.focus();
}

document.addEventListener("htmx:afterSettle", () => setTimeout(refocus, 0));
document.addEventListener("htmx:beforeRequest", (e) => {
	if (pending && e.detail.target?.id !== "ui-action-dialog-body" && !e.detail.target?.matches?.(".ui-table-results")) pending = null;
});

// Later, not now: htmx fires the answer's other events (a refresh) on the
// element that asked, which emptying the dialog would detach.
document.addEventListener("ui-action-done", () => setTimeout(closeActionDialog, 0));

// However it closes (its close button, Esc, an answer), it is emptied and
// loses the loaded title. Watched through its open attribute: the close
// event waits for rendering (V5's finding).
// Observed on the document, not the dialog itself: a history restore or
// a boosted page replaces the body, and the dialog with it (review 0014).
new MutationObserver((records) => {
	for (const r of records) {
		const d = r.target;
		if (d.id !== "ui-action-dialog" || d.open) continue;
		document.getElementById("ui-action-dialog-body")?.replaceChildren();
		d.removeAttribute("aria-labelledby");
	}
}).observe(document.documentElement, { attributes: true, attributeFilter: ["open"], subtree: true });

// Refresh: the table's results reload at their current URL.
document.addEventListener("ui-table-refresh", (e) => {
	const id = e.detail?.id;
	const results = id && document.getElementById(id + "-results");
	if (!results || !window.htmx) return;
	htmx.ajax("GET", results.dataset.uiUrl, { target: "#" + results.id, select: "#" + results.id, swap: "outerHTML" });
});

// <ui-table>: the header checkbox, the count of checked rows, and status
// and focus after a swap.
class UITable extends HTMLElement {
	#focus = "";

	#results = null;

	connectedCallback() {
		this.addEventListener("change", this);
		this.addEventListener("htmx:beforeRequest", this);
		this.addEventListener("htmx:afterRequest", this);
		this.#results = this.querySelector(".ui-table-results");
		this.sync();
	}

	// After a swap: an outerHTML swap settles on the old, detached element,
	// so the event never reaches the table. The document hears it, and each
	// table checks whether its results are new.
	settled() {
		const results = this.querySelector(".ui-table-results");
		if (results === this.#results) return;
		this.#results = results;
		results?.removeAttribute("aria-busy");
		this.sync();
		if (results) this.#say(results.dataset.uiStatus ?? "");
		const lost = !document.activeElement || document.activeElement === document.body;
		if (this.#focus && lost) document.getElementById(this.#focus)?.focus();
		this.#focus = "";
	}

	// getAttribute, not .id: the rows' checkboxes are named "id" and belong
	// to this form, so form.id is them (named elements shadow properties).
	get #select() {
		return this.querySelector("form[id$='-select']")?.getAttribute("id");
	}

	#rows() {
		const sel = this.#select;
		return sel ? [...this.querySelectorAll(`input[name="id"][form="${sel}"]`)] : [];
	}

	sync() {
		const all = this.querySelector("[data-ui-select-all]");
		if (all) all.hidden = false;
		this.#update(false);
	}

	#update(say) {
		const rows = this.#rows();
		const checked = rows.filter((r) => r.checked).length;
		const all = this.querySelector("[data-ui-select-all]");
		if (all) {
			all.checked = rows.length > 0 && checked === rows.length;
			all.indeterminate = checked > 0 && checked < rows.length;
		}
		if (say) this.#say(checked ? (this.dataset.uiSelected ?? "%d").replace("%d", checked) : "");
	}

	#say(text) {
		const status = this.querySelector(".ui-table-status");
		if (status) status.textContent = text;
	}

	handleEvent(e) {
		switch (e.type) {
			case "change":
				if (e.target.matches("[data-ui-select-all]")) {
					for (const r of this.#rows()) r.checked = e.target.checked;
				}
				if (e.target.matches("[data-ui-select-all], input[name='id']")) this.#update(true);
				break;
			case "htmx:beforeRequest": {
				const a = document.activeElement;
				this.#focus = a && this.contains(a) && a.id ? a.id : "";
				this.querySelector(".ui-table-results")?.setAttribute("aria-busy", "true");
				break;
			}
			case "htmx:afterRequest":
				// A failed request never settles: clear it here too.
				if (!e.detail.successful) this.querySelector(".ui-table-results")?.removeAttribute("aria-busy");
				break;
		}
	}
}

if (!customElements.get("ui-table")) customElements.define("ui-table", UITable);

document.addEventListener("htmx:afterSettle", () => {
	for (const t of document.querySelectorAll("ui-table")) t.settled?.();
});

// A GET from a table's state controls: the table's own parameters come from
// the control (its link or its state form), every other one from the live
// address, which another table may have changed since this one rendered.
// Empty parameters are dropped, so the address holds only what is set.
document.addEventListener("htmx:configRequest", (e) => {
	const table = e.detail.elt.closest?.("ui-table");
	if (e.detail.verb !== "get" || !table) return;
	// Only the table's own state controls: an action's GET (the selection,
	// to the action dialog) keeps what it sends.
	const t = e.detail.target;
	if (t !== table && !t?.matches?.(".ui-table-results")) return;
	const own = new Set((table.dataset.uiOwn ?? "").split(" "));
	const url = new URL(e.detail.path, location.href);
	for (const k of [...url.searchParams.keys()]) {
		if (!own.has(k) || url.searchParams.getAll(k).every((v) => v === "")) url.searchParams.delete(k);
	}
	for (const [k, v] of new URLSearchParams(location.search)) {
		if (!own.has(k)) url.searchParams.append(k, v);
	}
	const p = e.detail.parameters;
	for (const k of [...p.keys()]) {
		if (!own.has(k) || p.getAll(k).every((v) => v === "")) p.delete(k);
	}
	e.detail.path = url.pathname + url.search;
});

// S21: a menu with an item that's disabled but can be reached moves over
// every item, that one included. Capturing, so menu.js doesn't skip it.
document.addEventListener(
	"keydown",
	(e) => {
		const menu = e.target instanceof Element ? e.target.closest(".ui-menu-popover") : null;
		if (!menu || !menu.querySelector("[data-ui-reachable]")) return;
		const items = [...menu.querySelectorAll('[role^="menuitem"]')];
		const i = items.indexOf(document.activeElement);
		let next = -1;
		switch (e.key) {
			case "ArrowDown":
				next = (i + 1) % items.length;
				break;
			case "ArrowUp":
				next = i < 0 ? items.length - 1 : (i - 1 + items.length) % items.length;
				break;
			case "Home":
				next = 0;
				break;
			case "End":
				next = items.length - 1;
				break;
			default:
				return;
		}
		e.preventDefault();
		e.stopImmediatePropagation();
		items[next]?.focus();
	},
	true,
);
