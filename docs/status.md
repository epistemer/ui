# Status log

Newest entry first. Each working session adds one: what landed, decisions
taken for epistemer, what's next, and what's waiting on epistemer.


## 2026-11-22 — D6: the visual packages' interface is locked

epistemer locked [visual-interface.md](visual-interface.md) with the
defaults of V1–V18. Those include:
- `…Spec` builders;
- `popovertarget` menus;
- keys `ui.<package>.<component>.<name>`;
- no `Action` type until D7;
- `Neutral` plus five accent tones;
- Lucide icons;
- the drawer for narrow screens;
- the `ToastSwap` exception to ADR 0005, still to be written into the ADR
  (V5).

uitest's `CheckNames` and `CheckSelectors` are amendment A4. The document
stays separate from interface.md, whose §10 points to it.

The roadmap gains milestones V1–V7: theme, catalogue and checks, icons and
elements, inputs, components, layout, release v0.2.0.

**Next.** V1, `ui/theme`.
## 2026-11-22 — Interface draft of the visual packages (D5)

**Landed** (this PR):
- docs/visual-interface.md: the interface of theme (with the nested
  `themeschema` module), icons, elements, inputs, components, layout and
  the catalogue, the ui script's contract and ui's i18n keys, for
  epistemer to lock at D6. On lock it folds into interface.md;
  interface.md itself is unchanged;
- review 0005;
- the roadmap's D5 ticked.

**What the draft settles** (each open choice is a question, V1–V18, with a
default):
- builders return `…Spec` values (finding 7);
- `theme.Config`'s getters carry the group prefix (`ThemeBrandColor()`),
  pinned with `GoName("Theme")`, and theme exports `CheckColor`,
  `CheckFont` and `CheckRadius` for cfgx's `.Validate`;
- ui's keys are `ui.<package>.<component>.<name>`; the spike's keys are
  renamed;
- no `Action` type in this lock (Q6): confirmation is attributes, and the
  table (D7) brings one action description with placements;
- uitest's `CheckNames` and `CheckSelectors` are amendment A4, proposed;
- forms' `Placeholder` fallback: no change proposed (V15).

Every piece the spike didn't prove is marked, and §14 lists them: tabs,
toasts, the detail list, layout, most elements, `themeschema` and
`catalogtest` among them.

