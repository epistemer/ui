// The shared confirmation (ADR 0005, design §6): one <dialog> per page,
// rendered by the layout, asks the question an action carries in
// data-ui-confirm.
//
// One path for every action a click starts (a link, a submit button, an
// htmx element, the app's own listener): a capturing listener stops the
// click before anything else sees it, asks, and on yes dispatches the
// click again, which this listener lets through. htmx:confirm covers the
// requests a click doesn't start (a form's submit, hx-trigger="change").

const dialog = () => document.querySelector("dialog[data-ui-confirm-dialog]");

function ask(question) {
	const d = dialog();
	if (!d) return Promise.resolve(window.confirm(question)); // no dialog on the page: the browser's own
	if (d.open) d.close("");
	d.querySelector("#ui-confirm-question").textContent = question;
	d.returnValue = "";
	d.showModal();
	// Answer on the click of the dialog's own buttons, closing the dialog
	// first, so the action runs after it is gone, in the person's own task;
	// the close event (Esc) only cancels. Chromium dispatches close with
	// rendering, which a page in the background holds back.
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
			if (!b || !d.contains(b)) return;
			event.preventDefault(); // the native command would close it again
			d.close(b.value);
			done(b.value === "confirm");
		};
		const onClose = () => done(d.returnValue === "confirm");
		d.addEventListener("click", onClick);
		d.addEventListener("close", onClose);
	});
}

let confirmed = null; // the element whose click is being dispatched again

function closeMenu(el) {
	// A menu closes before the dialog opens, so focus goes back to the
	// menu's trigger and from there to the dialog, not to a hidden item.
	el.closest("[popover]")?.hidePopover();
}

document.addEventListener(
	"click",
	(event) => {
		const el = event.target instanceof Element ? event.target.closest("[data-ui-confirm]") : null;
		if (!el || el === confirmed || el instanceof HTMLFormElement) return;
		// A click on a field that takes input isn't an action: a select or a
		// text box asks through htmx:confirm when its change sends a request.
		// Ticking a box or a radio is the action itself, so it asks here.
		if (el instanceof HTMLSelectElement || el instanceof HTMLTextAreaElement) return;
		if (el instanceof HTMLInputElement && !["submit", "button", "image", "reset", "checkbox", "radio"].includes(el.type)) return;
		if (el.closest('[aria-disabled="true"]') || el.disabled) return;
		event.preventDefault();
		event.stopImmediatePropagation();
		closeMenu(el);
		const init = {
			bubbles: true,
			cancelable: true,
			button: event.button,
			ctrlKey: event.ctrlKey,
			metaKey: event.metaKey,
			shiftKey: event.shiftKey,
			altKey: event.altKey,
		};
		ask(el.dataset.uiConfirm).then((yes) => {
			if (!yes) return;
			confirmed = el;
			try {
				el.dispatchEvent(new MouseEvent("click", init)); // a link follows, a submit button submits, htmx requests
			} finally {
				confirmed = null;
			}
		});
	},
	true,
);

// A form that carries the question asks on submit, unless htmx sends it
// (htmx:confirm asks then).
document.addEventListener(
	"submit",
	(event) => {
		const form = event.target;
		if (!(form instanceof HTMLFormElement) || !form.dataset.uiConfirm || form === confirmed) return;
		if ([...form.attributes].some((a) => /^(data-)?hx-(get|post|put|patch|delete)$/.test(a.name))) return;
		event.preventDefault();
		const submitter = event.submitter;
		ask(form.dataset.uiConfirm).then((yes) => {
			if (!yes) return;
			confirmed = form;
			try {
				form.requestSubmit(submitter);
			} finally {
				confirmed = null;
			}
		});
	},
	true,
);

document.addEventListener("htmx:confirm", (event) => {
	if (confirmed) return; // the click asked already
	const el = event.detail.elt.closest("[data-ui-confirm]");
	if (!el) return;
	event.preventDefault();
	closeMenu(el);
	ask(el.dataset.uiConfirm).then((yes) => yes && event.detail.issueRequest(true));
});
