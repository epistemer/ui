// Actions (server-state-interface.md §4.3, §4.4, §7): a 422 that swaps,
// the action dialog, and what Reply's events do.

// htmx 2 swaps no 4xx: a 422 is a form with its errors, or Reply.Fail's
// toast, so it swaps.
document.addEventListener("htmx:beforeSwap", (e) => {
	if (e.detail.xhr.status === 422) {
		e.detail.shouldSwap = true;
		e.detail.isError = false;
	}
});

const dialog = () => document.getElementById("ui-action-dialog");
const body = () => document.getElementById("ui-action-dialog-body");

// What started an action, by ids a swap keeps: its own (a keyed menu
// trigger, a keyed bulk button), its row's, its table's. A refresh or a
// replaced row can remove the element itself.
let opener = null;

const isAction = (elt, target) =>
	!elt?.matches?.("ui-combobox") &&
	(target?.id === "ui-action-dialog-body" ||
		elt.matches?.('[form="ui-action-form"], .ui-table-bulk [form]') ||
		elt.closest?.(".ui-menu-popover, .ui-action-buttons"));

document.addEventListener("htmx:beforeRequest", (e) => {
	const elt = e.detail.elt;
	// A request from inside the open dialog is its form: keep its opener.
	if (dialog()?.open || !isAction(elt, e.detail.target)) return;
	const menu = elt.closest?.("ui-menu");
	const trigger = menu?.querySelector(":scope > button, :scope > a");
	opener = {
		id: (trigger ?? elt).id || "",
		row: elt.closest?.("tr[id]")?.id || "",
		table: elt.closest?.("ui-table")?.id || "",
	};
});

// The dialog opens when content arrives in its body, named by the loaded
// ActionTitle, with focus on its first field.
document.addEventListener("htmx:afterSwap", (e) => {
	const d = dialog();
	if (!d || e.detail.target?.id !== "ui-action-dialog-body" || !e.detail.target.childElementCount) return;
	if (document.getElementById("ui-action-dialog-title")) d.setAttribute("aria-labelledby", "ui-action-dialog-title");
	else d.removeAttribute("aria-labelledby");
	if (!d.open) d.showModal();
	d.querySelector("#ui-action-dialog-body :is([autofocus], input:not([type=hidden]), select, textarea, button)")?.focus();
});

// However it closes (its button, Esc, an answer), it is emptied and loses
// the loaded name; closed without an answer, its opener is forgotten.
// Watched on the document through the open attribute: a dialog's close
// event waits for rendering, and a swap of the body can replace the dialog
// itself.
function cleared(d) {
	body()?.replaceChildren();
	d.removeAttribute("aria-labelledby");
}

new MutationObserver((records) => {
	for (const r of records) {
		if (r.target.id !== "ui-action-dialog" || r.target.open) continue;
		cleared(r.target);
		if (!closing) opener = null;
	}
}).observe(document.documentElement, { attributes: true, attributeFilter: ["open"], subtree: true });

let closing = false;

// Focus goes back to the opener once the answer's changes have settled:
// now, and again after a refresh that replaced it. Gone (a removed row, a
// cleared selection), it goes to the opener's table: its scroll region,
// else its search box.
let pending = null;

function refocus() {
	if (!pending) return;
	const active = document.activeElement;
	const lost = !active || active === document.body || !active.isConnected || active.closest("dialog:not([open])");
	if (!lost) return;
	const visible = (el) => el && el.isConnected && el.checkVisibility?.() !== false;
	const byID = pending.id ? document.getElementById(pending.id) : null;
	const inRow = pending.row ? document.getElementById(pending.row)?.querySelector("ui-menu button") : null;
	const table = pending.table ? document.getElementById(pending.table) : null;
	const fallback = table?.querySelector(".ui-table-scroll") || table?.querySelector('input[type="search"]');
	const target = [byID, inRow, fallback].find(visible);
	target?.focus();
}

document.addEventListener("htmx:afterSettle", () => setTimeout(refocus, 0));
document.addEventListener("htmx:beforeRequest", (e) => {
	// Another request, not the answer's own refresh: the user moved on.
	if (e.defaultPrevented) return;
	const t = e.detail.target;
	if (pending && t?.id !== "ui-action-dialog-body" && !t?.matches?.(".ui-table-results")) pending = null;
});

// Reply's ui-action-done closes the dialog, a task later: htmx fires the
// answer's other events (a refresh) on the element that asked, which
// emptying the dialog would detach. The dialog is emptied before focus
// moves, so nothing in it holds focus.
document.addEventListener("ui-action-done", () =>
	setTimeout(() => {
		const d = dialog();
		pending = opener;
		opener = null;
		if (d?.open) {
			closing = true;
			d.close();
			cleared(d);
			closing = false;
		}
		refocus();
	}, 0),
);
