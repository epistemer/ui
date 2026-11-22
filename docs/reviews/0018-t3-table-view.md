# Review 0018: T3, the table's rendering

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 36k tokens |
| Under review | `TableOf`, `Pagination`, their templates, `table.js` and the table's CSS, with server-state-interface.md §4.1, §6 and §7 |
| Scope | Markup against the contract, htmx attributes, accessibility, the script's events, the CSS rules |

There were 13 findings. Claude checked each against the code and the pane;
nine changed it.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | A state form's answer swaps the whole `<ui-table>`, and the new element skips its status and focus | Valid | Focus is kept by table id across the swap; a table swapped in whole settles as new results; checked in the pane |
| 2 | Row and bulk actions mark the results busy, and it sticks | Valid | Only state requests (to the table or its results) mark them |
| 3 | Parameters may be sent twice by path and parameters | Rejected: a state control sends its own parameters one way only, a link by its address, the form and the box by their fields | — |
| 4 | The actions header's span lacks the contract's id | Valid | `<table>-actions-label` |
| 5 | The bulk bar's selector matches any checkbox with a form | Valid | The row cells' checkboxes only |
| 6 | A row can render a checkbox or a menu it lacks | Rejected: both are the table's, not a row's (`Bulk`, `Actions`), so every row has them or none | — |
| 7 | `marked` checks the text's runes but not the query's, nor invalid UTF-8 | Valid | Both checked; tested |
| 8 | Chips, the clear link, size links and empty-state links have no ids, so focus is lost after them | Valid | Keyed ids; a gone control falls back to the scroll region, else the search; the Apply button keyed too |
| 9 | The pager's table id comes from its target by a suffix | Rejected: a pager's target is always its table's results id | — |
| 10 | Sort and page links keep their underlines | Valid | None |
| 11 | The status region is `display: none` when empty, which can silence it | Valid | Visually hidden always, never removed from rendering |
| 12 | The refresh doesn't guard an empty address | Valid | Guarded |
| 13 | A redundant `formmethod` | Rejected: harmless, and explicit for methods a form can't send | — |
