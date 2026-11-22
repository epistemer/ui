# Review 0011: V6, layout

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 63k tokens |
| Under review | The V6 diff (ui/layout, its CSS layer, the script's drawer and colour scheme), with visual-interface.md §8, §2's rules and the script rows |
| Scope | §8's markup, accessibility of the shells and the drawer, the script, CSS, the Go API |

There were 9 findings. Claude checked each; eight changed the code, and
the behaviour was checked again in the browser pane.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | On narrow screens the open drawer covers the bar and its own toggle | Valid | The bar is 3.5em high and the drawer starts below it, so the toggle stays in reach |
| 2 | A wide app without navigation keeps an empty column | Valid | Two columns only for `.ui-app:has(> .ui-app-nav)` |
| 3 | The toggle comes before the brand, not after as §8.2 lists them; unfilled brand and bar-end render empty boxes | Valid for the empty boxes. The order is kept: the toggle opens a drawer from the start edge, so it sits at that edge, ahead of the brand, as is usual. §8.2 lists the bar's parts without fixing their order | Brand and bar-end render only when filled |
| 4 | Esc can race a popover's own listener; focus doesn't always return; the key isn't consumed | Valid in part. ui's menus and dialogs leave Esc to the browser, which acts after the event, so no race is possible with them; another script's could be | The listener captures, consumes the key when it closes a drawer, and returns focus to the toggle whenever focus was in the drawer |
| 5 | `ColorSchemeMenu` panics when called, not when rendered | Valid | Checked at render |
| 6 | `rem` in media queries, `85vi`, the skip link's 1px box; the sticky column's guessed offset; no scroll padding; the page header's grid gaps for missing parts | Valid for the offset, the scroll padding and the gaps. The units are rejected: the checker allows literals in media queries and relative units, and the 1px box is the visually-hidden pattern base.css already uses | A fixed bar height that the column, the drawer and `scroll-padding-block-start` start below; the page header spaces by margins, so a missing part leaves no gap |
| 7 | The actions render with `PageHeader`'s children still in the context; a crumb above the page may lack its href; a blank description renders | Valid | Children cleared; such a crumb panics; a blank description is left out |
| 8 | Choosing an item with a modifier key or a middle click closes the drawer; a swap while focus is in the drawer loses focus | Valid | Only a plain, uncancelled click on an item closes it; a swap returns focus to the toggle |
| 9 | After a back-forward restore, the scheme items' `aria-checked` can disagree with `<html>`; an unknown value resets to system | Valid | Items are synced from `<html>` on `pageshow` and `htmx:historyRestore`; an unknown value is ignored |
