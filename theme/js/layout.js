// ui/layout's behaviour (visual-interface.md §8.2, §10.2): the app
// shell's navigation drawer and the colour scheme menu.

// The drawer. Its state is the toggle's aria-expanded, which CSS reads, so
// there's no other state to keep in step. It is a non-modal disclosure:
// the page behind stays reachable. The breakpoint is layout.css's.
const wide = matchMedia("(width >= 60rem)");

const openToggles = () => document.querySelectorAll('[data-ui-nav-toggle][aria-expanded="true"]');
const navOf = (toggle) => document.getElementById(toggle.getAttribute("aria-controls"));

function setDrawer(toggle, open) {
	toggle.setAttribute("aria-expanded", String(open));
	if (open) navOf(toggle)?.querySelector("a[href], button:not(:disabled)")?.focus();
}

function closeAll() {
	for (const toggle of openToggles()) toggle.setAttribute("aria-expanded", "false");
}

document.addEventListener("click", (event) => {
	const target = event.target instanceof Element ? event.target : null;
	if (!target) return;
	const toggle = target.closest("[data-ui-nav-toggle]");
	if (toggle) {
		setDrawer(toggle, toggle.getAttribute("aria-expanded") !== "true");
		return;
	}
	for (const open of openToggles()) {
		const nav = navOf(open);
		if (!nav?.contains(target)) {
			open.setAttribute("aria-expanded", "false"); // a click outside it
			continue;
		}
		// Choosing an item closes it, but not a click that opens the link
		// elsewhere (a new tab or window) or that a listener cancelled.
		const elsewhere = event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey;
		if (target.closest("a[href]") && !elsewhere && !event.defaultPrevented) open.setAttribute("aria-expanded", "false");
	}
});

// close hides a drawer; focus inside it, which would be lost, goes back to
// the toggle.
function close(toggle) {
	const hadFocus = navOf(toggle)?.contains(document.activeElement);
	toggle.setAttribute("aria-expanded", "false");
	if (hadFocus) toggle.focus();
}

// Capturing, so this runs before any listener that might close a popover
// or dialog: Esc belongs to an open popover or modal first.
document.addEventListener(
	"keydown",
	(event) => {
		if (event.key !== "Escape" || event.defaultPrevented) return;
		if (document.querySelector(":popover-open, dialog:modal")) return;
		const open = [...openToggles()];
		if (!open.length) return;
		event.preventDefault();
		for (const toggle of open) close(toggle);
	},
	true,
);

document.addEventListener("htmx:afterSwap", () => {
	for (const toggle of openToggles()) close(toggle);
});
wide.addEventListener("change", (event) => event.matches && closeAll());

// The colour scheme menu: <html data-color-scheme> and the cookie
// ui-color-scheme, which layout.ColorSchemeOf reads, so the next page
// renders in the same scheme. "system" clears both.
const schemes = new Set(["system", "light", "dark"]);

// syncSchemes checks the item for the scheme in use, from <html>: after a
// pick, and when htmx or the back-forward cache restores a page rendered
// with an older one.
function syncSchemes() {
	const current = document.documentElement.dataset.colorScheme ?? "system";
	for (const el of document.querySelectorAll("[data-ui-color-scheme]")) {
		el.setAttribute("aria-checked", String(el.dataset.uiColorScheme === current));
	}
}

document.addEventListener("click", (event) => {
	const item = event.target instanceof Element ? event.target.closest("[data-ui-color-scheme]") : null;
	if (!item || item.getAttribute("aria-disabled") === "true") return;
	const picked = item.dataset.uiColorScheme;
	if (!schemes.has(picked)) return;
	const root = document.documentElement;
	if (picked === "system") {
		delete root.dataset.colorScheme;
		document.cookie = "ui-color-scheme=; path=/; max-age=0; SameSite=Lax";
	} else {
		root.dataset.colorScheme = picked;
		document.cookie = `ui-color-scheme=${picked}; path=/; max-age=31536000; SameSite=Lax`;
	}
	syncSchemes();
});