**Review 0005** (Gemini 4 Argon (High), about 97k tokens, the milestone's
one Argon (High) run; Gemini 3.8 Flash (Low) restated three cut findings):
18 findings, 5 blocking. All handled, two answered rather than taken
(file kinds are forms' file input, not D7's uploads; V15's default stays).
The ones that changed the design:
- the toast region is a manual popover, so it shows above a modal
  dialog, and each toast is a `<ui-toast>` element;
- the app shell's narrow-screen drawer is driven by the ui script through
  `aria-expanded`, not a popover;
- an unknown asset release is a 404 again, as spiked: serving the current
  files would mix modules of two releases;
- a `Neutral` tone, so a neutral Cancel button can be expressed;
- a required slot left unfilled panics.

**Waiting on epistemer:** the D6 lock of visual-interface.md, with
answers to V1–V18 or their defaults.

## 2026-11-22 — Guidance cross-check of the spike

epistemer upgraded Node to 24, so the `modern-web-guidance` tool now runs.
Its guides for menus, dropdown placement, dialogs, dark mode, validation
and toasts were checked against the spike (visual-design.md §11,
finding 20). Most of the spike's choices hold. Four corrections go to D5:
- Browser support: anchor positioning has limited availability (Firefox
  lacks it), and popover is only newly available. ADR 0004's figures are
  amended.
- Menus: `commandfor` triggers and native `focusgroup`, and a fallback
  placement that flips and clamps.
- Fields: hints above the control, and an "Error" text alternative on
  error messages.
- Dialogs: scroll containment, and `autofocus` on the confirmation's
  Cancel.

## 2026-11-22 — Design of the visual packages (D1–D3)

**Landed** (this PR):
- docs/ui-inventory.md (D1);
- ADR 0004 (ui's own CSS) and ADR 0005 (client behaviour, proposed);
- docs/visual-design.md (D3 draft);
- review 0002;
- the roadmap's D track (D1–D7).

**Decided by epistemer (2026-11-21).** After a brainstorm on four axes
(volume, atomic granularity, splitting logic from UI, a configurable
theme), ui ships its own CSS, written in modern CSS, with no Tailwind and no
Flowbite. Claude's reading of the other axes is in visual-design.md §2:
- coverage instead of volume;
- layers with one dependency rule instead of atomic design's vocabulary;
- splitting logic from the view only where there is logic;
- only tokens as configuration.

**Storybook or our own catalogue.** templ ships `templ/storybook`, which
runs Storybook's server mode. Claude recommends our own catalogue
(visual-design.md §8, Q8):
- `templ/storybook` brings Node back (`npx sb init`, `npm run
  build-storybook`).
- Its arguments are only text, booleans, ints, floats and objects. ui's
  components take `i18n.Ter`, enums, slots and context values, so every
  story would need a hand-written wrapper anyway.
- It adds `rs/cors` and `x/mod` to go.mod.
- Storybook's server mode is a less maintained corner of Storybook:
  `@storybook/server-webpack5` is at 9.1.x, with no word on version 10.
- Our own registry drives goldens, id and name checks inside `go test`.

**Review 0002** (Gemini 4 Argon (High), about 71k tokens; Argon was out of
cooldown): 19 findings. Two were rejected after checking:
- Generated ids are process-wide, so htmx swaps don't collide.
- Depending on models' interfaces is epistemer's chosen direction.

Six blocking findings changed the design:
- Tailwind v4's Preflight would beat ui's layers unless ui sits after
  `base`.
- A CSS-only popover fallback can't anchor in the top layer.
- HTML keeps the first of two duplicate attributes, so behaviour
  attributes are merged in Go.
- A rendered `aria-expanded` goes stale.
- Per-row confirm dialogs break inside menus and forms; one shared dialog
  replaces them.
- `embed` can't reach sibling packages, so all CSS lives in `theme/css/`.

Q2's default moved from variadic options to builders, as in forms.

**Tools.**
- The `modern-web-guidance` skill can't run here: its package needs Node 22
  or newer, and this machine has Node 20.20.2 (snap). The browser-support
  claims in ADR 0004 come from web searches: invoker commands reached
  Baseline in December 2025, anchor positioning in January 2026 (Firefox
  147).
- `templ/storybook` was read in the module cache (v0.3.1118).

**Decided by epistemer (2026-11-22),** each on its default:
- Q2: components take options through builders, as in forms;
- Q3: variants in markup are data attributes;
- Q4: the accent colours, font families and radius scale can change at run
  time;
- Q7: ADR 0005 is accepted.

Q1, Q5, Q6, Q8 and Q9 stand on their defaults, open to change until the
D6 lock.

**D4 part 1** (PR #17, Claude):
- **theme:** the token table; OKLab and WCAG maths; accent partners derived
  for contrast; the stylesheet joined in layers under one hash; the
  run-time theme stylesheet; ADR 0004's CSS checks.
- **elements:** the Button builder and attribute merging.
- **a11y:** the names and selector checks.
- **catalog**, and Button stories.

Findings are in visual-design.md §11. The two that changed the design:
- Shades are derived in Go, away from the on-colour.
- Mixing is in OKLab, not OKLCh.

Checked in the browser pane:
- LTR, RTL, light and dark;
- RTL mirroring;
- live re-theming under a strict CSP, with no violations.

**Review 0003** (Gemini 4 Argon (Medium), about 71k tokens): 20 findings,
all handled. The main ones:
- the catalogue's unescaped URLs;
- a possible panic in `Derive` on admin input, now replaced by a stepping
  loop and proven by a sweep of 5,832 colours;
- disabled links still reaching htmx;
- gaps in the CSS checker and the names check;
- the selector check comparing classes and attributes separately.

**D4 part 2** (PR #18, Claude):
- **inputs:** a field rendered from `forms.Control`: label, help and error
  wired by generated ids, `aria-invalid`, `dir="ltr"` for email, the
  condition handed to the script.
- **components:** Dialog with slots and `DialogClose`, the shared
  confirmation, Menu with `<ui-menu>`.
- **ui script:** the condition evaluator, the menu, the confirmation.
- **internal/jstest** (ADR 0006): runs the evaluator against forms' 21
  vectors through goja, in a module of its own.

Checked in the browser:
- conditions showing and hiding fields;
- the menu's keyboard, focus and anchoring in LTR, RTL and inserted rows;
- Dialog's open and close, with focus returning to the trigger;
- the confirmation: cancel; and confirm, which submits once.

Bugs found that way, all fixed (visual-design.md §11): `this.popover` on a
custom element, a select without an empty option, and a confirmation
waiting on a deferred `close` event.

**Review 0004** (Gemini 4 Argon (High), about 80k tokens): 14 findings,
plus a verdict that §11's conclusions needed correcting before D5. Two
were rejected after checking:
- htmx forms don't double-confirm;
- relative units are allowed by ADR 0004.

The rest were fixed and checked again in the browser:
- the confirmation rebuilt as one capture-and-replay path;
- `refresh` no longer enabling what the app disabled;
- an htmx swap inside a form now refreshed;
- groups as fieldsets that carry their conditions;
- a three-state toggle as a Yes/No select;
- `multiple` for files, and the server's ticked rule for checkboxes;
- a textarea's leading newline kept;
- disabled menu items, arrow keys on the trigger, and typeahead.

Claude's own reading of forms' bind code found one more gap, which the
reviewer missed: the client read `["fixed", " "]` as absent and the server
as "fixed". Two vectors were added, and both evaluators pass them.

D4 is done.

**For epistemer (API):** forms' `Control.Placeholder` falls back to the
label (finding 12), so a renderer can't tell an explicit placeholder from
none. ui works around it by comparing texts. Exposing whether one was set
would be a forms amendment.

**Next.** D5: interface.md sections for what Q9 locks, built from the
spike, with a cross-model review.

## 2026-11-21 — v0.1.0

**Landed.** PR #13 (F6.1), PR #14 (R.1), PR #15 (release review fixes),
the CHANGELOG, the tag, and the global `ui` skill. Every roadmap item to
v0.1.0 is done.

- **F6.1** (Gemini 3.8 Flash High, about 205k tokens; reviewed by Claude,
  one snippet fixed): forms godoc examples that run as tests, the README
  guide for declaring and handling a form, `forms/rule/CONTRIBUTING.md`.
- **R.1** (Claude): docs/api-review.md. All 228 exported identifiers are in
  interface.md, every func signature matches, every identifier is
  exercised by a test; a new test pins §5.10's default kinds.
- **Release review** (Gemini 4 Neon XHigh, about 261k tokens; Argon was in
  cooldown, so the strongest available ran): 19 findings, two rejected
  after checking (flushing does reach templ's buffer; the root package has
  its doc.go). The rest fixed with tests and recorded as **amendment A3**:
  junk list keys at any depth, a POST whose query net/http refuses,
  checkbox state before Bind, an unreadable Submit and sub-forms, Enum with
  foreign options, Options kinds, root When, NaN and negative bounds,
  `*/*`, `Matches`' pattern portability, `URL` host names, uitest on whole
  pages and ids with white space, the `-update` flag, collect-pass order.
- **Models this session** (agy): Gemini 3.8 Flash (High and Medium) for
  delegated leaf items and small reviews, Gemini 4 Neon (Medium, High,
  XHigh) standing in for Argon, which was in quota cooldown all session.

**Waiting on epistemer.** Amendments A1–A3 (§10) are clarifications Claude
made as owner; epistemer may overturn any of them. Nothing else.

**Next.** v0.1.x fixes as they come; the next ui packages (icons,
elements, inputs, components, layout) start from their own designs.

## 2026-11-21 — Forms F5 (nesting, Control, formstest) and kernel M5.1 (docs)

**Landed.** PR #11 (F5) and PR #12 (M5.1). The kernel is feature-complete
for v0.1.0 (M1–M5.1); forms has F1–F5.

- **F5** (Claude). A form is a tree of fields, groups and lists. Groups
  and lists post under paths; lists bind sparse indices, renumber, honour
  limits, and expose a template row at `__i__`. Activity and cycle checks
  follow container chains; conditions can't reach into a list's items
  from outside. `Control` is the typed-free view for renderers; formstest
  has `Submit`, `Errors`, `AssertLabelled`. examples/shop has the
  promotion form of forms-design §11.2 (conditions, two groups, a list
  with conditions inside items).
- **F5 review** (Gemini 4 Neon High, about 86k tokens): junk list indices
  could trigger `ErrMaxItems` and push out real items; the building flag
  that lets a list declare items after Bind covered the whole form; list
  items accepted `When` and ignored it; an inactive group kept its error;
  `AssertLabelled` skipped empty lists; a template row inherited its
  list's inactivity, so a renderer would emit it disabled. All fixed with
  tests, recorded as **amendment A2** (clarifications; epistemer may
  overturn them). staticcheck caught an unused test variable in CI.
- **M5.1** (Gemini 3.8 Flash High, about 277k tokens; reviewed by
  Claude): godoc examples that run as tests for Provider, ID, slots and
  uitest; the package overview; README guides for writing and filling a
  slotted component, with HTML from a real run. Claude corrected one name
  (`Provider.Provide`).

**Next.** F6 (forms docs: godoc examples, README guide, the rule
contributor note), then R.1 (API review against interface.md) and R.2
(CHANGELOG, v0.1.0, the `ui` skill).

## 2026-11-21 — Kernel M4 (slots) and forms F4 (conditions)

**Landed.** PR #9 (F4) and PR #10 (M4). The kernel's milestones M1–M4 are
complete; forms has F1–F4.

- **M4** (Claude: M4.1–M4.4). `NewSlot`, `NewSlotOf`, `Slotted`, `Place`,
  `Filled`, `Value`; lexical lookups in fill content (rule 5); the five
  errors, each naming the slot's declaring line; `ErrInBlock` from
  `Provide`, `Scope` and `Place` in a block. Go reports a package-level
  initializer's line as `<autogenerated>:1` when a cross-package call is
  inlined into it, so `NewSlot` and `NewSlotOf` are `//go:noinline`.
- **M4.5–M4.6** (Gemini 3.8 Flash High, about 278k tokens; reviewed by
  Claude): shop components with goldens and `CheckIDs` (cart modal,
  dropdown, an order table whose rows put their actions in a dropdown's
  fill and target their own keyed ids, a product card dropping an empty
  footer), and benchmarks below. Allocations per render do not grow with
  the fill: 26 for a small fill and for one 100× larger.
- **M4 review** (Gemini 4 Neon High, about 77k tokens): templ buffers a
  block's output, so stray content before another collect-pass error was
  never seen, against §9.1's first-error rule. The kernel now flushes the
  block's buffer before reporting its own errors. Also: the sink's first
  stray content and a block that drops a write error, nil and zero slots,
  the declaring frame matched by import path. All fixed with tests.
- **F4** (Claude). `Cond`, `When`, cycle detection, the wire format, 21
  conformance vectors in `forms/testdata/cond` for a client evaluator.
- **F4 review** (Gemini 4 Neon High, about 60k tokens): a rule could read a
  later inactive field's stale value (now every inactive field is cleared
  before any rule); a checkbox's `Eq` disagreed with `Filled` for an "on"
  submission (both now read its value); cycle checks were exponential on
  chains of `In` (now linear). All fixed with tests.

**M4.6 benchmarks** (2026-11-21, Intel(R) Core(TM) i5-5300U CPU @ 2.30GHz)

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

**Next.** Kernel M5.1 (docs and README guides) and forms F5 (groups,
lists, Control, formstest), then F6 and the v0.1.0 release items.

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
