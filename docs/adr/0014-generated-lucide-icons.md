# ADR 0014: Generated Lucide icons with `IconSpec`

| | |
|---|---|
| Status | Accepted by epistemer on 2026-11-24, with the lock of A9 (Q3, Q4, Q6, Q7, Q9) |
| Date | 2026-11-24 |
| Roadmap item | C5 |
| Deciders | epistemer took the defaults of Q3, Q4, Q6, Q7 and Q9 at the A9 lock; Claude proposed the mechanism |

Supersedes [ADR 0007](0007-lucide-icons.md).

## Context

Confab iteration 1 (finding F20, amendment A9 §5) needs Lucide icons beyond
the 22 hand-copied in `icons/icons.templ`, plus sizes, semantic colours,
stroke width, accessible labels and merged attributes. Hand-copying
hundreds of SVGs does not scale, fetching npm at `go generate` or CI time
would tie offline builds to the network, and generating 2,410 `.templ`
functions would produce roughly 100,000 lines of generated template code.

## Options

1. **Generate `.templ` files for every Lucide icon.** Matches ADR 0007's
   pattern, but produces ~100,000 generated lines for an identical `<svg>`
   wrapper and cannot share one runtime `Render` method for options.
2. **Generate Go functions returning `IconSpec` from vendored, validated shapes.**
   `icons/internal/gen -fetch <version>` downloads `lucide-static` from npm,
   verifies its `sha512` integrity and archive safety, strictly parses every
   SVG with `encoding/xml` against an element and attribute allowlist, and
   vendors only the serialised shapes in `icons/internal/lucide/shapes.txt`
   (plus `LICENSE`, `VERSION`, and `kept.txt` for dropped names).
   `go generate ./icons` runs offline from those vendored files to write
   `icons/lucide.go` and `internal/iconlist/iconlist.go`.
3. **Fetch icons dynamically at build or `go generate` time.** Keeps no
   vendored file, but breaks offline builds and CI's drift check.

## Decision

Option 2, as Q3, Q4, Q6, Q7 and Q9 decided:

- Pin `lucide-static` **1.77.0** (vendored checked shapes, `LICENSE`, and
  `VERSION`, with `kept.txt` preserving any name removed in a later pin as
  `// Deprecated:`).
- Each generated function returns an immutable `IconSpec` (a
  `templ.Component`) with `Size`, `Stroke`, `Color`, `Label` and `Attrs`
  methods, rendered by one shared `IconSpec.Render` in `icons/icons.go`.
- `Size` (`SizeText`, `SizeS`, `SizeM`, `SizeL`) and `Color` (`ColorCurrent`,
  `ColorMuted`, `ColorBrand`, `ColorDanger`, `ColorWarning`, `ColorSuccess`,
  `ColorInfo`) emit `data-size` and `data-color` attributes styled in CSS,
  with no inline styles.
- Left/right pairs outside the fixed exclusion list (`pilcrow-*`,
  `decimals-arrow-*`, `mouse-*`, `toggle-*`) also generate `…Start`/`…End`
  functions with `data-mirror`, while `…Left`/`…Right` do not mirror.
- `More` (`ellipsis`) and `Warning` (`triangle-alert`) stay as hand-written
  aliases; `Labelled` stays for wrapping any component.

## Consequences

- Every Lucide icon is available with tree-shaking by the Go linker, and
  rendering with no options emits the exact markup of ADR 0007 so no
  existing golden changes.
- Updating the Lucide pin requires running `icons/internal/gen -fetch` by
  hand and reviewing the diff of `shapes.txt` and `kept.txt`; CI verifies
  that `go generate ./icons` leaves no diff offline.
- Callers that previously assigned an icon function to `func() templ.Component`
  or inferred a variable from an icon before assigning another
  `templ.Component` must adapt to the `IconSpec` return type.
