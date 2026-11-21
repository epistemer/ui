# Roadmap to v0.1.0

The module reaches v0.1.0 with its first two packages: the kernel (root
package `ui`, plus `ui/uitest`) and forms (`ui/forms`, `ui/forms/rule`,
`ui/forms/formstest`). M0 is shared. After it, the kernel's milestones
(M1–M5) and forms' (F1–F6) are independent tracks: forms imports no templ,
and the kernel imports nothing of forms. The module's v0.1.0 waits for both.

The designs are [brief.md](brief.md) (kernel) and
[forms-design.md](forms-design.md) (forms); the contract is
[interface.md](interface.md), locked by epistemer at the end of M0.

## How to read this

- **Ownership.** Claude owns ui until v1.0.0 on the terms of
  [ownership.md](ownership.md) (delegated 2026-11-21).
- **IDs.** Every item has an ID (M2.1, F3.2) and fits one PR or one focused
  session. Branch names, PR titles and commit subjects start with the ID.
- **Interface first.** M0 ends with the lock of interface.md. After that, a
  change to what a component author, a form author or a caller writes is an
  amendment.
- **Versions.** The first usable release is v0.1.0. v1.0.0 is epistemer's
  call; it is gated on neozp alone (ui/inputs and ui/components built on the
  kernel and forms without a breaking amendment), never on moving zp.
- **Agents.** Claude owns the core (M1.1, M2.1, M4.1–M4.4, F2, F4, F5) and
  reviews Gemini's work. Gemini, through agy, takes leaf items with a crisp
  done-when (M2.3, M3.2, M3.3, M4.5, M4.6, F1.2, F6.1). Each delegated item
  runs in its own git worktree outside the repository.

**Definition of done, for every item.**

1. `go test -race ./...` passes on the CI matrix (Go 1.27 on Linux, macOS and
   Windows), and `templ generate` leaves no diff.
2. It lands as a PR with green CI and a review by the other model family.
3. If the behaviour differs from interface.md, that document is amended in
   the same PR.
4. A new dependency needs an ADR.
5. Nothing of zp's names enters the repository: examples and tests use a toy
   shop.

## M0 — Foundations (shared)

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M0.1 | Module `github.com/epistemer/ui` (Go 1.27, MIT), doc.go, CLAUDE.md, CI with a `templ generate` drift check, and an `examples/shop` module with its own go.mod | CI runs green on a PR |
| ☑ | M0.2 | brief.md, forms-design.md, this roadmap, ADR 0001 (kernel primitives), ADR 0002 (forms model), interface.md for both packages, status.md, ownership.md; cross-model review of interface.md | epistemer locks the interface |
| ☑ | M0.3 | templ contract tests against the templ version pinned in go.mod, and forms' compiler spike (forms-design §13.1 step 2) | Two templ tests pass (a Go component that skips ClearChildren leaks its block, one that clears does not; a block of only component calls, `if` and `for` writes nothing but white space). The spike's results and any fallback are in status.md and interface.md |

## Kernel

### M1 — Context values

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M1.1 | `Provider[T]`: NewProvider, Provide, With, Value | Table tests cover the default, nesting (the inner value wins, and the next sibling sees the outer value again), With from Go code, a value read three components down, and two concurrent renders with different values under -race |
| ☑ | M1.2 | The internal block helper every kernel component uses for K3 | A component that renders `{ children... }`, called without a block inside Provide, renders nothing |

### M2 — Element IDs

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M2.1 | NewID and its name check, Scope, Get, Part | Tests cover nesting, siblings getting different values, Get outside a scope panicking with the id's name, and a label and input rendering matching `for` and `id` |
| ☑ | M2.2 | IDSource: the process-wide default, Sequence, WithIDs | 64 goroutines drawing 10,000 ids each under -race produce no duplicate, and Sequence output is byte-stable across runs |
| ☑ | M2.3 | Key and part escaping | A fuzz test shows every output is a valid CSS identifier, decoding gives the parts back, and no keyed id equals a generated one |

### M3 — Test kit

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M3.1 | Render, RenderErr, Context | The M1 and M2 tests use them |
| ☑ | M3.2 | The normaliser and Golden with -update; ADR 0003 accepts golang.org/x/net/html for uitest only | A golden test fails on a changed attribute, shows it as one changed line, and passes after -update |
| ☑ | M3.3 | CheckIDs and External | Fixtures with a duplicate id, a `for` that points nowhere and a broken hx-target each fail with a message naming the attribute and the id; a correct fixture passes |

