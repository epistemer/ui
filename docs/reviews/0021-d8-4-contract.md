# Review 0021: the contract draft of the remote select, uploads and list fields (D8.4)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 76k tokens, in one run with handlers' ADR 0006 (handlers' review 0007 records its two findings, 14 and 15) |
| Under review | [rich-inputs-interface.md](../rich-inputs-interface.md), with the design's §3.1–§3.2, §4.2–§4.5 and §9, forms' locked §5.3–§5.5, §5.8 and §5.10, and handlers' §4.2 and §12.1 |
| Scope | Signatures that won't compile or form import cycles; contradictions with the locked text and forms' invariants; gaps an implementer would have to guess |

16 findings concerned this draft. Claude checked each; 15 were valid
and changed it. One was valid in part.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | `MinFiles`/`MaxFiles` reused forms' list errors: `rule` can't import forms, and the locked texts say "add", not "choose" | Valid | `ErrMinFiles`, `ErrMaxFiles` in `rule` (§2.4) |
| 2 | `inputs` can't tell an integer remote field from a text one, for `inputmode` | Valid | `Control.Attrs` gives `inputmode="numeric"` for an integer remote field (§2.1) |
| 3 | The upgrade kept `inputmode` on the search box, could blank unconverted text, and left the chosen span's id in `aria-describedby` | Valid (the spike already did the last two; the contract didn't say) | §3.1 and §6 say what the upgrade removes and keeps |
| 4 | An upload field's presence, its value before `Bind`, `Set`/`Options` on it, and several files on a single `Upload` were undefined | Valid | Presence row, value before `Bind`, panics, first file taken (§2.2) |
| 5 | A handler that re-renders on an oversized body answers 422, swapping the files away; the 413 the script relies on never comes | Valid | `Submit` wraps the `*http.MaxBytesError`; the handler returns it (§2.2, design §4.5) |
| 6 | A `Labeler` that isn't comparable panics as a map key in the cache | Valid | Such a `Labeler` isn't cached (§3) |
| 7 | `Previews` named `File`/`Files`, which rendered without the upload shape: no checks, no progress, no 413 handling on a create form | Valid | All four file kinds render the upload shape; `data-ui-previews` (§3.1) |
| 8 | "A new list holds `MinItems` blank items" left its interaction with `Add` open | Valid | The control yields blank items; the list itself is unchanged (§2.3) |
| 9 | `data-ui-texts`' keys per element were unnamed; two `%s` in one text could be confused by a `%s` in a file name | Valid | §7 names each key; the server fills what it knows; one verb per text, filled in one pass |
| 10 | `Lost` missed a `Fail` after `Bind` and fired on a field that failed itself | Valid | Evaluated when called, only without an error of its own (§2.2) |
| 11 | The chip and the filter field ask `LabelOf` twice; a table with a remote filter needs a gate; nil arguments to `Column.Remote` | Valid | Said so (§5); nil panics |
| 12 | Nested templates aren't reached by `querySelectorAll`; an item of only unchecked checkboxes posts nothing and vanishes | Valid in part: §6 already rewrites "at every depth", and the spike tested a nested list. The second half was valid | A bare item marker that forms counts, L7, a change to locked §5.8 (§2.3) |
| 13 | A send error was read as "too large" (it fires when the network is gone); a file the system gives no type was refused; `rule.Files` didn't pass its inner rules' attributes on | Valid | 413 only; an untyped file defers to the server; `Files` passes attributes on (§2.4, §6; the spike changed too) |
| 16 | `Control.Stored()` and `Text()` on other kinds were unspecified | Valid | An empty iterator, never nil; `Text()` nil for uploads (§2.2) |
| 17 | The search link's own query, and that a handler fetches one more than the limit | Valid | `URLSearchParams` (§6); `SuggestionLimit`'s comment (§4) |
| 18 | A link inside the keep checkbox's label; `Image` with no `Href` | Valid | The label holds text only, the link sits beside it; no thumbnail without an `Href` (§3.1) |
