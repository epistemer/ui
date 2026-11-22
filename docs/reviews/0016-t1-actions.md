# Review 0016: T1, actions

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 64k tokens |
| Under review | The T1 diff without tests, generated code and goldens (`ui/actions`, the Unavailable states, `Menu.ID`, the placements, the action form and dialog, `Reply`, `actions.js`, `menu.js`), with server-state-interface.md §2, §4.2–§4.4 and §5 |
| Scope | The code against the locked contract: inertness, `Reply`, names and descriptions, the script's focus and dialog handling, templ pitfalls |

The review ran long and repeated itself; its findings, deduplicated, were
these. Claude checked each against the code, the goldens and the pane.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | An unavailable Button loses its label inside the Tooltip's blocks | Rejected: templ captures `children` once at the template's start; the golden shows the label | — |
| 2 | `ButtonSpec.Attrs` replaces, so an item's own attributes are lost | Rejected: it merges | — |
| 3 | Unavailable controls keep `data-hx-*` (htmx's alias), and a menu item keeps `command`, `commandfor`, `popovertarget` | Valid | Both drop them; tested |
| 4 | A click on an unavailable button reaches listeners, and closes a menu | Rejected: ui.js stops clicks on any `aria-disabled="true"` element, and menu.js closes only for available items; the pane showed the menu open | — |
| 5 | Focus is remembered only for dialog actions, not for a request whose answer replaces or removes its row; a fallback is never reached when the row id is empty (`??` against `""`) | Valid | Any action control's request records its opener (its own id, its row's, its table's); the fallback picks the first connected, visible candidate |
| 6 | A dialog closed by Esc or its button keeps a stale opener | Valid | Closed without an answer, the opener is forgotten |
| 7 | `refocus` runs before the observer empties the dialog, so focus is still inside it and nothing moves | Valid | The dialog is emptied before focus moves, and focus in a closed dialog counts as lost; checked in the pane |
