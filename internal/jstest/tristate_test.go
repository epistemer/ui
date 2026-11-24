package jstest

import (
	"os"
	"testing"

	"github.com/dop251/goja"
)

func TestTristatePureHelpers(t *testing.T) {
	vm := load(t, "../../theme/js/tristate.js")
	res, err := vm.RunString(`
		[
			stateOf(""),
			stateOf("true"),
			stateOf("false"),
			stateOf("other"),
			nextState("unset"),
			nextState("yes"),
			nextState("no"),
			arrowState("no", "ArrowRight", false),
			arrowState("unset", "ArrowRight", false),
			arrowState("yes", "ArrowRight", false),
			arrowState("yes", "ArrowLeft", false),
			arrowState("no", "ArrowLeft", false),
			arrowState("no", "ArrowLeft", true),
			arrowState("unset", "ArrowLeft", true),
			arrowState("yes", "ArrowLeft", true),
			arrowState("yes", "ArrowRight", true),
			arrowState("no", "ArrowRight", true),
			thirdState(0.1, false),
			thirdState(0.5, false),
			thirdState(0.9, false),
			thirdState(0.1, true),
			thirdState(0.5, true),
			thirdState(0.9, true),
		]
	`)
	if err != nil {
		t.Fatal(err)
	}
	got := res.Export().([]any)
	want := []string{
		"unset", "yes", "no", "unset",
		"yes", "no", "unset",
		"unset", "yes", "yes", "unset", "no",
		"unset", "yes", "yes", "unset", "no",
		"no", "unset", "yes",
		"yes", "unset", "no",
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("case %d: got %v, want %q", i, got[i], w)
		}
	}
}

