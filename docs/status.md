# Status log

Newest entry first. Each working session adds one: what landed, decisions
taken for epistemer, what's next, and what's waiting on epistemer.

## 2026-11-21 — Kernel M3 and forms F3

**Landed.** PR #7 (F3, conversion) and PR #8 (M3, the test kit).

- **F3.** `Conv`, `As`, `Number`, `Parsed`. Rules run stage by stage, and
  an earlier stage's rule wins over a later conversion error. `Number`
  picks its parser once per field (no reflection per call). examples/shop
  wraps `Parsed` as `money.Decimal`; 0.10000000000000000001 binds exactly.
- **M3.** `Render`, `RenderErr` and `Context` (Claude); ADR 0003 accepts
  golang.org/x/net/html for uitest only, and a test checks the kernel
  still links none of it. `Golden` with its normaliser and `CheckIDs` were
  delegated to Gemini 3.8 Flash (High), about 327k tokens; Claude reviewed
  them and made the normaliser escape text, so text that looks like markup
  never reads as an element.
- **Review** of F3 and M3.1: Gemini 4 Neon (Medium), about 33k tokens. A
  nil interface value between conversion stages could panic on user input;
  `As` after `Set` desynchronised value and text. Both fixed, with the
  tests it asked for; `Render` now derives from `t.Context()`.

**Next.** Kernel M4 (slots), the kernel's largest milestone, and forms F4
(conditions).

## 2026-11-21 — Kernel M2 and forms F2

**Landed.** PR #5 (M2, element ids) and PR #6 (F2, the forms core).

- **M2.** `NewID`, `Scope`, `Get`, `Part`, `Key`, injective escaping,
  `IDSource`, `Sequence`, `WithIDs` and the process source (6 random
  base-36 characters, rejection-sampled, and a base-36 counter). 640,000
  ids from 64 goroutines without a duplicate under -race. An exported
  `ErrBadToken` was drafted and removed before review: the locked
  interface has none, so a bad token is a plain error naming the source.
  M2.3's fuzz tests were written by Gemini 3.8 Flash (Medium, about 110k
  tokens) and strengthened by Claude after review (strict decoding, two
  independent parts, hyphenated names).
- **M2 review**: Gemini 4 Neon (Medium), about 36k tokens. No critical
  finding; a duplicate id drew tokens before it was refused, the random
  prefix had a modulo bias, and tests lacked K3/K8 cases. All fixed.
- **F2.** `Form`, `Field[T]`, `Source`, the non-generic constructors, and
  `Enum` and `Options`, pulled forward from F3.2 because the store example
  needs them. `examples/shop` has a hand-written stand-in for generated
  models and the store form of forms-design §11.1, tested through its
  create, invalid, service-error and edit paths and in two locales. The
  handlers in the example redirect by hand: wiring them to handlers'
  generated routes is for the F6 example.
- **F2 review**: Gemini 4 Neon (High), about 93k tokens. Ten findings, all
  valid. The serious one: `Options()` with an empty list switched
  membership checking off, so any value passed. Also: conversion errors
  appeared only at Validate, an `Enum` let rules read an unchecked value,
  `Submit` didn't refuse a second call when the request was unreadable.
  All fixed with tests.
- A force-push after a rebase was blocked by the session's safety checks;
  branches now take main by merge.

**Models.** Argon stayed in quota cooldown all session; Neon (High) and
Neon (Medium) stood in for Argon (High) and Argon (Medium).

**Next.** Kernel M3 (test kit) and forms F3 (conversion: `Conv`, `As`,
`Number`, `Parsed`).

## 2026-11-21 — Forms F1: rules

**Landed.** PR #3 (F1.1) and PR #4 (F1.2). `ui/forms/rule` is complete for
v0.1: `Rule[T]`, `New`, `Err`, `Check`, `Attrs`, the twenty declared
defaults, and the whole catalogue of interface §6.2.

- **Amendment A1** (clarification, Claude): error texts follow errors'
  convention, lower case with no final period. epistemer may overturn it.
- `NotBefore`/`NotAfter` were written by Claude with an internal hook
  (`forms/internal/hook.DateOnly`) through which forms makes them compare
  calendar dates on a Date field and emit date attributes (§9.4).
- F1.2 was delegated to Gemini 3.8 Flash (High) in a worktree outside the
  repository, about 229k tokens; zp's data was read from a scratch clone,
  so zp itself stayed untouched. Claude reviewed it: correct, no private
  names; Claude tightened the no-compile test so each case names the
  compiler error it expects.
- F1.1's review (Gemini 3.8 Flash High, about 187k tokens; Argon still in
  cooldown): the date-only hook kept datetime attributes, errors showed the
  bound in its own zone, and `run` used a type assertion. All fixed, with
  tests for every default error.

**Next.** Kernel M2 (element ids) and forms F2 (the core), in parallel.

## 2026-11-21 — M1: context values

