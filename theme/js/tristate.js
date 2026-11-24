// <ui-tristate> (confab-1-interface.md §3.3): upgrades a Bool field's
// Yes/No/empty <select> into a three-position switch (No, no answer, Yes),
// keeping the value in the select so form submission and conditions work as
// they do without script.

const order = ["no", "unset", "yes"];
const stateToValue = { no: "false", unset: "", yes: "true" };
const stateToAria = { no: "false", unset: "mixed", yes: "true" };
// Template marks order: Check (yes), Minus (unset), X (no).
const stateToMark = { yes: 0, unset: 1, no: 2 };

export function stateOf(value) {
	if (value === "true") return "yes";
	if (value === "false") return "no";
	return "unset";
}

// Space cycles no answer -> Yes -> No -> no answer (Q2).
export function nextState(state) {
	if (state === "unset") return "yes";
	if (state === "yes") return "no";
	return "unset";
}

// Arrow keys move the knob towards the arrow's side and stop at the ends;
// in RTL, start (no) is on the right and end (yes) is on the left.
export function arrowState(state, key, rtl) {
	let step = 0;
	if (key === "ArrowRight") step = rtl ? -1 : 1;
	else if (key === "ArrowLeft") step = rtl ? 1 : -1;
	else return null;
	const i = order.indexOf(state);
	if (i < 0) return null;
	return order[Math.min(order.length - 1, Math.max(0, i + step))];
}

// Pointer position across the track's inline width: start third is No,
// middle is no answer, end third is Yes (mirrored in RTL).
export function thirdState(fraction, rtl) {
	const pos = rtl ? 1 - fraction : fraction;
	if (pos < 1 / 3) return "no";
	if (pos < 2 / 3) return "unset";
	return "yes";
}

const say = (status, text) => {
	if (!status) return;
	// Clear first so repeating a word is still announced; a task, not a
	// frame, because a hidden page runs no frames.
	status.textContent = "";
	setTimeout(() => (status.textContent = text), 0);
};

let seq = 0;