// TestTristateElement drives <ui-tristate> with a minimal fake DOM in goja:
// upgrade, label for move, required -> aria-required, aria-describedby + state
// word, aria-invalid, disabled observer, pointer thirds (LTR and RTL), label
// click, keyboard, select events, live region, form reset, idempotent
// re-upgrade from a snapshot, and disconnect cleanup.
func TestTristateElement(t *testing.T) {
	src, err := os.ReadFile("../../theme/js/tristate.js")
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	harness := `
	const timers = [];
	const microtasks = [];
	globalThis.setTimeout = (fn) => { timers.push(fn); return timers.length; };
	globalThis.queueMicrotask = (fn) => { microtasks.push(fn); };
	function flushMicrotasks() { while (microtasks.length) microtasks.shift()(); }
	function flushTimers() { while (timers.length) timers.shift()(); }

	globalThis.Event = class Event {
		constructor(type, init = {}) {
			this.type = type;
			this.bubbles = Boolean(init.bubbles);
			this.defaultPrevented = false;
		}
		preventDefault() { this.defaultPrevented = true; }
	};

	globalThis.AbortController = class AbortController {
		constructor() {
			this.signal = { aborted: false, _cbs: [] };
		}
		abort() {
			if (this.signal.aborted) return;
			this.signal.aborted = true;
			for (const cb of this.signal._cbs) cb();
			this.signal._cbs = [];
		}
	};

	globalThis.CSS = {
		escape: (s) => String(s).replace(/([^a-zA-Z0-9_-])/g, "\\$1"),
	};

	const observers = [];
	globalThis.MutationObserver = class MutationObserver {
		constructor(cb) { this._cb = cb; this._target = null; }
		observe(target) { this._target = target; observers.push(this); }
		disconnect() {
			const i = observers.indexOf(this);
			if (i >= 0) observers.splice(i, 1);
			this._target = null;
		}
		_notify() { this._cb([]); }
	};
	function notifyAttr(target) {
		for (const o of [...observers]) if (o._target === target) o._notify();
	}

	class FakeElement {
		constructor(tag) {
			this.tagName = tag.toUpperCase();
			this.attrs = new Map();
			this.children = [];
			this.parentNode = null;
			this.textContent = "";
			this.dataset = {};
			this.isConnected = true;
			this._listeners = new Map();
			this._disabled = false;
			this._value = "";
			this.tabIndex = 0;
			this.classList = {
				add: (...cls) => {
					const cur = new Set((this.attrs.get("class") || "").split(/\s+/).filter(Boolean));
					for (const c of cls) cur.add(c);
					this.attrs.set("class", [...cur].join(" "));
				},
				contains: (c) => (this.attrs.get("class") || "").split(/\s+/).includes(c),
			};
		}
		get id() { return this.attrs.get("id") || ""; }
		set id(v) { this.attrs.set("id", String(v)); }
		get className() { return this.attrs.get("class") || ""; }
		set className(v) { this.attrs.set("class", String(v)); }
		get disabled() { return this._disabled; }
		set disabled(v) {
			this._disabled = Boolean(v);
			if (this._disabled) this.attrs.set("disabled", "");
			else this.attrs.delete("disabled");
			notifyAttr(this);
		}
		get value() { return this._value; }
		set value(v) { this._value = String(v); }
		getAttribute(k) { return this.attrs.has(k) ? this.attrs.get(k) : null; }
		setAttribute(k, v) { this.attrs.set(k, String(v)); }
		hasAttribute(k) { return this.attrs.has(k); }
		removeAttribute(k) { this.attrs.delete(k); }
		append(...nodes) {
			for (const n of nodes) {
				if (n.parentNode) n.remove();
				n.parentNode = this;
				this.children.push(n);
			}
		}
		replaceChildren(...nodes) {
			for (const c of this.children) c.parentNode = null;
			this.children = [];
			this.append(...nodes);
		}
		remove() {
			if (!this.parentNode) return;
			const i = this.parentNode.children.indexOf(this);
			if (i >= 0) this.parentNode.children.splice(i, 1);
			this.parentNode = null;
		}
		cloneNode() {
			const c = new FakeElement(this.tagName);
			c.attrs = new Map(this.attrs);
			c.textContent = this.textContent;
			return c;
		}
		addEventListener(type, fn, opts) {
			if (opts?.signal?.aborted) return;
			if (!this._listeners.has(type)) this._listeners.set(type, []);
			this._listeners.get(type).push(fn);
			if (opts?.signal) {
				opts.signal._cbs.push(() => {
					const arr = this._listeners.get(type);
					if (!arr) return;
					const i = arr.indexOf(fn);
					if (i >= 0) arr.splice(i, 1);
				});
			}
		}
		dispatchEvent(ev) {
			for (const fn of [...(this._listeners.get(ev.type) || [])]) fn(ev);
			return !ev.defaultPrevented;
		}
		getRootNode() { return document; }
		querySelector(sel) { return this.querySelectorAll(sel)[0] || null; }
		querySelectorAll(sel) {
			const parts = sel.split(",").map((s) => s.trim());
			const out = [];
			for (const p of parts) {
				if (p.startsWith(":scope > ")) {
					const sub = p.slice(":scope > ".length);
					for (const c of this.children) if (matches(c, sub)) out.push(c);
				} else {
					const walk = (n) => {
						for (const c of n.children) {
							if (matches(c, p)) out.push(c);
							walk(c);
						}
					};
					walk(this);
				}
			}
			return out;
		}
	}

	function matches(el, sel) {
		if (sel === "select") return el.tagName === "SELECT";
		if (sel === "option") return el.tagName === "OPTION";
		if (sel === ".ui-tristate-track") return el.classList.contains("ui-tristate-track");
		if (sel === '[role="status"]') return el.getAttribute("role") === "status";
		if (sel === "template.ui-tristate-marks") return el.tagName === "TEMPLATE" && el.classList.contains("ui-tristate-marks");
		const m = /^label\[for="(.+)"\]$/.exec(sel);
		if (m) return el.tagName === "LABEL" && el.getAttribute("for") === m[1].replace(/\\(.)/g, "$1");
		return false;
	}

	globalThis.HTMLElement = FakeElement;
	let dir = "ltr";
	globalThis.getComputedStyle = () => ({ direction: dir });

	const defined = new Map();
	globalThis.customElements = {
		get: (name) => defined.get(name),
		define: (name, cls) => defined.set(name, cls),
	};

	const root = new FakeElement("div");
	globalThis.document = {
		createElement: (tag) => new FakeElement(tag),
		querySelector: (sel) => root.querySelector(sel),
		querySelectorAll: (sel) => root.querySelectorAll(sel),
	};
	`
	if _, err := vm.RunString(harness); err != nil {
		t.Fatalf("harness: %v", err)
	}
	if _, err := vm.RunScript("../../theme/js/tristate.js", exportKeyword.ReplaceAllString(string(src), "")); err != nil {
		t.Fatalf("tristate.js: %v", err)
	}

	script := `
	const UITristate = customElements.get("ui-tristate");
	if (!UITristate) throw new Error("ui-tristate not defined");

	const form = new FakeElement("form");
	const label = new FakeElement("label");
	label.setAttribute("for", "field--1");
	label.textContent = "Wrapped as a gift?";

	const host = new UITristate("ui-tristate");
	host.className = "ui-tristate";
	host.dataset.uiUnset = "No answer";

	const select = new FakeElement("select");
	select.id = "field--1";
	select.className = "ui-input";
	select.setAttribute("aria-describedby", "field--1-help field--1-error");
	select.setAttribute("aria-invalid", "true");
	select.setAttribute("required", "");
	select.form = form;

	const optEmpty = new FakeElement("option"); optEmpty.value = ""; optEmpty.textContent = "Choose…";
	const optYes = new FakeElement("option"); optYes.value = "true"; optYes.textContent = "Yes";
	const optNo = new FakeElement("option"); optNo.value = "false"; optNo.textContent = "No";
	select.options = [optEmpty, optYes, optNo];
	select.append(optEmpty, optYes, optNo);

	const tpl = new FakeElement("template");
	tpl.className = "ui-tristate-marks";
	const svgYes = new FakeElement("svg"); svgYes.textContent = "CHECK";
	const svgUnset = new FakeElement("svg"); svgUnset.textContent = "MINUS";
	const svgNo = new FakeElement("svg"); svgNo.textContent = "X";
	tpl.content = { children: [svgYes, svgUnset, svgNo] };

	host.append(select, tpl);
	form.append(label, host);
	root.append(form);

	host.connectedCallback();

	const track = host.querySelector(":scope > .ui-tristate-track");
	const live = host.querySelector(':scope > [role="status"]');
	const knob = track.children[0];
	const stateSpan = track.children[1];
	track.getBoundingClientRect = () => ({ left: 100, right: 220, top: 10, bottom: 38, width: 120, height: 28 });

	function assert(cond, msg) { if (!cond) throw new Error(msg); }

	// Initial upgrade state.
	assert(track.getAttribute("role") === "checkbox", "role");
	assert(track.getAttribute("aria-checked") === "mixed", "initial aria-checked");
	assert(track.dataset.state === "unset", "initial data-state");
	assert(knob.children[0].textContent === "MINUS", "initial mark");
	assert(stateSpan.textContent === "No answer", "initial word");
	assert(track.getAttribute("aria-describedby") === "field--1-help field--1-error field--1-state", "describedby: " + track.getAttribute("aria-describedby"));
	assert(track.getAttribute("aria-invalid") === "true", "aria-invalid");
	assert(track.getAttribute("aria-required") === "true", "aria-required");
	assert(!select.hasAttribute("required"), "required removed from select");
	assert(label.getAttribute("for") === track.id, "label for moved");
	assert(select.classList.contains("ui-visually-hidden"), "select visually hidden");
	assert(select.tabIndex === -1, "select tabindex");
	assert(select.getAttribute("aria-hidden") === "true", "select aria-hidden");

	// Track select events.
	const events = [];
	select.addEventListener("input", () => events.push("input:" + select.value));
	select.addEventListener("change", () => events.push("change:" + select.value));

	// Space cycles unset -> yes -> no -> unset.
	const key = (k) => {
		const ev = new Event("keydown");
		ev.key = k;
		track.dispatchEvent(ev);
		return ev;
	};
	assert(key(" ").defaultPrevented, "space prevented");
	flushTimers();
	assert(select.value === "true" && track.dataset.state === "yes" && track.getAttribute("aria-checked") === "true", "space -> yes");
	assert(knob.children[0].textContent === "CHECK" && live.textContent === "Yes", "yes mark and live");
	assert(events.join(",") === "input:true,change:true", "events: " + events.join(","));

	key(" ");
	flushTimers();
	assert(select.value === "false" && track.dataset.state === "no" && track.getAttribute("aria-checked") === "false", "space -> no");
	assert(knob.children[0].textContent === "X" && live.textContent === "No", "no mark and live");

	key(" ");
	flushTimers();
	assert(select.value === "" && track.dataset.state === "unset" && track.getAttribute("aria-checked") === "mixed", "space -> unset");

	// Arrows LTR: Right moves toward yes and stops; Left moves toward no and stops.
	key("ArrowRight");
	assert(track.dataset.state === "yes", "LTR ArrowRight -> yes");
	events.length = 0;
	key("ArrowRight"); // at end: no change event
	assert(track.dataset.state === "yes" && events.length === 0, "stop at yes");
	key("ArrowLeft");
	assert(track.dataset.state === "unset", "LTR ArrowLeft -> unset");
	key("ArrowLeft");
	assert(track.dataset.state === "no", "LTR ArrowLeft -> no");
	key("ArrowLeft");
	assert(track.dataset.state === "no", "stop at no");

	// Home / End / Delete / Backspace.
	key("End"); assert(track.dataset.state === "yes", "End -> yes");
	key("Delete"); assert(track.dataset.state === "unset", "Delete -> unset");
	key("Home"); assert(track.dataset.state === "no", "Home -> no");
	key("Backspace"); assert(track.dataset.state === "unset", "Backspace -> unset");

	// Pointer thirds in LTR.
	const clickAt = (x, y) => {
		const ev = new Event("click");
		ev.detail = 1; ev.clientX = x; ev.clientY = y;
		track.dispatchEvent(ev);
	};
	clickAt(110, 20); assert(track.dataset.state === "no", "LTR start third -> no");
	clickAt(160, 20); assert(track.dataset.state === "unset", "LTR middle third -> unset");
	clickAt(210, 20); assert(track.dataset.state === "yes", "LTR end third -> yes");

	// Pointer thirds and arrows in RTL.
	dir = "rtl";
	clickAt(110, 20); // left edge = end in RTL = yes (already yes)
	assert(track.dataset.state === "yes", "RTL left third -> yes");
	clickAt(210, 20); // right edge = start in RTL = no
	assert(track.dataset.state === "no", "RTL right third -> no");
	key("ArrowLeft"); assert(track.dataset.state === "unset", "RTL ArrowLeft -> unset");
	key("ArrowLeft"); assert(track.dataset.state === "yes", "RTL ArrowLeft -> yes");
	key("ArrowRight"); assert(track.dataset.state === "unset", "RTL ArrowRight -> unset");
	dir = "ltr";

	// Label click cycles to next state (unset -> yes).
	label.dispatchEvent(new Event("click"));
	track.dispatchEvent(new Event("click"));
	flushMicrotasks();
	assert(track.dataset.state === "yes", "label click -> yes");

	// MutationObserver mirrors disabled on select.
	select.disabled = true;
	assert(track.disabled === true, "disabled mirrored true");
	key("Home");
	assert(track.dataset.state === "yes", "disabled ignores keys");
	select.disabled = false;
	assert(track.disabled === false, "disabled mirrored false");

	// External change on select keeps track in step.
	select.value = "false";
	select.dispatchEvent(new Event("change"));
	assert(track.dataset.state === "no", "external select change synced");

	// Form reset syncs in a task (after value resets).
	form.dispatchEvent(new Event("reset"));
	select.value = "";
	flushTimers();
	assert(track.dataset.state === "unset", "form reset synced");

	// Idempotent re-upgrade (htmx history snapshot): track & live replaced,
	// never doubled, aria-required and label for preserved.
	select.value = "true";
	host.connectedCallback();
	assert(host.querySelectorAll(":scope > .ui-tristate-track").length === 1, "single track after re-upgrade");
	assert(host.querySelectorAll(':scope > [role="status"]').length === 1, "single live after re-upgrade");
	const track2 = host.querySelector(":scope > .ui-tristate-track");
	assert(track2.getAttribute("aria-required") === "true", "aria-required kept on re-upgrade");
	assert(track2.dataset.state === "yes", "state read from select on re-upgrade");
	assert(label.getAttribute("for") === track2.id, "label for kept on re-upgrade");

	// Disconnect removes form listeners and observer.
	host.isConnected = false;
	host.disconnectedCallback();
	assert(observers.length === 0, "observer disconnected");
	assert((form._listeners.get("reset") || []).length === 0, "form reset listener removed");
	assert((label._listeners.get("click") || []).length === 0, "label listener removed");
	`
	if _, err := vm.RunString(script); err != nil {
		t.Fatalf("tristate element test failed: %v", err)
	}
}
