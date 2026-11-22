# Review 0022: the search through htmx, and L7 (D8 answers)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 47k tokens |
| Under review | The diff recording epistemer's first answers: the remote select's search rewritten for htmx (R4) in the design §3.3, ADR 0011 and the contract §3.1, §4 and §6, and L7 (no item marker), with the actions and table scripts' request handling |

9 findings. Claude checked each against the scripts and htmx 2.0.11's
documented behaviour; seven were valid, one wasn't, one is out of reach.
None was run: the search through htmx isn't spiked, and the build's
first milestone tries it.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | With `hx-select`, an empty answer and a page that isn't an answer both swap nothing and read "No matches"; the actions script forces a 422 to swap | Valid | The element's own `beforeSwap` listener stops the event and needs status 200 and `data-ui-suggestions` in the body |
| 2 | `xhr.responseURL` is absolute and carries `q`, so comparing it to the address asked fails every time | Valid | Origin and path are compared; a refused answer is skipped by the script's after-swap step |
| 3 | htmx sends the combobox's own hidden input with the search | Valid | `hx-params="none"`; `q` is added in `configRequest`, which runs after the filter |
| 4 | `htmx:abort` doesn't cancel htmx's `delay` timer, so a search starts after a choice, and its `beforeRequest` drops the actions script's pending refocus | Valid | The script debounces itself and triggers `ui-search` without `delay` |
| 5 | L7's text named checkboxes only; a multi-select with nothing chosen and controls a condition disabled post nothing too, and a dropped row's rules never run | Valid | The contract's §2.3 says exactly which controls, and that the row's rules don't run. The decision itself is epistemer's, put to them again with this |
| 6 | `table.js` settles every table after any swap | Not valid | `settled()` returns at once when a table's results are the same element, so another swap changes nothing |
| 7 | A combobox inside action buttons or a menu would be claimed by `isAction` | Out of reach (no form field sits there); a one-line guard goes into the build | — |
| 8 | Moving the `data-ui-more` item "into the status" put an `li` in a `p` | Valid | The item is removed and the status takes the `more` text |
| 9 | The order of rewriting, appending and `htmx.process` was loose | Valid | Rewritten detached, appended, then processed (§6) |
