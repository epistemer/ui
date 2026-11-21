# Review 0002: visual packages design draft (D2, D3)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 71k tokens |
| Under review | docs/ui-inventory.md, ADR 0004, ADR 0005, docs/visual-design.md and the roadmap's D track, as a diff, with excerpts of the kernel and forms contracts and of epistemer's standing rules |
| Scope | Wrong technical claims, contradictions between the documents, breaks of the kernel or forms contracts, the run-time theme, contrast, Q2, attribute pass-through, tests, accessibility, and what would be expensive to change after a lock |

There were 19 findings: 6 ranked blocking, 9 major and 4 minor. Claude
checked each one against the documents, and checked the kernel's code
where a finding made a claim about it.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | CSS kept next to each package can't be embedded by `ui/theme`: `embed` can't reach sibling directories, and importing the visual packages would invert the layers | Valid | All CSS lives in `theme/css/`. The visual packages don't import `ui/theme` (ADR 0004 delivery, design §3) |
| 2 | An open popover is in the top layer, so an `@supports` CSS fallback can't place it next to its trigger. Per-instance anchor names would need inline styles, which a strict CSP blocks | Valid | The fallback is the script, positioning through CSSOM. How a trigger names its anchor is tried in D4, which now includes the menu |
| 3 | Generated ids restart with each render and collide across htmx swaps; cloned template rows share ids | Half valid. Generated ids come from a process-wide source (`idsource.go`: a random prefix and a counter), so swapped-in fragments never collide. Clones do share the template's ids | `<ui-repeat>` gives each clone fresh ids and rewrites references (ADR 0005) |
| 4 | Behaviour attributes "applied last" lose: HTML keeps the first copy of a duplicate attribute | Valid | Attributes are merged in Go before one spread: case-insensitive keys, joined id lists and classes, behaviour keys replace the caller's. Each component documents which element receives them (design §5) |
| 5 | Ordering ui before Tailwind v4's `base` lets Preflight beat every ui rule. Banning `!important` breaks `hidden` | Valid | Layered apps put ui after their resets. `[hidden] { display: none !important }` in `ui.reset` is the one `!important` (ADR 0004) |
| 6 | Native popovers don't change a rendered `aria-expanded`, so a fixed value goes stale; nothing moves focus into a menu | Valid | `<ui-menu>` keeps `aria-expanded` in step on `toggle` and focuses the first item. The server never renders a state the browser changes (ADR 0004, 0005) |
| 7 | `color-scheme` on `<html>` as a style is blocked by `style-src 'self'`; htmx injects an inline indicator style | Valid | `<html data-color-scheme>` mapped in `ui.tokens`. Apps turn off htmx's indicator styles, and `ui.base` styles `.htmx-indicator` |
| 8 | Theme values can inject CSS; a stale hash 404s or poisons caches; coupling to cfgx's change callbacks; the head component needs state | Valid in the main | Values are validated by kind at the key and when the stylesheet is built. The hash comes from the current values. A stale hash gets the current theme, not cached. No cfgx callback is needed. The head component takes the theme as an argument (design §4) |
| 9 | Status colours have the brand's text-contrast problem; shifting oklch lightness leaves sRGB; raised surfaces; derived shades break the on-colour's contrast | Valid | Four tokens per accent role. Gamut-mapped derivation against the stricter surface. The on-colour is checked against the hover and active mixes too. The focus ring is checked at 3:1 (design §4) |
| 10 | A confirm dialog inside a menu's popover is hidden with it; inside the form it confirms it nests forms; one dialog per row bloats tables | Valid | One shared confirmation dialog in the shell, answering `htmx:confirm` and `data-ui-confirm` (ADR 0005) |
| 11 | Typed variadic options collide across components in one package, hide what applies in `go doc`, and make conditions awkward in templ; a disabled `<a>` | Valid | Q2's default becomes builders, as in forms. The disabled link rule is in design §5 |
| 12 | The literal-value check would reject media and container conditions and structural values; the physical-property check misses shorthands, `translateX` and gradients; "RTL changes only CSS" is false; class names in markup and CSS can drift apart | Valid | Allowlist and condition exemption; the side check covers shorthands, `translateX` and gradients; `dir` belongs to the HTML side; a selector check joins the catalogue tests |
| 13 | Hashed file names break relative module imports; an import map needs an inline script; a test-only engine still enters go.mod | Valid | One hashed directory for all assets. The engine's test lives in a nested module (ADR 0005) |
| 14 | A CSS-only tooltip can't be dismissed with Esc; `<details>` isn't a fieldset; links with `aria-current` aren't APG tabs; toast region lifetime; gaps in the names check | Valid | Esc in the delegated listener; `<fieldset>` with `<legend>`; navigation tabs named as such; the region stays in the shell; the names check tightened (design §6, §7) |
| 15 | The slice skips the riskiest bets; server-state components would be locked before their design; depending on models breaks the getter-interface rule | First two valid. The third is rejected: epistemer decided UI packages consume models' `option.SelectOption` and `listing.*` interfaces (2026-11-16), which point the dependency from ui to models on purpose | D4 adds the menu and the run-time theme; Q9 keeps server-state components out of the first lock |
| 16 | The ADR says no generated CSS while the design generates `tokens.css`; `ui.reset` and `ui.base` have no owner | Valid | The token layer is rendered at start-up from the table, so no file is generated; `ui/theme` owns reset and base |
| 17 | Twelve iframes per variant; inline style edits break CSP | Valid | A two-by-two grid with a width switcher; token edits through CSSOM |
| 18 | The amount renderer from the inventory was dropped | Valid | `Amount` is back, with tabular numerals and isolated direction |
| 19 | The invoker fallback can fire twice or submit a form | Valid | Feature-detected early return, `preventDefault`, and every ui `<button>` states its `type` |
