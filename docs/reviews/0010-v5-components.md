# Review 0010: V5, components

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 40k tokens |
| Under review | The V5 diff (ui/components, its CSS, the script's menu, confirmation and toasts), with visual-interface.md §7 and §10.5 |
| Scope | §7's markup and wiring, the script's behaviour around modals, focus and live regions, CSS, accessibility |

There were 12 findings. Claude checked each; eight changed the code. The
behaviour was checked again in the browser pane.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | Toasts already shown stay behind the inert page when a modal opens; the page's region isn't hidden again after its toasts move, nor when the dialog's region empties | Valid | A modal opening moves the page's toasts into its region; the page's region is synced after every move and when the dialog's region is removed |
| 2 | A toast's timer isn't reset after `clearInterval`, so a toast connected again never times out | Valid | Reset to 0; a toast moved between regions keeps its timer, and only one that stays disconnected is stopped |
| 3 | A live region shown together with its content isn't announced | Valid | The region is shown first and the toast added in a later task |
| 4 | The pause flag can stay set after a focused toast is removed | Valid | No flag: each tick reads `:hover, :focus-within` from the page |
| 5 | The first `:modal` found isn't necessarily the top one; focus falls back to a dialog that can't take it | Valid | The modal holding focus, else the last opened; focus falls back to the dialog's first control |
| 6 | The menu's own placement measures the last placement, doesn't flip on the inline axis, runs on every scroll event, and leaks listeners when removed while open | Valid, except the leak: removal already stops following | Measured from a clean start, flips to the other edge before clamping, one placement per frame; a pending frame is cancelled when it stops following |
| 7 | CSS: `overflow-x` on tabs, `@container (min-width)`, no `:focus-visible` on the toast's close button and the tabs, `--ui-focus-width` reused for indicator borders, a hard-coded 4px gap in the script | Valid for the container query and the gap. The rest rejected: `overflow-inline` isn't supported widely enough, and `overflow-x` is the inline axis in the horizontal modes ui supports; base.css gives every element a `:focus-visible` ring; a separate indicator width would be a new token, a contract change for no visible gain | `@container (inline-size >= 32em)`; the script resolves `--ui-space-1` for its gap |
| 8 | The Dialog's default trigger is brand, the Menu's neutral; the close button is small; blank titles and labels are accepted | Valid for blank names. The tones are deliberate: a dialog's trigger starts an action, a menu's only opens a list (§7 asks for an outline Button in both) | Dialog, Menu, Tabs and DetailRow panic on a blank name, tested by message |
| 9 | The confirmation's click listener skips checkboxes and radios, whose click is the action | Valid | They ask on click |
| 10 | A toast parsed before `#ui-toasts` exists never moves | Rejected: the script is a module, so its elements are defined after the document is parsed, when the region exists. A page without a region keeps its toasts where they were rendered | — |
| 11 | The triggers' placed attributes aren't owned | Rejected: the default trigger takes no caller attributes, and a fill spreads the value on a button it owns (§2.2) | — |
| 12 | The 250ms timer polls | Rejected: it's what makes finding 4's fix safe, costs one comparison per toast, and stops with the toast | — |
