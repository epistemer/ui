// Uploads (rich-inputs-interface.md §6, rich-inputs-design.md §4.3–§4.5):
// keep cards, chosen files with remove and optional image previews, size and
// type checks before sending, upload progress, and 413 handling.

const textsOf = (el) => {
	try {
		return JSON.parse(el.dataset.uiTexts || "{}");
	} catch {
		return {};
	}
};

// fmt fills the single %s or %d verb in a single pass, so a verb inside a
// file name stays as it is (rich-inputs-interface.md §3.1).
export function fmt(s, arg) {
	return String(s ?? "").replace(/%[sd]/, () => String(arg ?? ""));
}

export function size(n) {
	return n >= 1 << 20 ? `${(n / (1 << 20)).toFixed(1)} MB` : n >= 1024 ? `${Math.round(n / 1024)} KB` : `${n} B`;
}

// fitsAccept reports whether a file with name and MIME type matches an
// input's comma-separated accept attribute. An untyped file defers to the
// server unless accept names an extension.
export function fitsAccept(name, type, acceptAttr) {
	const accept = (acceptAttr || "")
		.split(",")
		.map((s) => s.trim().toLowerCase())
		.filter(Boolean);
	if (!accept.length) return true;
	const t = (type || "").toLowerCase();
	if (!t && !accept.some((a) => a.startsWith("."))) return true;
	const n = (name || "").toLowerCase();
	return accept.some((a) =>
		a.endsWith("/*") ? t.startsWith(a.slice(0, -1)) : a.startsWith(".") ? n.endsWith(a) : t === a,
	);
}

const say = (status, text) => {
	if (!status) return;
	status.textContent = "";
	setTimeout(() => (status.textContent = text), 0);
};

function setUpUpload(field) {
	if (field.dataset.uiUp !== undefined) return;
	field.dataset.uiUp = "";
	const t = textsOf(field);
	for (const li of field.querySelectorAll(".ui-upload-file")) {
		const box = li.querySelector('input[type="checkbox"]');
		if (!box) continue;
		const name = li.querySelector("a")?.textContent.trim() ?? li.querySelector("label")?.textContent.trim() ?? "";
		const b = document.createElement("button");
		b.type = "button";
		b.className = "ui-button ui-upload-toggle";
		b.dataset.variant = "ghost";
		const sync = () => {
			li.toggleAttribute("data-ui-removed", !box.checked);
			b.textContent = box.checked ? fmt(t.remove, name) : t.undo;
		};
		b.addEventListener("click", () => {
			box.checked = !box.checked;
			sync();
			b.focus();
		});
		box.classList.add("ui-visually-hidden");
		box.tabIndex = -1;
		li.append(b);
		sync();
	}
}

function revokePreviews(list) {
	if (!list || typeof URL === "undefined" || typeof URL.revokeObjectURL !== "function") return;
	for (const img of list.querySelectorAll('img[src^="blob:"]')) {
		URL.revokeObjectURL(img.src);
	}
}

function chosen(input) {
	// Only an upload field: it carries the texts its checks need.
	const field = input.closest(".ui-upload");
	if (!field) return;
	const list = field.querySelector(".ui-upload-chosen");
	const t = textsOf(field);
	const max = Number(input.dataset.uiMaxSize || 0);
	let problem = "";
	for (const f of input.files) {
		if (max && f.size > max) problem ||= fmt(t.tooLarge, f.name);
		if (input.accept && !fitsAccept(f.name, f.type, input.accept)) problem ||= fmt(t.wrongType, f.name);
	}
	input.setCustomValidity(problem);
	let err = field.querySelector(".ui-upload-check");
	const tokens = () => (input.getAttribute("aria-describedby") ?? "").split(/\s+/).filter(Boolean);
	if (problem) {
		if (!err) {
			err = document.createElement("p");
			err.className = "ui-field-error ui-upload-check";
			err.id = `${input.id}-check`;
			input.after(err);
		}
		const cur = tokens();
		if (!cur.includes(err.id)) input.setAttribute("aria-describedby", [...cur, err.id].join(" "));
		err.textContent = problem;
		input.setAttribute("aria-invalid", "true");
	} else if (err) {
		const rest = tokens().filter((d) => d !== err.id);
		if (rest.length) input.setAttribute("aria-describedby", rest.join(" "));
		else input.removeAttribute("aria-describedby");
		err.remove();
		// The server's own error, if any, still stands.
		if (!field.querySelector(".ui-field-error")) input.removeAttribute("aria-invalid");
	}
	if (!list) return;
	revokePreviews(list);
	const previews = field.hasAttribute("data-ui-previews");
	list.replaceChildren(
		...[...input.files].map((f, i) => {
			const li = document.createElement("li");
			if (previews && /^image\//i.test(f.type) && typeof URL !== "undefined" && typeof URL.createObjectURL === "function") {
				const img = document.createElement("img");
				img.className = "ui-upload-thumb";
				img.src = URL.createObjectURL(f);
				img.alt = "";
				li.append(img, " ");
			}
			li.append(`${f.name} (${size(f.size)}) `);
			const b = document.createElement("button");
			b.type = "button";
			b.className = "ui-button";
			b.dataset.variant = "ghost";
			b.textContent = fmt(t.remove, f.name);
			b.addEventListener("click", () => {
				const dt = new DataTransfer();
				[...input.files].forEach((g, j) => j !== i && dt.items.add(g));
				input.files = dt.files;
				// As if chosen: conditions and this list hear it.
				input.dispatchEvent(new Event("input", { bubbles: true }));
				input.dispatchEvent(new Event("change", { bubbles: true }));
				input.focus();
			});
			li.append(b);
			return li;
		}),
	);
	const names = [...input.files].map((f) => f.name).join(", ");
	if (names) say(field.querySelector(":scope > [role='status']"), fmt(t.chosen, names));
}

