# Changelog

All notable changes to this module. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).


## Unreleased

### Added
- `ui/theme`: the token table, `Config` with `Defaults` and the checks
  `CheckColor`, `CheckFont` and `CheckRadius` (errors `errors.theme.*`),
  and `Theme`. `Theme` serves ui's stylesheet and script under one release
  hash, and the run-time theme whose accent colours are derived for
  contrast. Its `Head` links both (visual-interface.md §3).
- `github.com/epistemer/ui/theme/themeschema`, a nested module: the cfgx
  group `theme`, whose generated settings satisfy `theme.Config`.
## [v0.1.0] - 2026-11-21

The first release: the kernel and forms, the whole of the interface locked
on 2026-11-21 with amendments A1–A3.

### Added

- **Kernel (`ui`).** `Provider[T]` for typed values a subtree reads.
  `ID` with `Scope`, `Get` and `Part` for generated ids unique across
  separately rendered fragments, and `Key` for stable keyed ids; `IDSource`,
  `Sequence` and `WithIDs`. Named slots: `NewSlot`, `NewSlotOf[T]`,
  `Slotted`, `Place`, `Filled`, `Value`, with a strict block (only fills,
  run once), lexical lookups in fill content, no buffering, and the errors
  `ErrStrayContent`, `ErrFilledTwice`, `ErrNotDeclared`, `ErrNoComponent`,
  `ErrInBlock`.
- **Test kit (`ui/uitest`).** `Render`, `RenderErr` and `Context` with
  deterministic ids; `Golden` with a readable normaliser and `-update`;
  `CheckIDs` and `External` for id references (labels, aria, href, htmx).
- **Forms (`ui/forms`).** `Form` and `Field[T]` declared once as typed
  handles; `Source` from a request or values; constructors for text,
  numbers (`Number[T]`), booleans, checkboxes, dates, times, files, enums
  (models' `option.SelectOption`) and text marshallers (`Parsed[T]`);
  presence as state; conversion stages (`Conv`, `As`); conditions (`Cond`,
  `When`) shared with the client through a JSON wire format and
  conformance vectors; groups and lists posting path names; `FieldError`
  and declared errors; `Control` and `Kind` for renderers.
- **Rules (`ui/forms/rule`).** `Rule[T]`, `New`, `Err`, `Check` and a
  catalogue for strings, ordered values, decimals, times, files and slices,
  each with a declared, translatable default error.
- **`ui/forms/formstest`.** `Submit`, `Errors`, `AssertLabelled`.
- Examples in `examples/shop`: slotted components with goldens, and the
  store and promotion forms.

[v0.1.0]: https://github.com/epistemer/ui/releases/tag/v0.1.0
