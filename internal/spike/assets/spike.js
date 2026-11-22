// D8.2's spike: <ui-combobox>, <ui-repeat> and the upload behaviours of
// rich-inputs-design.md §3–§5. No htmx dependency: it uses htmx's events
// when htmx is there, nothing more.

const textsOf = (el) => {
	try {
		return JSON.parse(el.dataset.uiTexts || "{}");
	} catch {
		return {};
	}
};

// fmt fills %s and %d in order, as the server's texts keep them.
const fmt = (s, ...args) => {
	let i = 0;
	return String(s ?? "").replace(/%[sd]/g, () => String(args[i++] ?? ""));
};

const say = (status, text) => {
	if (!status) return;
	// A change of text is what a live region announces; clear first so the
	// same text twice is still heard.
	status.textContent = "";
	// A task, not a frame: a hidden page runs no frames.
	setTimeout(() => (status.textContent = text), 0);
};

const visible = (el) => !el.closest("[hidden]") && !el.closest("template");
const focusable =
	'input:not([type="hidden"]):not(:disabled), select:not(:disabled), textarea:not(:disabled), button:not(:disabled):not([hidden]):not([tabindex="-1"]):not([aria-disabled="true"])';
const firstControl = (root) => [...root.querySelectorAll(focusable)].find(visible);

// The same test as the CSS that positions the popup (spike.css).
const anchors = CSS.supports("position-area: block-end");
let seq = 0;

// <ui-combobox> (§3): the APG editable combobox with list autocomplete.
class UICombobox extends HTMLElement {
	#up = false;
	#t = {};
	#input;
	#hidden;
	#popup;
	#list;
	#status;
	#clear;
	#committed = { value: "", text: "" };
	#dirty = false;
	#timer = 0;
	#abort = null;
	#pending = false;
	#want = 0;
	#initial = { value: "", text: "" };
	#live;

