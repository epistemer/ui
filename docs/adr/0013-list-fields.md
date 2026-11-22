# ADR 0013: `<ui-repeat>` clones forms' template row, owning only its own index

| | |
|---|---|
| Status | Proposed |
| Date | 2026-11-22 |
| Roadmap item | D8.1 |
| Deciders | Claude proposed it (design R23–R25); ADR 0005 already chose the element; epistemer decides at the lock (D8.4) |

## Context

forms posts a list as `list[i].field` and exposes a template row at
`__i__` (interface §5.8). ADR 0005 chose a custom element that clones it
with fresh ids. Two facts shape the rest: forms gives a nested list's
template the same placeholder at both levels
(`orders[__i__].lines[__i__].sku`), and forms sorts and renumbers indices
on bind, so gaps are harmless.

## Decision

- `inputs.Field` renders a list as nested fieldsets, a `<template>` of the
  template row, and Add and Remove buttons the script reveals.
- A clone replaces only its own list's prefix (`lines[__i__]` →
  `lines[2]`), everywhere it occurs and inside nested `<template>`
  content too, so a nested template keeps its own placeholder and learns
  its new path; ids at this level get a fresh suffix and every reference
  to them is rewritten, token by token. Indices only grow.
- Focus goes to the new item's first field on add, to a neighbour on
  remove, and a status line says what happened. Min and max make the
  buttons unavailable with a reason. Rows don't move.

## Consequences

- Lists edit without script but add and remove only with it.
- Behaviours in a clone (a remote select) set themselves up; nothing in a
  clone needs htmx to be processed.