// A body past the route's limit is a 413, which htmx doesn't swap: the
// form stays as it is, chosen files included, and its upload fields say
// why (§4.5).
function bodyTooLarge(e) {
	const form = e.detail.elt?.closest?.("form") ?? e.detail.elt;
	const fields = [...(form?.querySelectorAll?.(".ui-upload") ?? [])].filter(
		(f) => f.querySelector('input[type="file"]')?.files.length,
	);
	for (const field of fields) {
		const input = field.querySelector('input[type="file"]');
		const t = textsOf(field);
		let err = field.querySelector(".ui-upload-check");
		if (!err) {
			err = document.createElement("p");
			err.className = "ui-field-error ui-upload-check";
			err.id = `${input.id}-check`;
			input.after(err);
		}
		const cur = (input.getAttribute("aria-describedby") ?? "").split(/\s+/).filter(Boolean);
		if (!cur.includes(err.id)) input.setAttribute("aria-describedby", [...cur, err.id].join(" "));
		err.textContent = t.bodyTooLarge ?? "";
		input.setAttribute("aria-invalid", "true");
	}
	fields[0]?.querySelector('input[type="file"]')?.focus();
}

if (typeof document !== "undefined") {
	document.addEventListener("change", (e) => {
		if (e.target instanceof HTMLInputElement && e.target.type === "file") chosen(e.target);
	});

	// ui's forms are novalidate, so a custom validity blocks nothing by
	// itself: an upload that failed its check refuses the submit here, before
	// htmx or the browser sends it, and takes focus.
	document.addEventListener(
		"submit",
		(e) => {
			const bad = [...(e.target.querySelectorAll?.('.ui-upload input[type="file"]') ?? [])].find(
				(i) => i.validity.customError,
			);
			if (!bad) return;
			e.preventDefault();
			e.stopImmediatePropagation();
			bad.focus();
		},
		true,
	);

	for (const ev of ["dragenter", "dragover"]) {
		document.addEventListener(ev, (e) => e.target.closest?.(".ui-upload")?.setAttribute("data-ui-dragging", ""));
	}
	document.addEventListener("dragleave", (e) => {
		const field = e.target.closest?.(".ui-upload");
		// Crossing into a child isn't leaving.
		if (field && !field.contains(e.relatedTarget)) field.removeAttribute("data-ui-dragging");
	});
	document.addEventListener("drop", (e) => e.target.closest?.(".ui-upload")?.removeAttribute("data-ui-dragging"));

	// Progress (R19): htmx:xhr:progress names the upload and the download
	// alike, so the bar listens to the request's own upload, which
	// htmx:beforeSend hands over before the body is sent.
	document.addEventListener("htmx:beforeSend", (e) => {
		const form = e.detail.elt?.closest?.("form") ?? e.detail.elt;
		const bars = [...(form?.querySelectorAll?.(".ui-upload") ?? [])]
			.filter((f) => f.querySelector('input[type="file"]')?.files.length)
			.map((f) => f.querySelector(".ui-upload-progress"))
			.filter(Boolean);
		if (!bars.length || !e.detail.xhr?.upload) return;
		e.detail.xhr.upload.addEventListener("progress", (p) => {
			if (!p.lengthComputable || !p.total) return;
			for (const b of bars) {
				b.hidden = false;
				b.value = Math.round((100 * p.loaded) / p.total);
			}
		});
	});

	// Only a 413: a send error is as likely a network gone (review 0021).
	document.addEventListener("htmx:responseError", (e) => e.detail.xhr?.status === 413 && bodyTooLarge(e));

	document.addEventListener("htmx:afterRequest", (e) => {
		for (const bar of (e.detail.elt?.closest?.("form") ?? document).querySelectorAll(".ui-upload-progress")) {
			bar.hidden = true;
		}
	});

	const setUp = (root) => root.querySelectorAll?.(".ui-upload").forEach(setUpUpload);
	setUp(document);
	document.addEventListener("htmx:afterSwap", (e) => setUp(e.target));
}
