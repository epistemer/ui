// <ui-combobox> (rich-inputs-interface.md §6, rich-inputs-design.md §3.3–§3.4):
// the APG editable combobox with list autocomplete, searching through htmx.

const textsOf = (el) => {
	try {
		return JSON.parse(el.dataset.uiTexts || "{}");
	} catch {
		return {};
	}
};

// fmt fills the single %s or %d verb in a single pass, so a verb inside the
// argument itself stays as it is (rich-inputs-interface.md §3.1).
const fmt = (s, arg) => String(s ?? "").replace(/%[sd]/, () => String(arg ?? ""));

const say = (status, text) => {
	if (!status) return;
	// A change of text is what a live region announces; clear first so the
	// same text twice is still heard.
	status.textContent = "";
	// A task, not a frame: a hidden page runs no frames.
	setTimeout(() => (status.textContent = text), 0);
};

// The same test as the CSS that positions the popup (components.css).
const anchors = () =>
	typeof CSS !== "undefined" &&
	CSS.supports("position-area: block-end") &&
	!document.documentElement.hasAttribute("data-ui-no-anchor");

let seq = 0;

if (typeof HTMLElement !== "undefined") {
// requester is the element that made an htmx request. In htmx 2.0.11 the
// swap events' detail.elt is the swap target (the listbox), so the
// requesting element comes from requestConfig.
const requester = (e) => e.detail?.requestConfig?.elt ?? e.detail?.elt;

	class UICombobox extends HTMLElement {
		#up = false;
		#t = {};
		#input;
		#hidden;
		#popup;
		#list;
		#status;
		#clear;
		#live;
		#committed = { value: "", text: "" };
		#initial = { value: "", text: "" };
		#dirty = false;
		#timer = 0;
		#pending = false;
		#refused = false;
		#want = 0;

		connectedCallback() {
			// Without a search link (the gate refused it) or without htmx it
			// stays the text input of the id (§3.3, R7).
			if (this.#up || !this.dataset.uiSearch || !globalThis.htmx) return;
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
			const described = (input.getAttribute("aria-describedby") ?? "")
				.split(/\s+/)
				.filter((d) => d && d !== chosen?.id);
			if (described.length) input.setAttribute("aria-describedby", described.join(" "));
			else input.removeAttribute("aria-describedby");
			chosen?.remove();

			const id = input.id || `ui-combobox-${++seq}`;
			input.id = id;

			let clear = this.querySelector(":scope > .ui-combobox-clear");
			if (!clear) {
				clear = document.createElement("button");
				clear.type = "button";
				clear.className = "ui-combobox-clear";
				clear.tabIndex = -1;
				clear.textContent = "×";
				input.after(clear);
			}
			clear.tabIndex = -1;
			clear.setAttribute("aria-label", this.#t.clear ?? clear.getAttribute("aria-label") ?? "Clear");

			let popup = this.querySelector(":scope > .ui-combobox-popup");
			if (!popup) {
				popup = document.createElement("div");
				popup.className = "ui-combobox-popup";
				popup.setAttribute("popover", "manual");
				clear.after(popup);
			}
			let list = popup.querySelector(".ui-combobox-options");
			if (!list) {
				list = document.createElement("ul");
				list.className = "ui-combobox-options";
				list.setAttribute("role", "listbox");
				popup.prepend(list);
			}
			if (!list.id) list.id = `${id}-options`;
			let status = popup.querySelector(".ui-combobox-status");
			if (!status) {
				status = document.createElement("p");
				status.className = "ui-combobox-status";
				popup.append(status);
			}
			let live = this.querySelector(':scope > .ui-visually-hidden[role="status"]');
			if (!live) {
				live = document.createElement("p");
				live.className = "ui-visually-hidden";
				live.setAttribute("role", "status");
				this.append(live);
			}
			input.setAttribute("aria-controls", list.id);

			this.prepend(hidden);
			if (anchors()) {
				input.style.anchorName = `--${id}`;
				popup.style.positionAnchor = `--${id}`;
			}

			this.#input = input;
			this.#hidden = hidden;
			this.#popup = popup;
			this.#list = list;
			this.#status = status;
			this.#clear = clear;
			this.#live = live;
			// Text that didn't convert (no name) posts as it is until changed,
			// but isn't a choice to come back to.
			this.#initial = name ? { value: hidden.value, text: name } : { value: "", text: "" };
			this.#committed = { ...this.#initial };
			this.#syncClear();

			input.addEventListener("input", this);
			input.addEventListener("change", this);
			input.addEventListener("keydown", this);
			input.addEventListener("focusout", this);
			this.addEventListener("htmx:configRequest", this);
			this.addEventListener("htmx:beforeSwap", this);
			this.addEventListener("htmx:afterSwap", this);
			this.addEventListener("htmx:responseError", this);
			this.addEventListener("htmx:sendError", this);
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
			hidden.form?.addEventListener("reset", () =>
				setTimeout(() => this.#commit(this.#initial.value, this.#initial.text), 0),
			);
		}

		disconnectedCallback() {
			this.#cancel();
			if (this.#popup?.matches(":popover-open")) this.#popup.hidePopover();
		}

		get #open() {
			return this.#popup.matches(":popover-open");
		}

		#show() {
			if (this.#open) return;
			this.#popup.showPopover();
			if (!anchors()) this.#place();
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
				li.scrollIntoView?.({ block: "nearest" });
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
			this.#timer = 0;
			const wasPending = this.#pending;
			this.#pending = false;
			this.#want = 0;
			this.#refused = true;
			if (wasPending) {
				const ev = () =>
					new CustomEvent("htmx:abort", { bubbles: true, cancelable: true, detail: { elt: this } });
				this.dispatchEvent(ev());
				if (!this.isConnected) document.body?.dispatchEvent(ev());
			}
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
			this.#timer = setTimeout(() => {
				this.#timer = 0;
				this.#search();
			}, 250);
		}

		#search() {
			clearTimeout(this.#timer);
			this.#timer = 0;
			this.#pending = true;
			this.#refused = false;
			this.dispatchEvent(new Event("ui-search"));
		}

		#fail() {
			this.#pending = false;
			this.#want = 0;
			this.#activate(null);
			this.#list.replaceChildren();
			this.#list.hidden = true;
			this.#status.textContent = this.#t.failed ?? "";
			if (document.activeElement !== this.#input) return;
			this.#show();
			say(this.#live, this.#status.textContent);
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
				case "htmx:configRequest": {
					if (requester(e) && requester(e) !== this) return;
					const url = new URL(e.detail.path, location.href);
					url.searchParams.delete("q");
					e.detail.path = url.pathname + url.search;
					if (e.detail.parameters) e.detail.parameters.q = this.#input.value;
					break;
				}
				case "htmx:beforeSwap": {
					if (requester(e) && requester(e) !== this) return;
					e.stopPropagation();
					const xhr = e.detail?.xhr;
					const want = new URL(this.dataset.uiSearch, location.href);
					let got = null;
					try {
						if (xhr?.responseURL) got = new URL(xhr.responseURL, location.href);
					} catch {}
					const body = String(e.detail?.serverResponse ?? xhr?.responseText ?? "");
					const ok =
						xhr?.status === 200 &&
						got !== null &&
						got.origin === want.origin &&
						got.pathname === want.pathname &&
						body.includes("data-ui-suggestions");
					if (!ok) {
						e.detail.shouldSwap = false;
						this.#refused = true;
						this.#fail();
						return;
					}
					this.#refused = false;
					break;
				}
				case "htmx:afterSwap": {
					if (requester(e) && requester(e) !== this) return;
					if (this.#refused) return;
					this.#activate(null);
					const more = this.#list.querySelector("li[data-ui-more]");
					more?.remove();
					const items = this.#options();
					items.forEach((li, i) => {
						li.id = `${this.#input.id}-opt-${i}`;
					});
					let text = items.length ? fmt(this.#t.results ?? "%d", items.length) : (this.#t.none ?? "");
					if (more && this.#t.more) text += ` ${this.#t.more}`;
					this.#status.textContent = text;
					this.#list.hidden = items.length === 0;
					this.#pending = false;
					const want = this.#want;
					this.#want = 0;
					if (document.activeElement !== this.#input) return;
					this.#show();
					say(this.#live, this.#status.textContent);
					if (want) this.#move(want);
					break;
				}
				case "htmx:responseError":
				case "htmx:sendError":
					if (requester(e) && requester(e) !== this) return;
					this.#fail();
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
						if (this.#timer || !this.#options().length) {
							this.#want = e.altKey ? 0 : 1;
							return void this.#search();
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
						if (this.#timer || !this.#options().length) {
							this.#want = -1;
							return void this.#search();
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
}
