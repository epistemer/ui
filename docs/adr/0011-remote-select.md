# ADR 0011: a remote select is declared on its field and searches with the ui script's own request

| | |
|---|---|
| Status | Proposed |
| Date | 2026-11-22 |
| Roadmap item | D8.1 |
| Deciders | Claude proposed it (design R1, R2, R4); epistemer decides at the lock (D8.4) |

## Context

A remote select holds a record's id, shows its name, and searches the
server as people type ([rich-inputs-design.md](../rich-inputs-design.md)
§1). forms-design §10.3 put the search link and models' `Labeler` on the
widget call, not on the field. But `inputs.Field` renders groups, list
rows and a table's filter form generically, so a widget whose arguments
the generic path can't pass can't appear in a list row or a filter. And
ADR 0005 named htmx for the search, while the actions script already
treats every htmx request as a possible action (a 422 swaps, an opener is
remembered, a pending refocus is dropped) and htmx 2.0.11 doesn't set up
`hx-*` on nodes another script inserts.

## Decision

- The field declares it: `Field[T].Remote(search, labeler)` gives
  `KindRemote`, and the control hands both to a renderer. forms carries
  the `Labeler` and never calls it; `inputs` calls it once at render to
  show the chosen name.
- `<ui-combobox>` fetches suggestions itself, debounced and cancelled,
  from the search link with `q`. The answer is HTML from
  `components.Suggestions` over models' `option.SelectOption`; the script
  reads values and texts from it and builds its own options, so no markup
  from an answer reaches the page.
- Without script it is a text input of the id with the name beside it.

## Consequences

- A remote select works in groups, list rows and table filters with no
  per-page wiring, and a clone sets itself up.
- forms' locked contract gains a kind, a builder and a control method, and
  imports models' `listing` (standard library only).
- The remote select leaves ADR 0005's list of what emits `hx-*`.
- `LabelOf` is memoized per request; a batch method waits for a
  measurement.
- The gate on the search link governs searching only; what a form
  accepts stays its handler's check.
