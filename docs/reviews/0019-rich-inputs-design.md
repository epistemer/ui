# Review 0019: the design of the remote select, uploads and list fields (D8.1)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 69k tokens |
| Under review | The first draft of [rich-inputs-design.md](../rich-inputs-design.md) and ADRs 0011–0013, as a diff, with excerpts of the locked text they must fit (forms interface §5.4, §5.5, §5.8, §5.10; forms' invariants and design §10.3; visual-interface §6.1; ADR 0005's custom elements; the actions script's request handling; models' `Labeler`; handlers' `WithMaxBody`) |
| Scope | Contradictions with the locked contracts and forms' invariants; security; accessibility against the APG combobox; cloning; what breaks without script; htmx; defaults that are wrong |

There were 18 findings. Claude checked each against the draft and the
code it cites (`template.content` isn't reached by `querySelectorAll`;
forms gives both levels of a nested list `__i__`; an unbound list with
`MinItems(1)` holds no items, all checked). All 18 changed the text.
Nothing is built, so the browser behaviours are also on the spike's list
(design §8).

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | A clone of an outer list doesn't reach the inner `<template>`'s content, and a nested `<ui-repeat>` keeps its old `data-ui-path` | Valid | The clone walk enters every nested template's content and rewrites the prefix in `data-ui-path` too (§5.2, ADR 0013) |
| 2 | `Upload` in a list row has no `Set` value, since items are built during `Bind`; and a POST handler that forgets `Set` drops every kept key, which reads as "delete every file" | Valid | The stored files are an argument of `Form.Upload(name, stored)`, so they can't be forgotten; `Upload` in a list panics (R26) (§4.2, ADR 0012) |
| 3 | Text that didn't convert (`abc` for an `int64`) reached `LabelOf`, whose SQL fails: user input as a 500 | Valid | `Control.Remote` gives the converted id, or none; only that is looked up. A `Labeler` must answer `false` for an id that can't name a row (§3.2, §6) |
| 4 | A refused search link made the field read-only in markup only: a crafted post still changes it, and a required field on a create form could never be filled | Valid | Refused, the field is the shape without script; the gate governs searching, the handler what it accepts (R7, §3.2) |
| 5 | After a 422, the keep checkboxes forgot what was unchecked; a single `Upload` with a new file and a checked keep was ambiguous; a repeated key counted twice | Valid | The control reports each stored file's posted keep state; an added file empties `Kept` in the value only; keys are deduplicated (§4.2) |
| 6 | Option ids rendered by the search handler collide between two comboboxes on a page | Valid | The answer carries values and texts only; the script builds options with ids from its own input's (§3.3) |
| 7 | The script's texts (results, no matches, item legends, remove labels, size errors) had no translated source | Valid | `data-ui-texts` on each element, rendered by the server; Remove labels renumber with legends (§3.2, §4.3, §5.1–§5.3) |
| 8 | `fetch` follows a sign-in redirect and would put a whole page in the listbox | Valid | A redirected answer, or one whose root isn't `ul[data-ui-suggestions]`, is a failed search (§3.3) |
| 9 | APG faults: a non-option line in the listbox; no way to clear by keyboard; the clear button a Tab stop per field; Enter during the debounce submits a cleared value; a click blurs before it chooses | Valid | The "more" line moves to the status; empty text clears on Tab; the clear button is `tabindex="-1"`; Enter never submits uncommitted text; `pointerdown` is prevented in the popup (§3.4) |
| 10 | Setting a hidden input's value fires no event, so conditions never see a choice; the visible input's events reached the form | Valid | Bubbling `input` and `change` on the hidden input at each commit; the visible input's events stop at the element (§3.4) |
| 11 | Esc in a combobox inside the modal dialog closes the dialog | Valid | Esc stops at the combobox while the popup is open or the text is uncommitted, and passes on otherwise (§3.4) |
| 12 | Progress showed on fields with no files and announced "Uploaded" before the server answered, and the status element didn't exist | Valid | Progress only on fields with files, hidden at the answer, never announced; the field has a status for chosen files (§4.3) |
| 13 | `hx-preserve` matches by id, which a cloned and renumbered row doesn't keep, and leaves stale attributes; the note had no way to know files were lost | Valid | `hx-preserve` is rejected; `Control.Lost()` drives a note that isn't an error (§4.4, R20) |
| 14 | Clone details: focus landing on a hidden input; a condition naming several fields; space-separated id lists; legends of nested groups renumbered | Valid | Focus goes to the first visible, enabled control; every occurrence of the prefix is replaced; id lists are rewritten token by token; renumbering is scoped to the list's own items (§5.2, §5.3) |
| 15 | `Remote` didn't panic on kinds it can't serve or with `Hint`; `MinItems` on `Upload` isn't typed (I4); `data-ui-max-size` had no source in `Control.Attrs` | Valid | `Remote` only on text and integer fields, panicking otherwise and with `Hint`; `rule.MinFiles`, `MaxFiles`; `MaxSize` adds the attribute (§3.1, §4.2, §6) |
| 16 | Up to 256 sequential `LabelOf` calls in a list, repeated ids included; the upgrade flashes the id | Valid | A per-request memo by `Labeler` and id is the default (R10); the flash is measured in the spike, with a CSS fallback (§3.2, §8) |
| 17 | A stored file's link and thumbnail weren't asked of the gate (principle 3) | Valid | `inputs` asks it; refused, the name shows as text (§4.3) |
| 18 | R24's "call `List.Add()`" workaround doesn't work while binding validates; a new form's list with `MinItems(1)` shows no row | Valid | The claim is gone; forms fills a new list to `MinItems` (R27, §5.1) |
