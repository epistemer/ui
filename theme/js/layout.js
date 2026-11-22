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
