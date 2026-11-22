# Changelog

All notable changes to this module. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).


## [Unreleased]

### Added
- `ui/forms/files` (U1, amendment A7): `Stored` and `Upload`.
- `ui/forms` (U1, amendment A7): `Form.Upload` (`KindUpload`) and
  `Form.Uploads` (`KindUploads`), `Field[T].Remote` (`KindRemote`),
  `Control.Remote`, `Control.Stored`, `Control.Lost` and `Control.Bounds`;
  `Control.Children` on a new list yields `MinItems` blank items.
- `ui/forms/rule` (U1, amendment A7): `Files`, `MinFiles` (`ErrMinFiles`),
  `MaxFiles` (`ErrMaxFiles`), and `MaxSize`'s `data-ui-max-size` attribute.
- `ui/inputs` (U2, amendment A7): `WithLabelCache`, `FieldSpec.Previews`,
  and `Field` rendering for `KindRemote` (`<ui-combobox>` with htmx search
  attributes and `LabelOf` lookup), `KindUpload`/`KindUploads` (and the
  upload shape for `KindFile`/`KindFiles`), and `KindList` (`<ui-repeat>`
  with its item groups and `<template>` row).
- `ui/components` (U2, amendment A7): `Suggestion`, `SuggestionLimit` and
  `Suggestions`, with CSS for comboboxes, uploads and list fields.
- The ui script (U3, amendment A7): `<ui-combobox>` (`theme/js/combobox.js`,
  searching through htmx with script-side debouncing, `htmx:beforeSwap`
  origin/path/suggestions verification, keyboard and focus rules, and
  `.ui-combobox-clear`), `<ui-repeat>` (`theme/js/repeat.js`, cloning
  template rows, rewriting own path prefixes and suffixing ids at every
  depth, calling `htmx.process`, renumbering, and enforcing bounds with
  `aria-disabled`), and upload behaviours (`theme/js/upload.js`, keep cards,
  chosen file list with `DataTransfer` removal and optional `blob:` image
  previews, client size/type checks with a capturing submit guard, upload
  progress from `xhr.upload`, and 413 handling).
- `ui/tables` (U4, amendment A7): `Column.Remote(search, labeler)` for a
  remote-select filter on a `Text` or `Integer` column with `listing.Eq`
  on its typed value, `View.Err` for propagating a remote filter's
  `LabelOf` failure (checked by `components.TableOf.Page`), and chip
  resolution through `LabelOf` (once per link filter per request for a
  valid state; miss falls back to the id).
- `internal/toyshop` (U4): a `customer_id` remote filter on the orders
  table with `GET /shop/customers/search`, plus the order form at
  `GET/POST /shop/orders/new` (customer remote select, receipt `Uploads`
  with stored files served at `GET /shop/files/{key}`, repeatable `lines`
  list with product remote select via `GET /shop/products/search`,
  1.5 MiB `http.MaxBytesHandler` returning 413, and `inputs.WithLabelCache`).

### Changed
- `inputs.Field` renders `KindList` with `<ui-repeat>` instead of returning
  an error, and renders `KindFile` and `KindFiles` in the `.ui-upload`
  drop-zone shape.
- `Form.List` builds its template row when declared, not on first use, so a mistake in an item's build function (such as `Upload` in a list) panics at declaration (I10). A build function runs once more, at declaration, and must not read the list's own handle.

### Fixed
- `ui/inputs`: `WithLabelCache` now checks cacheability dynamically with `reflect.ValueOf(l).Comparable()` (avoiding panics on comparable static types holding non-comparable dynamic values) and never caches failed `LabelOf` calls.
- `ui/inputs`: `Field` now checks `actions.Allowed` before `links.Href` for a remote field's search link and a stored file's non-nil `Href`, so an empty or sanitised URL cannot hide a missing `actions.Gate`.
- `ui/inputs` and `theme/js/upload.js`: stored upload items carry `data-ui-name`, which the keep-card button reads for `"Remove %s"` even when a file has no link.
- `ui/forms`: single-file `Upload` now keeps the last posted file, matching `File`, and clearing an inactive conditional upload field resets its keep map so every stored file re-renders kept.
- `theme/js/upload.js`: the submit guard inspects `form.elements` and ignores `:disabled` file inputs; keep cards sync on checkbox `change` and form `reset` (dispatched from the toggle button); `fitsAccept` strips MIME parameters and defers untyped files unless every accept entry is an extension; htmx progress, 413 and bar-hiding listeners resolve `form`, ignore `<ui-combobox>` and GET requests, skip `:disabled` file inputs, clear `bodyTooLarge` errors on `htmx:beforeSend` and file input `change`, scope drag highlights to `.ui-upload-input`, and set up a swapped root matching `.ui-upload`.
- `theme/js/combobox.js`: `htmx:beforeSwap` refuses stale or debounced answers silently, while `htmx:responseError` and `htmx:sendError` stop propagation, ignore non-pending requests and avoid announcing a response `beforeSwap` already refused; `#restore()` restores the hidden input, syncs the clear button and fires `input`+`change` when its value changed; ArrowDown/ArrowUp wait on an in-flight search instead of opening stale options and Escape cancels pending searches; the form `reset` listener is removed in `disconnectedCallback` via `AbortController`.
- `theme/js/repeat.js`: `add()` falls back to dispatching `input` on the enclosing form's first named control when a cloned row has no non-file named control, before focusing.


