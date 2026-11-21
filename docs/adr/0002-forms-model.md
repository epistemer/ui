# ADR 0002: Forms as typed field handles, in the ui module, on models' options

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-21 |
| Roadmap item | M0.2 |
| Deciders | epistemer and Claude designed it on 2026-11-16 (forms-design.md); on 2026-11-21 epistemer moved it into the ui module, confirmed built-in rule errors, adopted option.SelectOption, and accepted the other defaults |

## Context

zp's forms have five design flaws (forms-design §2): a chain that executes
instead of describing, every rule on every type, no codec, cross-field
conditions written twice in two languages, and compound fields that are not
forms. An abandoned rewrite fixed three of them but made one struct per
value type and put rendering inside the input. Since the first draft,
models v0.3.0 made `option.SelectOption` the interface UI packages consume
(models ADR 0006), errors fixed its key convention (`errors.<area>.<name>`),
and handlers moved admin handlers to `net/http`.

## Options

1. **Code generation** from a schema or struct tags. Rejected by epistemer:
   no generator, no struct tags.
2. **One struct per value type** with its own rule methods (the abandoned
   rewrite). Rules compile on the right types only, but the API doubles for
   optional values and grows per type.
3. **One generic `Field[T]`** registered on a form at declaration, with
   typed rule values (`rule.Rule[T]`), presence as state, a conversion
   chain changed with generic methods (`As[U]`, Go 1.27), conditions as
   typed values serialised for the client, groups and lists as sub-forms
   posting path names, and a non-generic `Control` view for widgets.
4. **A repository of its own** (`epistemer/forms`) against **a package in
   the ui module** (`ui/forms`).

## Decision

Option 3, as `github.com/epistemer/ui/forms` with `forms/rule` and
`forms/formstest`, at Go 1.27. Catalogue rules carry declared errors
(`errors.forms.rule.*`), overridable with `.Err(err)`. Choices are
`option.SelectOption`: `Enum[E]` takes `E` constrained by `~string` plus
`SelectOption`, and `Field.Options` takes `SelectOption` values, so models'
`All<Type>Options()` passes in unchanged.

## Consequences

- forms imports the standard library, errors, i18n and models/option; no
  templ, no web framework, no decimal. The examples module carries decimal.
- The whole module needs Go 1.27, so the kernel's CI drops Go 1.26.
- A compiler spike (M0.3) proves the generic-method signatures before F1;
  any fallback (for example a package-level `forms.Parsed[T](f, name)`) is
  recorded in status.md and interface.md.
- ui/inputs consumes `Control`, renders conditions with a client evaluator
  that must pass forms' conformance vectors, and takes `listing.Labeler`
  for remote selects.
- A JSON source is dropped: handlers' API validates JSON through
  oapi-codegen.
