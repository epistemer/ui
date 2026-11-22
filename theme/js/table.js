// Tables (server-state-interface.md §7): <ui-table>, the state requests,
// and Reply's refresh.

// <ui-table>: the header checkbox, the count of checked rows, and the
// status, focus and busy state around a swap of its results.
// The id that had focus when a table's state request left, by table: a
// state form's answer replaces the whole table, element and all.
const focusBefore = new Map();

// A state request targets the table or its results; a row's or a bulk
// action's doesn't.
const isState = (table, target) => target === table || target?.matches?.(".ui-table-results");

class UITable extends HTMLElement {
	#results = null;

	connectedCallback() {
		this.addEventListener("change", this);
		this.addEventListener("htmx:beforeRequest", this);
		this.addEventListener("htmx:afterRequest", this);
		// A table swapped in whole settles like new results.
		this.#results = focusBefore.has(this.id) ? null : this.querySelector(".ui-table-results");
		this.sync();
	}

	// The selection form's id, by attribute: its checkboxes are form
	// controls of it, and a named control could shadow a form's property.
	get #select() {
		return this.querySelector("form[id$='-select']")?.getAttribute("id");
	}

	#rows() {
		const sel = this.#select;
		return sel ? [...this.querySelectorAll(`input[type="checkbox"][form="${CSS.escape(sel)}"]`)] : [];
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
		if (say) this.#say(checked ? (this.dataset.uiSelected ?? "%d").replace("%d", String(checked)) : "");
	}

	#say(text) {
		const status = this.querySelector(".ui-table-status");
		if (status) status.textContent = text;
	}

	// After a swap. An outerHTML swap settles on the old, detached
	// element, so the document hears it and each table checks whether its
	// results are new.
	settled() {
		const results = this.querySelector(".ui-table-results");
		if (results === this.#results) return;
		this.#results = results;
		results?.removeAttribute("aria-busy");
		this.sync();
		if (results) this.#say(results.dataset.uiStatus ?? "");
		const id = focusBefore.get(this.id);
		focusBefore.delete(this.id);
		const lost = !document.activeElement || document.activeElement === document.body;
		if (id === undefined || !lost) return;
		// The control used, by id; gone, the scroll region, else the search.
		const back = (id && document.getElementById(id)) || this.querySelector(".ui-table-scroll") || this.querySelector('input[type="search"]');
		back?.focus();
	}

	handleEvent(e) {
		switch (e.type) {
			case "change":
				if (e.target.matches?.("[data-ui-select-all]")) {
					for (const r of this.#rows()) r.checked = e.target.checked;
				}
				if (e.target.matches?.('[data-ui-select-all], input[type="checkbox"][form]')) this.#update(true);
				break;
			case "htmx:beforeRequest": {
				if (!isState(this, e.detail.target)) break;
				const a = document.activeElement;
				focusBefore.set(this.id, a && this.contains(a) ? a.id : "");
				this.querySelector(".ui-table-results")?.setAttribute("aria-busy", "true");
				break;
			}
			case "htmx:afterRequest":
				// A failed request never settles.
				if (isState(this, e.detail.target) && !e.detail.successful) {
					this.querySelector(".ui-table-results")?.removeAttribute("aria-busy");
					focusBefore.delete(this.id);
				}
				break;
		}
	}
}

if (!customElements.get("ui-table")) customElements.define("ui-table", UITable);

document.addEventListener("htmx:afterSettle", () => {
	for (const t of document.querySelectorAll("ui-table")) t.settled?.();
});

// A GET from a table's state controls (the requests that target the table
// or its results) takes the table's own parameters from the control and
// every other one from the live address, which another table may have
// changed since this one rendered. Empty parameters are dropped.
document.addEventListener("htmx:configRequest", (e) => {
	const table = e.detail.elt.closest?.("ui-table");
	if (e.detail.verb !== "get" || !table) return;
	if (!isState(table, e.detail.target)) return;
	const own = new Set((table.dataset.uiOwn ?? "").split(" ").filter(Boolean));
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

// Reply's refresh: the table's results reload at their current address.
document.addEventListener("ui-table-refresh", (e) => {
	const id = e.detail?.id;
	const results = id && document.getElementById(id + "-results");
	if (!results?.dataset.uiUrl || !window.htmx) return;
	window.htmx.ajax("GET", results.dataset.uiUrl, { target: "#" + results.id, select: "#" + results.id, swap: "outerHTML" });
});
