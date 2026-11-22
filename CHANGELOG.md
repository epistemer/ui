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
- `ui/catalog`: stories shown in LTR and RTL, light and dark, with a width
  switcher and a live token panel, on relative URLs so it mounts anywhere
  (§9).
- `ui/catalog/catalogtest`: `Check` runs a golden, `CheckIDs`,
  `CheckNames` and `CheckSelectors` on every variant, then the reverse
  selector check on the app's CSS (§9.1).
- uitest: `CheckNames` and `CheckSelectors` with `Prefix` (amendment A4).
- `go run ./cmd/catalog` serves ui's own stories.
- `ui/icons`: 22 icons from Lucide (ISC, ADR 0007) and `Labelled` (§4).
- `ui/elements`: `Button` (with `Label` and the `Neutral` tone), `Badge`,
  `Alert`, `Card` with its slots, `Avatar`, `Spinner`, `Tooltip` with
  `TooltipTrigger`, `Separator`, `VerticalSeparator`, `Heading`, the value
  renderers (`Text`, `Bool`, `Date`, `DateTime`, `Enum`, `Link`, `Image`,
  `Amount`, `Empty`), `Merge`, and the `Variant`, `Size` and `Tone` types
  (§2.2, §2.3, §5).
- The ui script hides a tooltip on Esc and sets `<html data-ui-ready>`.
- `ui/inputs`: `Field(c forms.Control)` renders a field for every kind
  but lists (§6):
  - help above the control, and errors led by an icon and a hidden
    "Error:";
  - checkbox and password rows;
  - the password reveal;
  - groups as fieldsets that carry their condition, help and error.

  The ui script reveals passwords, and refreshes conditions when a page
  is restored from the back-forward cache.
- `ui/components` (§7):
  - `Dialog` with its slots and `DialogClose`;
  - `Confirm` and `ConfirmDialog`;
  - `Menu` and `MenuItem` (`Href`, `Disabled`, `Danger`, `Submit`);
  - `Tabs` and `Tab`;
  - `Toasts`, `Toast` and `ToastSwap`;
  - `DetailList` and `DetailRow`.

  The ui script adds `<ui-toasts>` and `<ui-toast>`, which move toasts
  into an open modal dialog and back. Menus are placed by script where
  anchor positioning is missing, flipping and clamping to the viewport.
- `ui/layout` (§8):
  - `Document`, with `lang` and `dir` from the request's locale, and
    `ColorScheme` with `ColorSchemeOf`;
  - `App` with its slots, `NavItem`, `NavSection` and `ColorSchemeMenu`;
  - `Auth` with its slots;
  - `PageHeader` with `Crumb`.

  The ui script opens and closes the app's navigation drawer on narrow
  screens, and keeps the colour scheme picked in the cookie
  `ui-color-scheme`.

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
