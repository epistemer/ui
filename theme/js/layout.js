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
		// A click outside the drawer closes it, and so does choosing an item.
		if (!nav?.contains(target) || target.closest("a[href]")) open.setAttribute("aria-expanded", "false");
	}
});

document.addEventListener("keydown", (event) => {
	if (event.key !== "Escape" || event.defaultPrevented) return;
	// Esc belongs to an open popover or modal first.
	if (document.querySelector(":popover-open, dialog:modal")) return;
	for (const toggle of openToggles()) {
		toggle.setAttribute("aria-expanded", "false");
		if (navOf(toggle)?.contains(document.activeElement)) toggle.focus();
	}
});

document.addEventListener("htmx:afterSwap", closeAll);
wide.addEventListener("change", (event) => event.matches && closeAll());

// The colour scheme menu: <html data-color-scheme> and the cookie
// ui-color-scheme, which layout.ColorSchemeOf reads, so the next page
// renders in the same scheme. "system" clears both.
const schemes = new Set(["light", "dark"]);

document.addEventListener("click", (event) => {
	const item = event.target instanceof Element ? event.target.closest("[data-ui-color-scheme]") : null;
	if (!item || item.getAttribute("aria-disabled") === "true") return;
	const picked = schemes.has(item.dataset.uiColorScheme) ? item.dataset.uiColorScheme : "system";
	const root = document.documentElement;
	if (picked === "system") {
		delete root.dataset.colorScheme;
		document.cookie = "ui-color-scheme=; path=/; max-age=0; SameSite=Lax";
	} else {
		root.dataset.colorScheme = picked;
		document.cookie = `ui-color-scheme=${picked}; path=/; max-age=31536000; SameSite=Lax`;
	}
	for (const el of document.querySelectorAll("[data-ui-color-scheme]")) {
		el.setAttribute("aria-checked", String(el.dataset.uiColorScheme === picked));
	}
});
