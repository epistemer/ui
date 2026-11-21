# Status log

Newest entry first. Each working session adds one: what landed, decisions
taken for epistemer, what's next, and what's waiting on epistemer.

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