### M4 — Slots

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M4.1 | NewSlot, NewSlotOf, Slotted, Place, filling, Value and Filled: rules 1 to 4 and 6 | A test counts executions: the block runs once, a fill once per placement, a fallback only when its slot is unfilled. A fill that renders nothing still replaces its fallback |
| ☑ | M4.2 | Lexical slot lookups, rule 5 | The shop's order row places its actions slot inside a dropdown menu's fill, and a modal nested in another modal's body renders each body in its own dialog |
| ☑ | M4.3 | The errors and their messages | One test per row of the error table, asserting the sentinel with errors.Is and the declaring file and line in the message |
| ☑ | M4.4 | ErrInBlock in Provide, Scope and Place, rule 7 | Each of the three returns ErrInBlock when written directly in a block, and reaches the fills when opened around the component call |
| ☑ | M4.5 | Shop examples with goldens: cart modal, dropdown, order table with row actions, and a product card that uses Filled to drop an empty footer | The goldens pass, and uitest.CheckIDs passes on every example |
| ☑ | M4.6 | Benchmarks: a component with three slots, and slotted components nested five deep | The numbers are in status.md, and the kernel's allocations per render do not grow with the size of the fills |

### M5 — Kernel docs

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | M5.1 | Godoc examples, and README guides: writing a slotted component, and filling one | The examples run as tests, and the README's output is pasted from a real run |

## Forms

### F1 — Rules (`ui/forms/rule`)

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | F1.1 | Rule[T], New, Err, Check, Attrs, and the declared default errors (`errors.forms.rule.*`) | Table tests; errors.Is finds each default; i18ntest shows a translated default |
| ☑ | F1.2 | The v0.1 catalogue (forms-design §6.2), with the data behind Email, Phone, PostCode, CountryCode, CurrencyCode | Table tests from the salvaged cases pass; `testdata/nocompile` proves a rule on the wrong type fails to build |

### F2 — Core

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | F2.1 | Form, Field[T], Source (FromRequest, FromValues), the non-generic constructors, presence, Required, Is, Set, SetPtr, Value, Get, Ptr, Fail | Every row of the presence table passes; the declaration-error panics name the field |
| ☑ | F2.2 | Bind, Validate, Submit, Check, FieldError, Err, Errors, forms' declared errors (`errors.forms.*`) | The store example (forms-design §11.1) passes through formstest; redisplay of `abc` in a number field |

### F3 — Conversion

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | F3.1 | Conv and As, with the dead-handle check | Rules before and after As check T and U; the old handle panics |
| ☑ | F3.2 | Number, Parsed (Enum and Options landed with F2, which the store example needed) | A models-style enum binds; `All<Type>Options()` passes to Options unchanged; decimal precision round-trips in examples/shop |

### F4 — Conditions

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | F4.1 | Cond, Eq, In, Filled, All, Any, Not, When, evaluation and cycle detection | A required field under a false When is not required; a stale value is cleared; a cycle panics at the closing When |
| ☑ | F4.2 | MarshalJSON in zp's wire format, and the conformance vectors in `testdata/cond/` | The Go evaluator passes every vector |

### F5 — Nesting and view

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | F5.1 | Group and List: path binding, sparse indices, renumbering, limits, the template row | Indices 0 and 3 bind as two items renumbered 0 and 1; item 257 is refused |
| ☑ | F5.2 | Control, Kind, Choice, Attrs | inputs-style exhaustive switch over Kind compiles in a test; the promotion example (§11.2) passes |
| ☑ | F5.3 | formstest, including the unlabelled-control check | Both examples' handler paths run through formstest |

### F6 — Forms docs

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☐ | F6.1 | Godoc examples, README guide: declaring a form and handling it, and the rule contributor note (adding a rule and its error) | The examples run as tests; `go list -deps ./forms/...` shows no templ, web framework or decimal |

## R — Release v0.1.0

| ✓ | ID | Item | Done when |
|---|---|---|---|
| ☑ | R.1 | API review of every exported identifier of both packages against interface.md, in docs/api-review.md | A written review lists each identifier and how it was checked, and its findings are fixed or filed |
| ☐ | R.2 | CHANGELOG, the v0.1.0 tag, and a global Claude Code skill (`~/.claude/skills/ui/`) covering the kernel and forms | `go get github.com/epistemer/ui@v0.1.0` works through the module proxy |

## Backlog

- **A default slot for loose content** (brief §8, decision 6). Gate:
  epistemer decides it is wanted and a real component in ui/components is
  clumsy without it. The late-collect design keeps K4 but buffers from the
  layout's first hole, an exception to K5. Running the block twice is
  rejected: it breaks K4.
- **Attribute merging**, `Merge(attrs ...templ.Attributes)`. Gate: two ui
  packages need it; until then it lives in ui/elements.
- **Condition operators** gt, lt, gte, lte, when a real form needs one.
- **A time zone for Date and Time**, when neozp carries one per request.
- **A recovering wrapper in handlers** so a render panic becomes a 500 page
  (brief §8). handlers' API, so epistemer's decision.
