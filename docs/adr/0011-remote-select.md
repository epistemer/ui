# ADR 0011: a remote select is declared on its field and searches through htmx

| | |
|---|---|
| Status | Accepted by epistemer on 2026-11-22 (R1, R2, R4), with the search through htmx rather than the proposed `fetch` |
| Date | 2026-11-22 |
| Roadmap item | D8.1 |
| Deciders | Claude proposed it (design R1, R2, R4); epistemer decided R1 and R2 as proposed and R4 for htmx |

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
- htmx makes the search: `hx-get` on `<ui-combobox>`, triggered by the
  script's `ui-search` event with htmx's own delay and `hx-sync`; the
  script adds `q`, vetoes and aborts stale requests through htmx's
  events, and gives the swapped options their ids. The answer is
  `components.Suggestions`, `hx-select`ed. Without htmx the field isn't
  upgraded. (Proposed instead: a `fetch` of the script's own, which the
  spike tried; epistemer chose htmx.)
- Without script it is a text input of the id with the name beside it.

## Consequences

- A remote select works in groups, list rows and table filters with no
  per-page wiring, and a clone sets itself up.
- forms' locked contract gains a kind, a builder and a control method, and
  imports models' `listing` (standard library only).
- The remote select stays on ADR 0005's list of what emits `hx-*`; its
  requests target the listbox, which the actions and table scripts don't
  claim.
- `<ui-repeat>` calls `htmx.process` on a clone when htmx is there.
- `LabelOf` is memoized per request; a batch method waits for a
  measurement.
- The gate on the search link governs searching only; what a form
  accepts stays its handler's check.