	connectedCallback() {
		// Without a search link (the gate refused it) it stays the text
		// input of the id (R7).
		if (this.#up || !this.dataset.uiSearch) return;
		const input = this.querySelector(":scope > input.ui-input");
		if (!input) return;
		this.#up = true;
		this.#t = textsOf(this);
		const chosen = this.querySelector(":scope > .ui-combobox-chosen");
		const name = chosen?.textContent.trim() ?? "";

		const hidden = document.createElement("input");
		hidden.type = "hidden";
		hidden.name = input.name;
		hidden.value = input.value;
		hidden.dataset.uiValue = "";
		input.removeAttribute("name");
		input.removeAttribute("inputmode");
		input.value = input.value && name ? name : input.value;
		input.setAttribute("role", "combobox");
		input.setAttribute("aria-autocomplete", "list");
		input.setAttribute("aria-expanded", "false");
		input.setAttribute("autocomplete", "off");
		input.setAttribute("spellcheck", "false");
		const described = (input.getAttribute("aria-describedby") ?? "").split(/\s+/).filter((d) => d && d !== chosen?.id);
		if (described.length) input.setAttribute("aria-describedby", described.join(" "));
		else input.removeAttribute("aria-describedby");
		chosen?.remove();

		const id = input.id || `ui-combobox-${++seq}`;
		input.id = id;
		const popup = document.createElement("div");
		popup.className = "ui-combobox-popup";
		popup.id = `${id}-popup`;
		popup.setAttribute("popover", "manual");
		const list = document.createElement("ul");
		list.className = "ui-combobox-options";
		list.setAttribute("role", "listbox");
		list.id = `${id}-options`;
		const label = input.labels?.[0]?.textContent.replace("*", "").trim();
		if (label) list.setAttribute("aria-label", label);
		// The visible status sits in the popup; what is announced goes to a
		// live region outside it, which is in the tree while the popup is
		// closed.
		const status = document.createElement("p");
		status.className = "ui-combobox-status";
		popup.append(list, status);
		const live = document.createElement("p");
		live.className = "ui-visually-hidden";
		live.setAttribute("role", "status");
		input.setAttribute("aria-controls", list.id);

		const clear = document.createElement("button");
		clear.type = "button";
		clear.className = "ui-combobox-clear";
		clear.tabIndex = -1;
		clear.setAttribute("aria-label", this.#t.clear ?? "Clear");
		clear.textContent = "×";

		this.prepend(hidden);
		input.after(clear, popup);
		this.append(live);
		this.#live = live;
		if (anchors) {
			input.style.anchorName = `--${id}`;
			popup.style.positionAnchor = `--${id}`;
		}

		this.#input = input;
		this.#hidden = hidden;
		this.#popup = popup;
		this.#list = list;
		this.#status = status;
		this.#clear = clear;
		// Text that didn't convert (no name) posts as it is until changed,
		// but isn't a choice to come back to.
		this.#initial = name ? { value: hidden.value, text: name } : { value: "", text: "" };
		this.#committed = { ...this.#initial };
		this.#syncClear();

		input.addEventListener("input", this);
		input.addEventListener("change", this);
		input.addEventListener("keydown", this);
		input.addEventListener("focusout", this);
		popup.addEventListener("pointerdown", (e) => e.preventDefault());
		popup.addEventListener("click", (e) => {
			const li = e.target.closest?.('[role="option"]');
			if (li) this.#choose(li);
		});
		clear.addEventListener("pointerdown", (e) => e.preventDefault());
		clear.addEventListener("click", () => {
			this.#commit("", "");
			input.focus();
		});
		hidden.form?.addEventListener("reset", () => setTimeout(() => this.#commit(this.#initial.value, this.#initial.text), 0));
	}

	disconnectedCallback() {
		this.#abort?.abort();
		clearTimeout(this.#timer);
		if (this.#popup?.matches(":popover-open")) this.#popup.hidePopover();
	}

	get #open() {
		return this.#popup.matches(":popover-open");
	}

	#show() {
		if (this.#open) return;
		this.#popup.showPopover();
		if (!anchors) this.#place();
		this.#input.setAttribute("aria-expanded", "true");
	}

	#hide() {
		if (this.#open) this.#popup.hidePopover();
		this.#input.setAttribute("aria-expanded", "false");
		this.#activate(null);
	}

	// Without anchor positioning: under the input, in the viewport.
	#place() {
		const r = this.#input.getBoundingClientRect();
		const p = this.#popup.style;
		p.position = "fixed";
		p.margin = "0";
		p.inset = "auto"; // the UA's inset: 0 would win over left in RTL
		p.left = `${r.left}px`;
		p.top = `${r.bottom + 4}px`;
		p.width = `${r.width}px`;
	}

	#options() {
		return [...this.#list.querySelectorAll('[role="option"]')];
	}

	#activate(li) {
		for (const o of this.#options()) o.setAttribute("aria-selected", String(o === li));
		if (li) {
			this.#input.setAttribute("aria-activedescendant", li.id);
			li.scrollIntoView({ block: "nearest" });
		} else this.#input.removeAttribute("aria-activedescendant");
	}

	#active() {
		return this.#options().find((o) => o.getAttribute("aria-selected") === "true") ?? null;
	}

	#move(step) {
		const opts = this.#options();
		if (!opts.length) return;
		const i = opts.indexOf(this.#active());
		const next = i < 0 ? (step > 0 ? 0 : opts.length - 1) : Math.min(opts.length - 1, Math.max(0, i + step));
		this.#activate(opts[next]);
	}

	// cancel stops a search waiting or in flight: a choice, a restore or
	// leaving must not be reopened by an answer arriving later.
	#cancel() {
		clearTimeout(this.#timer);
		this.#abort?.abort();
		this.#abort = null;
		this.#pending = false;
		this.#want = 0;
	}

	// commit sets the field's value and tells the form: the hidden input
	// fires input and change, so conditions and filters hear it (§3.4).
	#commit(value, text) {
		const changed = value !== this.#hidden.value;
		this.#cancel();
		this.#list.replaceChildren(); // the next Down searches afresh
		this.#hidden.value = value;
		this.#input.value = text;
		this.#committed = { value, text };
		this.#dirty = false;
		this.#hide();
		this.#syncClear();
		if (changed) {
			this.#hidden.dispatchEvent(new Event("input", { bubbles: true }));
			this.#hidden.dispatchEvent(new Event("change", { bubbles: true }));
		}
	}

	#choose(li) {
		this.#commit(li.dataset.value ?? "", li.textContent.trim());
	}

	#restore() {
		this.#cancel();
		this.#list.replaceChildren();
		this.#input.value = this.#committed.text;
		this.#dirty = false;
		this.#hide();
	}

