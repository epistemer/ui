// <ui-repeat> (rich-inputs-interface.md §6, rich-inputs-design.md §5.2–§5.3):
// clones forms' template row.

const textsOf = (el) => {
	try {
		return JSON.parse(el.dataset.uiTexts || "{}");
	} catch {
		return {};
	}
};

// fmt fills the single %s or %d verb in a single pass, so a verb inside the
// argument itself stays as it is (rich-inputs-interface.md §3.1).
export function fmt(s, arg) {
	return String(s ?? "").replace(/%[sd]/, () => String(arg ?? ""));
}

// rewritePath replaces the list's own segment (from → to) at the start of a
// field or nested list path, leaving a nested list's own [__i__] intact.
export function rewritePath(v, from, to) {
	return v && v.startsWith(from) ? to + v.slice(from.length) : v;
}

// rewriteWhen replaces the list's own segment (from → to) in each field name
// of a JSON-encoded forms condition.
export function rewriteWhen(when, from, to) {
	return when ? when.split(`"${from}`).join(`"${to}`) : when;
}

const say = (status, text) => {
	if (!status) return;
	status.textContent = "";
	setTimeout(() => (status.textContent = text), 0);
};

const visible = (el) => !el.closest("[hidden]") && !el.closest("template");
const focusable =
	'input:not([type="hidden"]):not(:disabled), select:not(:disabled), textarea:not(:disabled), button:not(:disabled):not([hidden]):not([tabindex="-1"]):not([aria-disabled="true"])';
const firstControl = (root) => [...root.querySelectorAll(focusable)].find(visible);

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
			if (v && v.startsWith(from)) el.setAttribute(a, rewritePath(v, from, to));
		}
		const when = el.getAttribute("data-ui-when");
		if (when) el.setAttribute("data-ui-when", rewriteWhen(when, from, to));
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

let seq = 0;

if (typeof HTMLElement !== "undefined") {
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
			if (li) globalThis.htmx?.process(li);
			const own = li?.querySelector(":scope > fieldset > [data-ui-remove]");
			if (own) own.hidden = false;
			this.#renumber();
			this.#sync();
			// Conditions in the clone first (the ui script re-evaluates its form
			// on input), so focus never lands on a control about to hide.
			li?.querySelector('[name]:not([type="file"])')?.dispatchEvent(new Event("input", { bubbles: true }));
			if (li) firstControl(li)?.focus();
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
				if (rm) {
					if (rm.firstChild?.nodeType === Node.TEXT_NODE) rm.firstChild.textContent = fmt(this.#t.remove, i + 1);
					else rm.prepend(fmt(this.#t.remove, i + 1));
				}
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
}