if (typeof HTMLElement !== "undefined") {
	class UITristate extends HTMLElement {
		#select = null;
		#track = null;
		#knob = null;
		#state = null;
		#live = null;
		#marks = [];
		#words = { no: "", unset: "", yes: "" };
		#mo = null;
		#ac = null;
		#fromLabel = false;

		connectedCallback() {
			this.#teardown();
			const select = this.querySelector(":scope > select");
			if (!select) return;

			// Upgrading from an htmx history snapshot finds the track and
			// live region already in the DOM, with required moved to the
			// track and the label's for pointing at it. Read those before
			// replacing them so re-upgrade is idempotent and loses nothing.
			const oldTrack = this.querySelector(":scope > .ui-tristate-track");
			const required = select.hasAttribute("required") || oldTrack?.getAttribute("aria-required") === "true";
			const label = this.#findLabel(select.id) ?? (oldTrack?.id ? this.#findLabel(oldTrack.id) : null);

			for (const el of this.querySelectorAll(':scope > .ui-tristate-track, :scope > [role="status"]')) {
				el.remove();
			}

			const tpl = this.querySelector(":scope > template.ui-tristate-marks");
			this.#marks = tpl?.content ? [...tpl.content.children] : [];

			const optText = (val) => {
				for (const o of select.options ?? select.querySelectorAll?.("option") ?? []) {
					if (o.value === val) return o.textContent.trim();
				}
				return "";
			};
			this.#words = {
				yes: optText("true"),
				no: optText("false"),
				unset: (this.dataset.uiUnset ?? "").trim(),
			};

			const baseId = select.id || `ui-tristate-${++seq}`;
			const trackId = `${baseId}-track`;
			const stateId = `${baseId}-state`;

			const track = document.createElement("button");
			track.type = "button";
			track.className = "ui-tristate-track";
			track.id = trackId;
			track.setAttribute("role", "checkbox");

			const described = (select.getAttribute("aria-describedby") ?? "")
				.split(/\s+/)
				.filter((d) => d && d !== stateId);
			described.push(stateId);
			track.setAttribute("aria-describedby", described.join(" "));

			if (select.hasAttribute("aria-invalid")) {
				track.setAttribute("aria-invalid", select.getAttribute("aria-invalid"));
			}
			if (required) {
				track.setAttribute("aria-required", "true");
				// A browser can't show its built-in tooltip on a hidden
				// control; the server's ErrRequired answers instead.
				select.removeAttribute("required");
			}

			const knob = document.createElement("span");
			knob.className = "ui-tristate-knob";
			knob.setAttribute("aria-hidden", "true");

			const state = document.createElement("span");
			state.className = "ui-tristate-state";
			state.id = stateId;

			track.append(knob, state);

			const live = document.createElement("span");
			live.className = "ui-visually-hidden";
			live.setAttribute("role", "status");

			select.classList.add("ui-visually-hidden");
			select.tabIndex = -1;
			select.setAttribute("aria-hidden", "true");
			if (label) label.setAttribute("for", track.id);

			this.append(track, live);

			this.#select = select;
			this.#track = track;
			this.#knob = knob;
			this.#state = state;
			this.#live = live;

			this.#ac = new AbortController();
			const { signal } = this.#ac;

			track.addEventListener("click", (e) => this.#onClick(e), { signal });
			track.addEventListener("keydown", (e) => this.#onKey(e), { signal });
			if (label) {
				label.addEventListener(
					"click",
					() => {
						this.#fromLabel = true;
						const clear = () => {
							this.#fromLabel = false;
						};
						if (typeof queueMicrotask === "function") queueMicrotask(clear);
						else setTimeout(clear, 0);
					},
					{ signal },
				);
			}
			select.addEventListener("input", () => this.#sync(), { signal });
			select.addEventListener("change", () => this.#sync(), { signal });
			if (select.form) {
				// reset fires before the form's values are restored.
				select.form.addEventListener(
					"reset",
					() => setTimeout(() => this.isConnected && this.#sync(), 0),
					{ signal },
				);
			}
			if (typeof MutationObserver !== "undefined") {
				this.#mo = new MutationObserver(() => this.#syncDisabled());
				this.#mo.observe(select, { attributes: true, attributeFilter: ["disabled"] });
			}

			this.#sync();
		}

		disconnectedCallback() {
			this.#teardown();
		}

		#teardown() {
			this.#mo?.disconnect();
			this.#mo = null;
			this.#ac?.abort();
			this.#ac = null;
			this.#fromLabel = false;
		}

		#findLabel(id) {
			if (!id) return null;
			const root = this.getRootNode?.() ?? document;
			if (!root?.querySelector) return null;
			const esc = typeof CSS !== "undefined" && typeof CSS.escape === "function" ? CSS.escape(id) : id;
			return root.querySelector(`label[for="${esc}"]`);
		}

		#rtl() {
			return typeof getComputedStyle === "function" && getComputedStyle(this).direction === "rtl";
		}

		#current() {
			return stateOf(this.#select.value);
		}

		#syncDisabled() {
			if (this.#track && this.#select) {
				this.#track.disabled = Boolean(this.#select.disabled);
			}
		}

		#sync() {
			if (!this.#select || !this.#track) return;
			const state = this.#current();
			this.#track.dataset.state = state;
			this.#track.setAttribute("aria-checked", stateToAria[state]);
			const mark = this.#marks[stateToMark[state]];
			if (mark) this.#knob.replaceChildren(mark.cloneNode(true));
			else this.#knob.replaceChildren();
			this.#state.textContent = this.#words[state] ?? "";
			this.#syncDisabled();
		}

		#set(next) {
			const cur = this.#current();
			if (!next || next === cur) return;
			this.#select.value = stateToValue[next];
			this.#sync();
			say(this.#live, this.#words[next] ?? "");
			this.#select.dispatchEvent(new Event("input", { bubbles: true }));
			this.#select.dispatchEvent(new Event("change", { bubbles: true }));
		}

		#onClick(e) {
			if (this.#select.disabled || this.#track.disabled) return;
			if (this.#fromLabel) {
				this.#fromLabel = false;
				this.#set(nextState(this.#current()));
				return;
			}
			const rect = this.#track.getBoundingClientRect?.();
			if (
				e.detail > 0 &&
				rect &&
				rect.width > 0 &&
				e.clientX >= rect.left &&
				e.clientX <= rect.right &&
				e.clientY >= rect.top &&
				e.clientY <= rect.bottom
			) {
				this.#set(thirdState((e.clientX - rect.left) / rect.width, this.#rtl()));
			} else {
				this.#set(nextState(this.#current()));
			}
		}

		#onKey(e) {
			if (this.#select.disabled || this.#track.disabled) return;
			const cur = this.#current();
			let target = null;
			switch (e.key) {
				case " ":
					target = nextState(cur);
					break;
				case "ArrowLeft":
				case "ArrowRight":
					target = arrowState(cur, e.key, this.#rtl());
					break;
				case "Home":
					target = "no";
					break;
				case "End":
					target = "yes";
					break;
				case "Delete":
				case "Backspace":
					target = "unset";
					break;
				default:
					return;
			}
			e.preventDefault();
			this.#set(target);
		}
	}

	if (!customElements.get("ui-tristate")) customElements.define("ui-tristate", UITristate);
}
