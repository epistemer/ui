# ADR 0001: The kernel's three primitives, strict slots and process-wide ids

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-21 |
| Roadmap item | M0.2 |
| Deciders | epistemer and Claude designed it on 2026-11-17 (brief.md); epistemer accepted the defaults of brief §8 with the plan of 2026-11-21 |

## Context

Every ui package needs to pass values down a subtree, wire elements together
by id, and let callers put inline content into named places of a component.
zp's answers needed a root component per request, re-ran a caller's block
once per slot, made loose content the fallback of every empty slot, and
could not let a fill remove a default (brief §7.2). templ already covers
once-per-page content, nonces, fragments, streaming and classes, and passes
children through the context as an immutable value that a Go component must
clear itself (brief §3).

## Options

1. **A root component per request** that holds mutable render state (zp's
   way). Simple lookups, but every page, fragment, SSE event and test needs
   the root, and a forgotten one fails at run time.
2. **Immutable context values keyed by handles, with no root** (K1, K2,
   K8). Providers, ID scopes and slot fills each live in the context; ids
   come from one process-wide source unless a test installs its own.
3. **For slots: loose content as the default slot** (zp's way) against **a
   strict block** that holds only fills, collected once into a sink, then a
   single unbuffered layout pass (brief §7.1).

## Decision

Option 2, with strict slots: the kernel is `Provider[T]`, `ID` (generated
and keyed ids, process-wide default source, `Get` panics outside a scope),
and `Slot`/`SlotOf[T]` with `Slotted` and `Place`, under invariants K1–K8.
A default slot for loose content stays in the backlog.

## Consequences

- Components render the same in a page, a fragment, an SSE event and a
  test, with no per-request setup.
- A fill always wins over its fallback, the block runs once, and nothing on
  the render path buffers.
- Callers write a little more for one-area components (plain children
  instead of a slot); README shows the pattern.
- Generated ids differ per render, so tests use `ui.Sequence` through
  uitest and goldens are normalised.
- M0.3 pins templ's children behaviour; it runs first after every templ
  upgrade.
