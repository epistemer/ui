# Review 0005: interface draft of the visual packages (D5)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 97k tokens: the milestone's one Argon (High) run. Gemini 3.8 Flash (Low) restated three findings the output limit had cut (about 74k tokens, mostly the session's context) |
| Under review | docs/visual-interface.md, alone in a scratch folder, with excerpts: visual-design.md §9 and findings 7–20, the kernel's slot rules (interface §4.1), forms' `Control` (§5.10), uitest (§8), cfgx's `Validate` rule, the essentials of ADRs 0004 and 0005, and epistemer's standing rules |
| Scope | Contradictions with the locked kernel, forms and uitest contracts and with the ADRs; API shapes expensive to change after the lock; accessibility; claims of what the spike proved; themeschema; missing pieces; the defaults of V1–V17 |

There were 18 findings: 5 ranked blocking, 9 major and 4 minor. Claude
checked each against the draft, the spike and the locked contracts. Two
were answered rather than taken; the rest changed the draft.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| C1 | `PageHeader` has one region but a slot, breaking §2.1's own rule; an unfilled `TooltipTrigger` renders an orphaned tooltip; `TooltipSpec` and `NavSectionSpec` have no `Attrs` | Valid | PageHeader's actions are children; an unfilled required slot (`TooltipTrigger`, `DialogBody`, `MenuItems`, `AppMain`, `AuthBody`) panics at render; every spec has `Attrs` (§2.1) |
| C2 | The popover drawer keeps light dismiss armed when the screen widens, doesn't move focus in, leaves `aria-expanded` unset, and stays open after an `hx-boost` navigation | Valid | V17's default is now a non-modal drawer driven by the toggle's `aria-expanded` through the ui script (focus, Esc, outside click, navigation, widening), a plain block without the script (§8.2, §10.2) |
| C3 | Toasts: appended children don't run the region's `connectedCallback`; a modal dialog hides and inerts the region; focus fallbacks to elements without `tabindex`; a full page's toasts aren't announced; `ToastSwap` breaks ADR 0005's `hx-*` rule | Valid | Each toast is a `<ui-toast>` element that moves itself into the region, so a full page's toast is announced too; the region is a manual popover raised when a toast arrives, which the V milestone checks first above a modal; `#ui-main` takes `tabindex="-1"`; `ToastSwap` becomes question V18, an exception to write into ADR 0005 (§7.5, §10.5) |
| C4 | Serving current assets under an unknown release mixes modules of two releases | Valid | A 404, as spiked; V16 withdrawn (§3.3) |
| C5 | Tones: Alert and Toast take `Brand`, which has no icon or word; no neutral button; `MenuItem.Danger` against `Button.Tone` | Valid in the first two parts | `Neutral` joins the tones; Alert and Toast take the four message tones and panic on others. `Danger()` stays: §2.1 gives an argument-less method to what fixes the thing, and a menu item has one tone besides neutral. V7's text updated (§2.3) |
| M1 | `CheckFont`'s bare word has no character set; `CheckRadius` unsigned isn't stated; the radius scale isn't tied to the base | Valid (the spike's regexp was stricter than the text) | The text now gives the character sets, refuses CSS-wide keywords, states the number is unsigned and the scale's `calc()` (§3.2) |
| M2 | "Internal hooks" contradict the import rule, or can't exist | Half valid: Go's `internal` packages are importable across ui, as forms' hook for formstest is | The module table names `ui/internal/themehook` and `ui/internal/a11y` and what each carries (§1, §9.1) |
| M3 | A hidden "Error" runs into the message ("Errormessage") | Valid | The level words carry their punctuation ("Error:"), per locale, and ui renders a space after them (§5.2, §6.2, §11) |
| M4 | `ConfirmDialog` lacks `aria-labelledby` and `aria-describedby` | Valid (the spike has both; the draft dropped them) | Restored (§7.2) |
| M5 | The reveal button: `EyeOff` unused, its class unlisted, form reset ignored, and a button that does nothing without the script | Valid | Both icons, shown by `aria-pressed`; class listed; reset handled; hidden until `<html data-ui-ready>` (§6.3, §10.1) |
| M6 | Esc on a tooltip inside a dialog closes the dialog; no HTML state for the dismissal; clipping | Valid | The script sets `hidden` on the text and stops the key; clipping is checked in the V milestone, with a manual popover as the fallback (§5.3, §10.2) |
| M7 | The colour scheme menu doesn't expose the current scheme | Valid | `ColorSchemeMenu(current)` renders `menuitemradio` items with `aria-checked`, kept in step by the script (§8.2) |
| M8 | Claims beyond the spike: V2 says no fallback is needed; htmx paths marked spiked; finding 20's corrections, themeschema and catalogtest unmarked; file kinds against "uploads wait for D7" | Valid except the file kinds: the spike rendered `KindFile` and `KindFiles`, and D7's uploads are a component (previews, progress), not forms' file input | Each marked; V2 reworded; the collected list at §14 now names every unspiked piece and what was spiked but never run in a browser |
| M9 | Missing markup: checkbox, password and group shapes; Tab's `<li>`; Auth's wrapper and Toasts inside `main`; `Head` takes one component; no body attributes; part classes | Valid | All specified: `ui-field-row`, the group's help and error, `Tab` renders its `<li>`, Auth's outer div with Toasts and the dialog outside `main`, `Head(...)` variadic, `BodyAttrs`, every part class (§6.2, §7.4, §8) |
| m1 | `Merge` reads as panicking on an absent key; `aria-controls` isn't joined; the backlog gate is met, so move `Merge` to the kernel | First two valid. The third is V5, epistemer's to decide | Wording fixed; `aria-controls` joined (§2.2) |
| m2 | `Text(i18n.Static(""))` renders an empty span; empty `src` or `href` | Valid | A text that renders empty or white space, and an empty `href` or `src`, render `Empty`; `Image`'s nil `alt` named as the exception (§5.4) |
| m3 | V15's default costs two lookups per field and drops a placeholder equal to the label; take `HasPlaceholder` | Answered, not taken | Two `T(ctx)` calls per text field are cheap beside a render, and a placeholder that repeats the label is the noise finding 12 removes. The amendment stays the alternative |
| m4 | `MenuItem` submits through raw `Attrs`; no `pageshow`; the invoker fallback's `value` | Valid | `MenuItem.Submit(form)`; `pageshow` refreshes conditions; the fallback passes `value` to `close` (§7.3, §10.2) |