**Landed** (PR #2). `Provider[T]` (NewProvider, Provide, With, Value) and
the internal `block` helper that every kernel component uses for K3.

- Values are boxed in the context, so a provided nil interface value wins
  over the default (found by review).
- A padding byte keeps providers of zero-sized types distinct (K8).
- Review: Gemini 3.8 Flash (High), about 193k tokens (Argon in cooldown).
  Five findings: the nil-interface bug and three missing tests were fixed;
  "missing package" was wrong (it is on main); `ErrInBlock` in `Provide` is
  M4.4's, once the collect pass exists.

**Next.** Forms F1 (rules), then kernel M2 (element ids).

## 2026-11-21 — M0: the repository, the designs, the interface draft

**Before starting.** The two designs existed only as Google Docs ("neozp ui
kernel roadmap", 2026-11-17; "neozp forms API design", 2026-11-16). Checked
against neozp as it stands:

- Still true: Go 1.27.4 is installed and its spec allows type parameters on
  methods; templ v0.3.1118 is the latest release; auth's CSRF needs no token
  on a form (auth D13); errors' `Key`/`Key1`, `Invalid` (400), `T`, `All`,
  `KindOf` are as forms uses them; nothing in neozp carries a time zone.
- Drift, now fixed in the docs (brief §9, forms-design §15): forms moves
  into this module; forms adopts models' `option.SelectOption` (models
  ADR 0006); error keys follow `errors.<area>.<name>`; the forms examples use
  handlers' `net/http` admin handlers instead of echo; the JSON source is
  dropped (handlers' API validates through oapi-codegen); the kernel's
  v1.0.0 is no longer gated on zp; the kernel's claim that `net/http` turns
  a panic into a 500 was wrong (it aborts the connection).
- zp's file names and identifiers were removed from the forms design, which
  is now public; the salvage guide to zp's files lives in the maintainers'
  private notes.

**Decided by epistemer (2026-11-21).**

1. One module, `github.com/epistemer/ui`, Go 1.27: kernel = root package
   `ui` (+ `ui/uitest`), forms = `ui/forms` (+ `rule`, `formstest`).
2. The repository is delegated to Claude until v1.0.0 on the terms of
   ownership.md; interface.md's lock and API design stay epistemer's.
3. Catalogue rules carry declared errors, overridable with `.Err(err)`.
4. forms consumes `option.SelectOption`; its own `Option[T]` goes.
5. The other defaults of both designs are accepted (brief §8,
   forms-design §14).

**Landed.**

- M0.1: go.mod (Go 1.27, templ v0.3.1118), LICENSE, doc.go, CLAUDE.md, CI
  (gofmt, `templ generate` drift, build, vet, staticcheck, tidy,
  `go test -race` on Linux, macOS and Windows), `examples/shop` module.
- M0.2 (docs): brief.md, forms-design.md, roadmap.md (kernel M1–M5 and
  forms F1–F6 as independent tracks after M0), ADR 0001 (kernel primitives),
  ADR 0002 (forms model), ownership.md, interface.md (draft).
- M0.3: `internal/templcontract` pins K3 (a Go component that renders its
  block without `ClearChildren` leaks it to a children-rendering component
  inside; one that clears does not) and the collect pass's premise (a block
  of only calls, `if` and `for` writes only white space). Both pass on
  v0.3.1118.
- M0.3 compiler spike (Go 1.27.4, throwaway module, not committed): all
  compile — `As[U]` on `Field[T]`; `f.Number[int32]("n")` promoted through
  an embedded `*Form`; `PT` inferred in `Parsed[Dec]`; `Int().Is(Min(0))`;
  `is := f.Type.Eq`; `Enum` over `~string` + SelectOption; `Group` inferring
  `G`. Refused as intended: `Bool().Is(MaxLen(5))`. Also refused:
  `Number[int32]().Is(Min(0))` — a bound's type comes from its argument, so
  int32 fields write `rule.Min[int32](1)`. No fallback needed; interface §5.3
  records it.

- Cross-model review of interface.md ([review 0001](reviews/0001-interface-draft.md)):
  Gemini 4 Neon (High), about 71k tokens — Argon was in quota cooldown for
  4.6 hours, so the strongest available Gemini ran. 18 findings, all valid;
  resolved as interface §9 (behaviour in detail) plus fixes in §1, §5.1,
  §5.3, §5.7, §6.2 and §8. Two claims confirmed by a second spike: `Parsed`
  needs the wider constraint to accept `big.Int`, and `rule.MinItems(1)`
  cannot infer `T`.

**Lock.** epistemer locked interface.md on 2026-11-21, including the items
marked *(new)*, §9 and `Parsed`'s wider constraint. From here, a change to
what an app writes is an amendment (interface §10).

**Next.** Kernel M1 and forms F1, in parallel.

**Was waiting on epistemer** (now done): the lock of interface.md. Its new items, not in
either design, are marked *(new)*: formstest's API (§7), the error keys and
texts of the catalogue and of forms (§5.9, §6.3), and the condition
operator names on the wire (§5.7). Also new since the designs: §9's
behaviour in detail, and `Parsed`'s wider constraint (§5.3).

### M4.6 benchmarks (2026-11-21, Intel(R) Core(TM) i5-5300U CPU @ 2.30GHz)

```
goos: linux
goarch: amd64
pkg: github.com/epistemer/ui
cpu: Intel(R) Core(TM) i5-5300U CPU @ 2.30GHz
BenchmarkThreeSlots-4   	  403431	      4749 ns/op	    1833 B/op	      40 allocs/op
BenchmarkThreeSlots-4   	  307603	      3820 ns/op	    1833 B/op	      40 allocs/op
BenchmarkThreeSlots-4   	  367142	      3088 ns/op	    1833 B/op	      40 allocs/op
BenchmarkNestedFive-4   	  104952	     14609 ns/op	    5468 B/op	     118 allocs/op
BenchmarkNestedFive-4   	   96424	     15421 ns/op	    5468 B/op	     118 allocs/op
BenchmarkNestedFive-4   	   77715	     16352 ns/op	    5469 B/op	     118 allocs/op
BenchmarkFillSize/small-4         	  457353	      2311 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/small-4         	  564014	      2783 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/small-4         	  527125	      2556 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  349038	      3696 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  329400	      3462 ns/op	    1209 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  341342	      4285 ns/op	    1209 B/op	      26 allocs/op
PASS
ok  	github.com/epistemer/ui	16.413s
```
