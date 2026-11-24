# ADR 0007: ui's icons are Lucide's paths, vendored

| | |
|---|---|
| Status | Superseded by 0014 |
| Date | 2026-11-22 |
| Roadmap item | V3 |
| Deciders | epistemer took V10's default at the D6 lock; Claude, as owner, records how |

## Context

The visual packages need about 20 icons (visual-interface.md §4): what
the components draw, plus a few for the layout and the auth pages. ui
adds no dependency for them and no Node toolchain, and every icon must be
inline SVG in `currentColor`, so it follows the text around it.

## Options

1. **Draw our own.** Full control, but drawing a consistent set is design
   work, and the result looks like no other set.
2. **Vendor an open set's paths.** Lucide (ISC) is a maintained set on a
   24-unit grid with a two-unit stroke, whose shapes are plain `path`,
   `circle` and `line` elements.
3. **Depend on an icon package.** Every Go package of icons is a
   dependency for a few strings.

## Decision

Option 2, Lucide, as V10's default.
- The paths are copied into `icons/icons.templ` with Lucide's stroke
  attributes, as `svg.ui-icon` with `aria-hidden="true"`. The four
  directional icons carry `data-mirror`.
- Lucide's licence sits beside them in `icons/LICENSE.lucide`, and the
  package doc names it.
- Sources, 2026-11-22: the lucide-icons/lucide repository's `icons/`, and
  lucide-static v1.77.0 for `triangle-alert`, which the repository's
  `icons/` path no longer served.

## Consequences

- Adding an icon means copying one more of Lucide's files and keeping its
  attributes. A change to an existing icon's look is a copy from a newer
  Lucide, checked in the catalogue.
- An app's own icons follow the same contract: a `templ.Component` that
  renders `svg.ui-icon`.