## [v0.3.0] - 2026-11-22

Actions and tables, amendment A6 (server-state-interface.md, locked
2026-11-22), milestones T1–T3. ui now requires models v0.4.0; route
links come from handlers v0.2.0 (`handlers.Link`, `handlers.Gate`), which
ui doesn't import. `theme/themeschema` is unchanged and stays at v0.2.0.

### Added
- `ui/actions`: `New` and its builders, `To`, `Unless`, `If`, `Resolve`,
  `Item`, `Gate`, `AllowAll`, `Allowed`. An action is absent when the
  request's gate refuses its link, and unavailable with a reason when the
  record's state forbids it.
- `ui/actions/actionstest`: `Context`, `Refuse`, `Assert`, `Unavailable`.
- `ui/tables` (T2): `New`, `Col` and its options, `Table` (`Label`,
  `RowLabel`, `Sort`, `Sizes`, `Unpaged`, `Prefixed`, `Actions`, `Bulk`,
  `Check`, `ID`, `RowID`, `Selected`, `SelectionName`, `Bind`, `Row`),
  `State` (a `listing.Source`; `Valid`, `Errors`, `Answer`, `Page`,
  `Rejected`), `Targeted`, and the view's types. On models v0.4.0's
  `column`.
- `ui/tables/tablestest`: `Bind`, `Query`, `Errors`, `Check`.
- `tables.ErrRange` (`errors.tables.range`), a range filter whose start
  is after its end.
- `ui/components` (T3): `TableOf` (`Cell`, `Attrs`, `Page`, `Row`) and
  `Pagination` (`Attrs`), with their CSS.
- The ui script (T3): `<ui-table>` (the header checkbox, the count of
  checked rows, status and focus after a swap), state requests built from
  the live address, and `Reply`'s refresh.
- `cmd/catalog` serves the toy shop's tables and actions at `/shop/orders`.
- `ui/components`: `ActionButtons`, `ActionMenu` (`ID`, `LabelledBy`),
  `ActionForm`, `ActionDialog`, `ActionTitle`, `InActionDialog`, and
  `Reply` (`Toast`, `Replace`, `Remove`, `Refresh`, `Fail`, `Else`, `Go`,
  `ErrNotLocal`).
- `ButtonSpec.Unavailable` and `MenuItemSpec.Unavailable`: aria-disabled,
  focusable, inert, described by a reason. Menus' arrow keys stop on them.
- `MenuSpec.ID`, a keyed id for a menu's trigger.
- The ui script: a 422 swaps; the action dialog opens with what is loaded
  into it, is named by its `ActionTitle`, and gives focus back after the
  answer.

### Changed
- `layout.App` and `layout.Auth` render `ActionForm` and `ActionDialog`
  beside the confirmation and the toasts.

### Fixed
- `forms.FromRequest` (and so `Submit`) bound a body that something had
  already parsed (`r.ParseForm`, a middleware) as a blank form. It reads
  `r.PostForm` or `r.MultipartForm` then.
- The ui script lost a toast that arrived as the modal dialog it was
  going into closed. It goes to the page's region instead.

## [v0.2.0] - 2026-11-22

The visual packages, to the contract locked on 2026-11-22
(visual-interface.md), with the kernel's amendments A4 (uitest) and A5
(links). ui ships its own
CSS and script (ADRs 0004 and 0005); a nested module,
`theme/themeschema/v0.2.0`, carries the cfgx group.

### Added
- The kernel's `Link`, an alias of `interface{ URL() string }`, and
  `URL(s)` for a literal or external address (amendment A5, ADR 0008).
  Everything in the visual packages that takes an address takes a
  `ui.Link` (`ButtonSpec.Href`, `AvatarSpec.Src`, `elements.Link`,
  `elements.Image`, `MenuItemSpec.Href`, `Tab`, `NavItem`, `Crumb.Href`)
  and renders it through `templ.URL`. A nil link, an empty one, or one
  `templ.URL` refuses is no address (visual-interface.md §2.7).
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

[v0.3.0]: https://github.com/epistemer/ui/releases/tag/v0.3.0
[v0.2.0]: https://github.com/epistemer/ui/releases/tag/v0.2.0
[v0.1.0]: https://github.com/epistemer/ui/releases/tag/v0.1.0