document.addEventListener("htmx:historyRestore", syncSchemes);
window.addEventListener("pageshow", (event) => event.persisted && syncSchemes());

// Boosted navigation and history restore: htmx 2 swaps only the body, so
// <html lang> and <html dir> would stay those of the first page. Read them
// from the response's <html> start tag (parsed in isolation, never executing
// the response) and remember them per path for htmx's history cache.
const htmlStartTag =
	/^\s*(?:<!--[\s\S]*?-->\s*)*(?:<!doctype[^>]*>\s*)?(?:<!--[\s\S]*?-->\s*)*(<html(?:\s+(?:[^>"']|"[^"]*"|'[^']*')*|\s*)>)/i;

function readHtmlAttrs() {
	const root = document.documentElement;
	return {
		lang: root.getAttribute("lang"),
		dir: root.getAttribute("dir"),
	};
}

function applyHtmlAttrs({ lang, dir }) {
	const root = document.documentElement;
	if (lang === null) root.removeAttribute("lang");
	else root.setAttribute("lang", lang);
	if (dir === null) root.removeAttribute("dir");
	else root.setAttribute("dir", dir);
}

function syncHtmlFromResponse(xhr) {
	const text = xhr?.responseText;
	if (typeof text !== "string") return false;
	const match = htmlStartTag.exec(text);
	if (!match) return false;
	const doc = new DOMParser().parseFromString(`${match[1]}</html>`, "text/html");
	const root = doc.documentElement;
	applyHtmlAttrs({
		lang: root.getAttribute("lang"),
		dir: root.getAttribute("dir"),
	});
	return true;
}

// lang and dir per page, for htmx's history cache, which restores only the
// body. Kept in sessionStorage, as htmx keeps its cache, so a reload
// between two boosted pages still restores the right ones.
const historyKey = "ui-html-attrs";
const historyLimit = 50;

function historyLoad() {
	try {
		return new Map(JSON.parse(sessionStorage.getItem(historyKey) ?? "[]"));
	} catch {
		return new Map();
	}
}

const historyAttrs = historyLoad();

// pathOf is a history path as htmx's historyRestore reports it: path and
// query, without the fragment.
function pathOf(path) {
	try {
		const u = new URL(path, location.href);
		return u.pathname + u.search;
	} catch {
		return null;
	}
}

function remember(path, attrs) {
	const key = path && pathOf(path);
	if (!key) return;
	historyAttrs.delete(key);
	historyAttrs.set(key, attrs);
	while (historyAttrs.size > historyLimit) historyAttrs.delete(historyAttrs.keys().next().value);
	try {
		sessionStorage.setItem(historyKey, JSON.stringify([...historyAttrs]));
	} catch {
		// Storage full or refused: the page still works, Back may keep the
		// current lang and dir.
	}
}

const here = () => location.pathname + location.search;

// The page as it is now, before htmx saves it to its cache.
if (typeof location !== "undefined") remember(here(), readHtmlAttrs());
// htmx saves the page it is leaving under its own path, also on Back,
// when location already names the page it goes to.
document.addEventListener("htmx:beforeHistorySave", (event) => remember(event.detail?.path ?? here(), readHtmlAttrs()));

// A boosted page: htmx has already pushed or replaced the address (before
// the swap), so location is the new page's. Only a swap of the whole body
// is a page; a boosted link with its own target swaps a part.
document.addEventListener("htmx:afterSwap", (event) => {
	if (!event.detail?.boosted || event.detail.target !== document.body) return;
	if (syncHtmlFromResponse(event.detail.xhr)) remember(here(), readHtmlAttrs());
});

document.addEventListener("htmx:historyCacheMissLoad", (event) => {
	if (syncHtmlFromResponse(event.detail?.xhr)) remember(event.detail?.path, readHtmlAttrs());
});

document.addEventListener("htmx:historyRestore", (event) => {
	const key = pathOf(event.detail?.path ?? "");
	const saved = key && historyAttrs.get(key);
	if (saved) applyHtmlAttrs(saved);
});
