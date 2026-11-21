// The shared confirmation (ADR 0005, design §6): one <dialog> per page,
// rendered by the layout, asks the question an action carries in
// data-ui-confirm. htmx actions are answered through htmx:confirm; links
// and submit buttons through their click.

const dialog = () => document.querySelector("dialog[data-ui-confirm-dialog]");

function ask(question) {
	const d = dialog();
	if (!d) return Promise.resolve(window.confirm(question)); // no dialog on the page: the browser's own
	d.querySelector("#ui-confirm-question").textContent = question;
	d.returnValue = "";
	d.showModal();
	// Answer on the click of the dialog's own buttons, so the action runs in
	// the same task as the person's click; the close event (Esc) only
	// cancels. Chromium dispatches close with rendering, which a background
	// page may hold back (spike finding).
	return new Promise((resolve) => {
		let settled = false;
		const done = (yes) => {
			if (settled) return;
			settled = true;
			d.removeEventListener("click", onClick);
			d.removeEventListener("close", onClose);
			resolve(yes);
		};
		const onClick = (event) => {
			const b = event.target instanceof Element ? event.target.closest("button[value]") : null;
			if (b && d.contains(b)) done(b.value === "confirm");
		};
		const onClose = () => done(d.returnValue === "confirm");
		d.addEventListener("click", onClick);
		d.addEventListener("close", onClose);
	});
}

const isHtmx = (el) => [...el.attributes].some((a) => /^(hx|data-hx)-(get|post|put|patch|delete)$/.test(a.name));

// htmx: every request fires htmx:confirm; ask when the element asks.
document.addEventListener("htmx:confirm", (event) => {
	const el = event.detail.elt.closest("[data-ui-confirm]");
	if (!el) return;
	event.preventDefault();
	ask(el.dataset.uiConfirm).then((yes) => yes && event.detail.issueRequest(true));
});

// Links and submit buttons: ask before the browser follows or submits.
let confirmed = null;
document.addEventListener("click", (event) => {
	const el = event.target instanceof Element ? event.target.closest("[data-ui-confirm]") : null;
	if (!el || isHtmx(el) || el === confirmed) return;
	event.preventDefault();
	event.stopPropagation();
	el.closest("[popover]")?.hidePopover();
	ask(el.dataset.uiConfirm).then((yes) => {
		if (!yes) return;
		if (el instanceof HTMLAnchorElement && el.href) {
			location.assign(el.href);
		} else if (el.form && el.type === "submit") {
			confirmed = el;
			el.form.requestSubmit(el);
			confirmed = null;
		}
	});
});