	#syncClear() {
		this.#clear.hidden = this.#hidden.value === "";
	}

	#schedule() {
		this.#cancel();
		this.#pending = true;
		this.#timer = setTimeout(() => this.#search(this.#input.value), 250);
	}

	async #search(q) {
		this.#abort?.abort();
		const abort = new AbortController();
		this.#abort = abort;
		this.#pending = true;
		const url = new URL(this.dataset.uiSearch, location.href);
		url.searchParams.set("q", q);
		try {
			const res = await fetch(url, {
				headers: { Accept: "text/html", "X-Requested-With": "ui-combobox" },
				credentials: "same-origin",
				signal: abort.signal,
			});
			// A redirect is a session that ended, not an answer (§3.3).
			if (!res.ok || res.redirected) throw new Error(`search: ${res.status}`);
			const tpl = document.createElement("template");
			tpl.innerHTML = await res.text();
			const root = tpl.content.firstElementChild;
			if (!root?.matches("ul[data-ui-suggestions]")) throw new Error("search: not suggestions");
			if (abort.signal.aborted || this.#abort !== abort) return;
			this.#activate(null);
			// Options are built here, from values and texts: nothing in the
			// answer's markup reaches the page.
			const items = [...root.querySelectorAll(":scope > li[data-value]")];
			this.#list.replaceChildren(
				...items.map((src, i) => {
					const li = document.createElement("li");
					li.setAttribute("role", "option");
					li.id = `${this.#input.id}-opt-${i}`;
					li.dataset.value = src.dataset.value;
					li.setAttribute("aria-selected", "false");
					li.textContent = src.textContent.trim();
					return li;
				}),
			);
			let text = items.length ? fmt(this.#t.results ?? "%d", items.length) : (this.#t.none ?? "");
			if (root.hasAttribute("data-ui-more")) text += ` ${this.#t.more ?? ""}`;
			this.#status.textContent = text;
			this.#list.hidden = items.length === 0;
		} catch (err) {
			if (abort.signal.aborted || this.#abort !== abort) return;
			this.#activate(null);
			this.#list.replaceChildren();
			this.#list.hidden = true;
			this.#status.textContent = this.#t.failed ?? "";
		}
		this.#pending = false;
		this.#abort = null;
		if (document.activeElement !== this.#input) return;
		this.#show();
		say(this.#live, this.#status.textContent);
		if (this.#want) this.#move(this.#want);
		this.#want = 0;
	}

	handleEvent(e) {
		switch (e.type) {
			case "input":
				// The visible text isn't the field's value: its events stop here.
				e.stopPropagation();
				this.#dirty = true;
				this.#activate(null);
				this.#schedule();
				break;
			case "change":
				e.stopPropagation();
				break;
			case "focusout":
				// A window losing focus keeps the input focused: nothing ends.
				if (this.contains(e.relatedTarget) || document.activeElement === this.#input) return;
				this.#cancel();
				if (this.#input.value.trim() === "") this.#commit("", "");
				else if (this.#dirty) this.#restore();
				else this.#hide();
				break;
			case "keydown":
				this.#key(e);
				break;
		}
	}

	#key(e) {
		const stop = () => {
			e.preventDefault();
			e.stopPropagation();
		};
		switch (e.key) {
			case "ArrowDown":
				stop();
				if (e.altKey || !this.#open) {
					if (!this.#options().length) {
						this.#want = e.altKey ? 0 : 1;
						return void this.#search(this.#input.value);
					}
					this.#show();
					if (e.altKey) return;
				}
				this.#move(1);
				break;
			case "ArrowUp":
				stop();
				if (e.altKey) return this.#hide();
				if (!this.#open) {
					if (!this.#options().length) {
						this.#want = -1;
						return void this.#search(this.#input.value);
					}
					this.#show();
				}
				this.#move(-1);
				break;
			case "Enter": {
				const li = this.#open ? this.#active() : null;
				if (li) {
					stop();
					this.#choose(li);
				} else if (this.#dirty || this.#pending) {
					// Uncommitted text never submits (§3.4).
					stop();
				}
				break;
			}
			case "Escape":
				if (this.#open) {
					stop();
					this.#hide();
				} else if (this.#dirty) {
					stop();
					this.#restore();
				}
				break;
		}
	}
}

if (!customElements.get("ui-combobox")) customElements.define("ui-combobox", UICombobox);

// <ui-repeat> (§5): clones forms' template row.
const idRefs = ["for", "list", "form", "commandfor", "popovertarget"];
const idLists = ["aria-controls", "aria-describedby", "aria-labelledby", "aria-activedescendant"];

// elementsIn yields every element under root and, recursively, inside
// each <template>'s content, which querySelectorAll doesn't reach.
function* elementsIn(root, deep) {
	for (const el of root.querySelectorAll("*")) {
		yield [el, deep];
		if (el instanceof HTMLTemplateElement) yield* elementsIn(el.content, true);
	}
}

// rewrite replaces the list's own segment (from → to) at the start of
// each path, and gives this level's ids a fresh suffix (§5.2).
function rewrite(root, from, to, suffix) {
	const ids = new Map();
	for (const [el, deep] of elementsIn(root, false)) {
		for (const a of ["name", "data-ui-path"]) {
			const v = el.getAttribute(a);
			if (v?.startsWith(from)) el.setAttribute(a, to + v.slice(from.length));
		}
		const when = el.getAttribute("data-ui-when");
		if (when) el.setAttribute("data-ui-when", when.replaceAll(`"${from}`, `"${to}`));
		if (!deep && el.id) {
			const id = `${el.id}-${suffix}`;
			ids.set(el.id, id);
			el.id = id;
		}
	}
	// References are rewritten at every depth: a nested template may point
	// at an id of this row.
	for (const [el] of elementsIn(root, false)) {
		for (const a of idRefs) {
			const v = el.getAttribute(a);
			if (v && ids.has(v)) el.setAttribute(a, ids.get(v));
		}
		for (const a of idLists) {
			const v = el.getAttribute(a);
			if (v) el.setAttribute(a, v.split(/\s+/).map((t) => ids.get(t) ?? t).join(" "));
		}
	}
}

class UIRepeat extends HTMLElement {
	#up = false;
	#t = {};

	connectedCallback() {
		if (this.#up) return;
		this.#up = true;
		this.#t = textsOf(this);
		this.addEventListener("click", this);
		for (const b of this.#own("[data-ui-add], [data-ui-remove]")) b.hidden = false;
		this.#sync();
	}

	// own is what belongs to this list, not to a list nested in it.
	#own(sel) {
		return [...this.querySelectorAll(sel)].filter((e) => e.closest("ui-repeat") === this);
	}

	#items() {
		return [...(this.querySelector(":scope > ol")?.children ?? [])];
	}

	#status() {
		return this.querySelector(":scope > [role='status']");
	}

	handleEvent(e) {
		const b = e.target.closest?.("button");
		if (!b || b.closest("ui-repeat") !== this || b.getAttribute("aria-disabled") === "true") return;
		if (b.hasAttribute("data-ui-add")) this.add();
		else if (b.hasAttribute("data-ui-remove")) this.remove(b.closest("li"));
	}

	add() {
		const tpl = this.querySelector(":scope > template");
		const ol = this.querySelector(":scope > ol");
		if (!tpl || !ol) return;
		const n = Number(this.dataset.uiNext ?? this.#items().length);
		const path = this.dataset.uiPath;
		const frag = tpl.content.cloneNode(true);
		rewrite(frag, `${path}[__i__]`, `${path}[${n}]`, `r${++seq}`);
		this.dataset.uiNext = String(n + 1);
		const li = frag.firstElementChild;
		ol.append(frag);
		const own = li.querySelector(":scope > fieldset > [data-ui-remove]");
		if (own) own.hidden = false;
		this.#renumber();
		this.#sync();
		// Conditions in the clone first (the ui script re-evaluates its form
		// on input), so focus never lands on a control about to hide.
		li.querySelector('[name]:not([type="file"])')?.dispatchEvent(new Event("input", { bubbles: true }));
		firstControl(li)?.focus();
		say(this.#status(), fmt(this.#t.added, this.#items().length));
	}

	remove(li) {
		const items = this.#items();
		const i = items.indexOf(li);
		if (i < 0) return;
		li.remove();
		this.#renumber();
		this.#sync();
		const rest = this.#items();
		const next = rest[i] ?? rest[i - 1];
		const target = (next && firstControl(next)) || this.#own("[data-ui-add]")[0];
		target?.focus();
		say(this.#status(), fmt(this.#t.removed, i + 1));
	}

	#renumber() {
		this.#items().forEach((li, i) => {
			const legend = li.querySelector(":scope > fieldset > legend");
			if (legend) legend.textContent = fmt(this.#t.item, i + 1);
			const rm = li.querySelector(":scope > fieldset > [data-ui-remove]");
			if (rm) rm.firstChild.textContent = fmt(this.#t.remove, i + 1);
		});
	}

	// sync makes Add and Remove unavailable at the bounds, with a reason.
	#sync() {
		const n = this.#items().length;
		const min = Number(this.dataset.uiMin) || 0;
		const max = Number(this.dataset.uiMax) || Infinity; // absent or 0: no maximum
		const set = (b, off, reason) => {
			let why = b.querySelector(":scope > .ui-visually-hidden");
			if (off) {
				b.setAttribute("aria-disabled", "true");
				if (!why) {
					why = document.createElement("span");
					why.className = "ui-visually-hidden";
					b.append(why);
				}
				why.textContent = ` (${reason})`;
			} else {
				b.removeAttribute("aria-disabled");
				why?.remove();
			}
		};
		for (const b of this.#own("[data-ui-add]")) set(b, n >= max, fmt(this.#t.atMost, max));
		for (const li of this.#items()) {
			const rm = li.querySelector(":scope > fieldset > [data-ui-remove]");
			if (rm) set(rm, n <= min, fmt(this.#t.atLeast, min));
		}
	}
}

if (!customElements.get("ui-repeat")) customElements.define("ui-repeat", UIRepeat);

// Uploads (§4.3): keep cards, chosen files, checks before sending,
// progress.
const size = (n) => (n >= 1 << 20 ? `${(n / (1 << 20)).toFixed(1)} MB` : n >= 1024 ? `${Math.round(n / 1024)} KB` : `${n} B`);

function setUpUpload(field) {
	if (field.dataset.uiUp !== undefined) return;
	field.dataset.uiUp = "";
	const t = textsOf(field);
	for (const li of field.querySelectorAll(".ui-upload-file")) {
		const box = li.querySelector('input[type="checkbox"]');
		const name = li.querySelector("a")?.textContent ?? li.querySelector("label")?.textContent.trim() ?? "";
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

function chosen(input) {
	// Only an upload field: it carries the texts its checks need.
	const field = input.closest(".ui-upload");
	if (!field) return;
	const list = field.querySelector(".ui-upload-chosen");
	const t = textsOf(field);
	const max = Number(input.dataset.uiMaxSize || 0);
	const accept = (input.accept || "").split(",").map((s) => s.trim().toLowerCase()).filter(Boolean);
	const fits = (f) => {
		const type = f.type.toLowerCase();
		// No type from the system: the server sniffs the bytes, so it decides.
		if (!type && !accept.some((a) => a.startsWith("."))) return true;
		const name = f.name.toLowerCase();
		return accept.some((a) => (a.endsWith("/*") ? type.startsWith(a.slice(0, -1)) : a.startsWith(".") ? name.endsWith(a) : type === a));
	};
	let problem = "";
	for (const f of input.files) {
		if (max && f.size > max) problem ||= fmt(t.tooLarge, f.name, size(max));
		if (accept.length && !fits(f)) problem ||= fmt(t.wrongType, f.name);
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
			input.setAttribute("aria-describedby", [...tokens(), err.id].join(" "));
		}
		err.textContent = problem;
		input.setAttribute("aria-invalid", "true");
	} else if (err) {
		input.setAttribute("aria-describedby", tokens().filter((d) => d !== err.id).join(" "));
		err.remove();
		// The server's own error, if any, still stands.
		if (!field.querySelector(".ui-field-error")) input.removeAttribute("aria-invalid");
	}
	if (!list) return;
	list.replaceChildren(
		...[...input.files].map((f, i) => {
			const li = document.createElement("li");
			li.textContent = `${f.name} (${size(f.size)}) `;
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

document.addEventListener("change", (e) => {
	if (e.target instanceof HTMLInputElement && e.target.type === "file") chosen(e.target);
});

// ui's forms are novalidate, so a custom validity blocks nothing by
// itself: an upload that failed its check refuses the submit here, before
// htmx or the browser sends it, and takes focus.
document.addEventListener(
	"submit",
	(e) => {
		const bad = [...(e.target.querySelectorAll?.('.ui-upload input[type="file"]') ?? [])].find((i) => i.validity.customError);
		if (!bad) return;
		e.preventDefault();
		e.stopImmediatePropagation();
		bad.focus();
	},
	true,
);

for (const ev of ["dragenter", "dragover"]) document.addEventListener(ev, (e) => e.target.closest?.(".ui-upload")?.setAttribute("data-ui-dragging", ""));
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
// A body past the route's limit is a 413, which htmx doesn't swap: the
// form stays as it is, chosen files included, and its upload fields say
// why (§4.5, found when the spike capped its route).
function bodyTooLarge(e) {
	const form = e.detail.elt?.closest?.("form") ?? e.detail.elt;
	const fields = [...(form?.querySelectorAll?.(".ui-upload") ?? [])].filter((f) => f.querySelector('input[type="file"]')?.files.length);
	for (const field of fields) {
		const input = field.querySelector('input[type="file"]');
		const t = textsOf(field);
		let err = field.querySelector(".ui-upload-check");
		if (!err) {
			err = document.createElement("p");
			err.className = "ui-field-error ui-upload-check";
			err.id = `${input.id}-check`;
			input.after(err);
			input.setAttribute("aria-describedby", `${input.getAttribute("aria-describedby") ?? ""} ${err.id}`.trim());
		}
		err.textContent = t.bodyTooLarge ?? "";
		input.setAttribute("aria-invalid", "true");
	}
	fields[0]?.querySelector('input[type="file"]')?.focus();
}
// Only a 413: a send error is as likely a network gone (review 0021).
document.addEventListener("htmx:responseError", (e) => e.detail.xhr?.status === 413 && bodyTooLarge(e));

document.addEventListener("htmx:afterRequest", (e) => {
	for (const bar of (e.detail.elt?.closest?.("form") ?? document).querySelectorAll(".ui-upload-progress")) bar.hidden = true;
});

const setUp = (root) => root.querySelectorAll?.(".ui-upload").forEach(setUpUpload);
setUp(document);
document.addEventListener("htmx:afterSwap", (e) => setUp(e.target));
