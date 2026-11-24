# Amendment A9: Confab's iteration 1

| | |
|---|---|
| Status | **Draft**, for epistemer's lock. Each question in §8 has a default; any left unanswered takes it |
| Date | 2026-11-24 |
| From | Confab's findings F7, F9, F20, F25, F26, F30 (example repo, docs/findings.md), as epistemer answered U1, U3 and U5–U8 on 2026-11-24 (the board's item "ui: API questions from Confab iteration 1") |
| Amends | [interface.md](interface.md) §5.10, §6.2, §7; [visual-interface.md](visual-interface.md) §4, §6.1, §8.2, §10, §11; [server-state-interface.md](server-state-interface.md) §4.4; ADR 0007 (superseded by ADR 0014) |
| Not here | U2 (theme serving htmx) waits for the discussion of forking htmx v2. U4 (F11) needed no amendment: clarification A8 documents it |
| Built by | Roadmap C3–C6, then v0.5.0 (C7) |

Everything an app writes today keeps compiling, except where §7 says
otherwise. What renders changes where the sections say so.

## 1. Navigation items ask the gate (U3, F7)

```go
func NavItem(to ui.Link) NavItemSpec // unchanged signature
```

- `NavItem` keeps its link and asks `actions.Allowed(ctx, to)` when it
  renders. Refused: it renders nothing. As `Allowed`, no gate in the
  context (or a nil one) panics, naming `actions.Gate` **[Q1]**. An app
  without permissions sets `actions.AllowAll`, as it already does for
  actions and tables.
- `NavItem` with no address still panics, before the gate is asked.
- **`NavSection`** renders nothing when its items render nothing, so a
  section whose every link is refused leaves no empty heading. Its
  children are rendered into a buffer first (as `PageHeader`'s actions
  are).
- A page the router renders before the guard (a 404, an error page) needs
  a gate too: the error writer sets one, as Confab's does, or
  `actions.AllowAll` where every link is public.
- If every item is refused, `AppNav`'s fill renders nothing; the shell's
  `<nav>` and its narrow-screen toggle stay, as with an empty fill today.
- Tabs, crumbs and links in cells don't ask; an app that must hide one
  asks `actions.Allowed`, as today.

What Confab deletes: the `navItem` wrapper in `internal/admin/layout.templ`.

## 2. Post codes (U5, F9)

```go
// PostCode checks the format of a postal code for country, an ISO 3166-1
// alpha-2 code, compared trimmed and in upper case. A country it doesn't
// know passes every value.
func PostCode(country string) Rule[string]

// HasPostCode reports whether PostCode knows country's format, comparing
// as PostCode does.
func HasPostCode(country string) bool
```

- Changes §6.2's row and drops "It panics at construction if country is
  unknown" (interface §6). `PostCode("")` passes every value too.
- **Values** are matched in upper case (`sw1a 1aa` is a GB code), as a
  person types them; the rule never sees an absent value (§6.1).
- Fixed with it, a bug of the table: GB's pattern anchored only its first
  and last alternatives, so it accepted any text containing a valid post
  code. Every pattern is now anchored as a whole, and a test checks each
  country's pattern against a valid code and a code with junk around it.

What Confab deletes: the `postCode` helper's `recover` in
`internal/admin/events.go`; the rule is used directly,
`rule.PostCode(country)`.

## 3. A switch for booleans (U6, F26)

`KindToggle` keeps its name. Both boolean fields can be switches.

### 3.1 forms

```go
// TwoState reports whether the control is a Checkbox: two states, never
// absent, whatever kind Hint gave it. Bool is three-state.
func (c Control) TwoState() bool
```

`Hint(KindToggle)` stays legal on a `Checkbox`; the renderer reads
`TwoState` to draw it.

### 3.2 What renders

| Field | Without script | With ui's script |
|---|---|---|
| `Bool` (`KindToggle`, three states) | the `<select>` of today (an empty choice, Yes, No) inside `<ui-tristate class="ui-tristate">` | a three-position switch (§3.3) |
| `Checkbox` with `Hint(KindToggle)` | `<input type="checkbox" role="switch" class="ui-input ui-switch" value="true">` in the checkbox's row: a styled checkbox, no script | the same |
| `Checkbox` (`KindCheckbox`) | unchanged | unchanged |

The two-state switch is CSS only: `appearance: none`, a track and a knob
drawn from tokens, the knob at the end when checked (logical properties,
so it mirrors in RTL), the focus ring of every control, `forced-colors`
borders. Native semantics: `role="switch"` on a checkbox reads "on/off".

### 3.3 The three-state switch

Markup the server renders (the label, help and error as every field's,
§6.2):

```html
<ui-tristate class="ui-tristate" data-ui-unset="No answer">
  <select class="ui-input" id="field--…" name="path" aria-describedby="field--…-help field--…-error" …>
    <option value="">Choose…</option>
    <option value="true">Yes</option>
    <option value="false">No</option>
  </select>
  <template class="ui-tristate-marks">
    <svg class="ui-icon" …>(Check)</svg><svg class="ui-icon" …>(Minus)</svg><svg class="ui-icon" …>(X)</svg>
  </template>
</ui-tristate>
```

The words come from the options (Yes, No) and `data-ui-unset`
(`ui.inputs.toggle.unset`); the marks are ui's icons, cloned from the
template, so nothing is drawn by a `data:` URL or inline style.

The script's `<ui-tristate>` upgrades it when it connects:

- **The control.** It adds `<button type="button" class="ui-tristate-track"
  role="checkbox" aria-checked="true|false|mixed" data-state="yes|no|unset">`
  holding `<span class="ui-tristate-knob" aria-hidden="true">` with the
  state's mark, and a visually hidden `<span class="ui-tristate-state">`
  with the state's word. The button takes the select's `aria-describedby`
  (help and error) plus the state word's id, its `aria-invalid`, and
  `aria-required="true"` when the select is `required`. The label's `for`
  moves to the button (found by the select's id through `CSS.escape`), so
  clicking the label and the name work as before.
- **The value stays in the select**, which stays in the form: visually
  hidden (`ui-visually-hidden`), `tabindex="-1"`, `aria-hidden="true"`,
  keeping `name`, `disabled` and `data-ui-when-off`. So a form posts what
  it posts without script, and a condition that disables the select
  disables the button (the script watches the select's `disabled`).
  **`required` moves off the select** to the button's `aria-required`:
  the browser can't show its message on a hidden control, so the server's
  `ErrRequired`, re-rendered beside the field, says it.
- **Positions**, from the start edge: No, no answer (the middle), Yes.
  CSS places the knob from `data-state` with logical properties (so it
  mirrors in RTL) and colours the track: `--ui-color-danger` for No,
  `--ui-color-success` for Yes, `--ui-color-border-strong` for no answer;
  the knob is `--ui-color-surface`, its mark in the track's colour; under
  `forced-colors` the track has a `CanvasText` border, the knob is
  `CanvasText` and the mark shows the state. The mark (✓, —, ✕) and the
  position carry the state, never colour alone.
- **Pointer.** A click on the track's start third chooses No, the middle
  no answer, the end third Yes (by the pointer's position, mirrored in
  RTL); a click on the label goes to the next state.
- **Keys.** Space goes to the next state, no answer → Yes → No → no answer
  (screen readers in browse mode pass Space and clicks; the rest are for
  focus mode). The arrow keys move the knob towards the arrow's side and
  stop at the ends; Home is No, End is Yes; Delete and Backspace clear to
  no answer.
- **Each change** sets the select's value and dispatches `input` and
  `change` on it, so conditions and the app's listeners see it, and puts
  the state's word in a polite live region (`<span class="ui-visually-hidden"
  role="status">` inside the element), since a screen reader announces
  `aria-checked` as checked, not checked or half checked and doesn't read
  a changed description.
- **Staying in step.** The switch reads the select again on its `input`
  and `change` events and after the form's `reset` (in a task, as the
  condition evaluator does: `reset` fires before the values change).
- **Idempotent.** Markup that was already upgraded (htmx's history
  restores a snapshot of the page) is upgraded again from the select: an
  existing track, live region and moved `for` are replaced, never doubled.
  Listeners on the form go when the element disconnects.
- Reduced motion stops the knob's transition. Before the script runs the
  select renders as any select.

### 3.4 Keys and classes

One new i18n key (ui's own text, §11): `ui.inputs.toggle.unset` ("No
answer"). Yes and No stay `ui.inputs.field.yes` and `.no`, read from the
options.

New classes: `ui-switch`, `ui-tristate`, `ui-tristate-track`,
`ui-tristate-knob`, `ui-tristate-state`, `ui-tristate-marks`. New element:
`<ui-tristate>`, and its attribute `data-ui-unset`. No new token.

What Confab deletes: nothing it wrote; its Bool fields become switches.

## 4. `AssertLabelled` skips hidden fields (U7, F30)

```go
// AssertLabelled fails t once for every field, named group and list, at
// any depth, whose label is still the default built from its name. Hidden
// fields (KindHidden) aren't checked: nobody reads their label.
func AssertLabelled(t testing.TB, f *forms.Form)
```

What Confab deletes: the `version` field's label.

## 5. Every Lucide icon (U8, F20)

### 5.1 The API

```go
// IconSpec is an icon. It renders <svg class="ui-icon" …>.
type IconSpec struct{ /* unexported */ }

func (i IconSpec) Size(s Size) IconSpec
func (i IconSpec) Stroke(width float64) IconSpec
func (i IconSpec) Color(c Color) IconSpec
func (i IconSpec) Label(t i18n.Ter) IconSpec
func (i IconSpec) Attrs(a templ.Attributes) IconSpec
func (i IconSpec) Render(ctx context.Context, w io.Writer) error

type Size uint8

const (
	SizeText Size = iota // 1.25em, follows the text around it (the default)
	SizeS                // 1rem
	SizeM                // 1.5rem
	SizeL                // 2rem
)

type Color uint8

const (
	ColorCurrent Color = iota // currentColor (the default)
	ColorMuted                // --ui-color-text-muted
	ColorBrand                // --ui-color-brand-text
	ColorDanger               // --ui-color-danger-text
	ColorWarning              // --ui-color-warning-text
	ColorSuccess              // --ui-color-success-text
	ColorInfo                 // --ui-color-info-text
)

// One function per icon of the pinned Lucide release, named after it:
func Pencil() IconSpec      // pencil
func Trash2() IconSpec      // trash-2
func ExternalLink() IconSpec
func Upload() IconSpec
func Download() IconSpec
func Calendar() IconSpec
// … one per SVG file of the release (2,410 in 1.77.0, Lucide's aliases included)

// Icons drawn towards a side get Start and End functions too, which
// mirror in right-to-left pages; the Left and Right ones don't.
func ArrowStart() IconSpec // arrow-left, mirrored in RTL
func ArrowEnd() IconSpec   // arrow-right, mirrored in RTL
func ArrowLeft() IconSpec  // always left
```

- **Markup.** `<svg class="ui-icon" viewBox="0 0 24 24" fill="none"
  stroke="currentColor" stroke-width="2" stroke-linecap="round"
  stroke-linejoin="round" aria-hidden="true">` with the icon's shapes, as
  today, so no golden changes. Options add, only when they differ from
  the default: `data-size="s|m|l"`, `stroke-width`, `data-color="…"` (CSS
  sets `color` from the token, and the stroke follows `currentColor`).
  Variants are data attributes, as everywhere; there is no inline style,
  so a strict CSP holds.
- **`Label(t)`** gives the icon a name: `role="img"` and `aria-label`
  instead of `aria-hidden`. A blank label panics at render. `Labelled`
  stays, for wrapping any component.
- **`Stroke(w)`**: `w` must be a number above 0 and at most 4 (NaN
  fails), else it panics at render; written with the fewest digits that
  round-trip (`1.5`). Lucide's default is 2, which renders no change.
- **Wire values.** `data-size`: `s`, `m`, `l` (none for `SizeText`);
  `data-color`: `muted`, `brand`, `danger`, `warning`, `success`, `info`
  (none for `ColorCurrent`). A `Size` or `Color` outside its constants, a
  nil or blank `Label`, and a zero `IconSpec` (not made by an icon
  function) panic at render, naming `icons`.
- **`Attrs`** merges as every spec's (§2.2), `class` joined; owned:
  `viewBox`, `fill`, `stroke`, `stroke-width`, `stroke-linecap`,
  `stroke-linejoin`, `aria-hidden`, `role`, `aria-label`, `data-mirror`,
  `data-size`, `data-color`.
- **Mirroring** stays `data-mirror`, flipped by CSS under `:dir(rtl)`
  (§4). Names are split into words at `-`. An icon whose name has the word
  `left` or `right`, and whose name with every `left` and `right` swapped
  is an icon too, is a pair (69 files in 1.77.0). Each icon of a pair
  also gets a function named with `left` → `Start` and `right` → `End`:
  `arrow-left` → `ArrowStart`, `arrow-right` → `ArrowEnd`,
  `arrow-left-right` → `ArrowStartEnd`, `arrow-right-left` →
  `ArrowEndStart`, `panel-left-close` → `PanelStartClose`. Those carry
  `data-mirror`; the `Left` and `Right` functions don't. Words inside
  words (`copyleft`) don't count.
- **Not mirrored**, though paired, because they draw something that
  doesn't turn with the text: `pilcrow-*` (a glyph), `decimals-arrow-*`
  (digits run left to right in every script), `mouse-*` (a mouse's
  buttons), `toggle-*` (a switch's state). They get no `Start`/`End`. The
  list lives in the generator and is part of this contract.
- **The names.** Lucide's file name in Pascal case: `circle-x` →
  `CircleX`, `trash-2` → `Trash2`. Lucide's own `-start` and `-end` icons
  (`align-start-vertical`) keep their names. The generator fails, and
  the pin waits, on a name two icons would share, a Go keyword, a name
  starting with a digit (none in 1.77.0), or one of the package's own
  identifiers (`IconSpec`, `Size*`, `Color*`, `Labelled`, `More`,
  `Warning`). Lucide's deprecated aliases are files too; each becomes a
  function, which costs nothing an app doesn't call.
- **Today's names keep working**, with the same markup: `X`, `Plus`,
  `Minus`, `Check`, `ChevronDown`, `ChevronUp`, `Menu`, `Search`, `Info`,
  `CircleCheck`, `CircleX`, `Eye`, `EyeOff`, `Sun`, `Moon`, `User` are
  Lucide's names; `ChevronStart`, `ChevronEnd`, `ArrowStart`, `ArrowEnd`
  are generated by the pair rule; `More` (`ellipsis`) and `Warning`
  (`triangle-alert`) stay as hand-written aliases.

### 5.2 The generator (ADR 0014, superseding ADR 0007)

- **Vendored data, not SVG files.** `icons/internal/lucide/shapes.txt`
  holds one line per icon, its name and its shapes as the generator
  checked them (about 520 kB for 1.77.0), with Lucide's `LICENSE` and a
  `VERSION` file naming the release and the npm tarball's `sha512`
  integrity.
- **Generating** (`go generate ./icons`, offline) reads `shapes.txt` and
  writes `icons/lucide.go`: one function per icon returning an `IconSpec`
  whose shapes are a Go string constant written with `strconv.Quote`, and
  the internal list the catalogue's page of every icon reads. One
  `Render` writes the `<svg>` around them **[Q7]**. CI's drift check runs
  it. No `.templ` per icon: about 2,400 templ components would be some
  100,000 generated lines for one identical wrapper.
- **Moving the pin** (`go run ./icons/internal/gen -fetch 1.78.0`, by
  hand): downloads the tarball from the npm registry, checks it against
  the registry's `sha512` integrity, reads only `package/icons/*.svg` and
  `package/LICENSE` from it (no other entry is written anywhere; entries
  with `..`, absolute paths or links are refused; at most 64 MB unpacked,
  4 MB per file), and rewrites the vendored files. The diff of
  `shapes.txt` is reviewed like code.
- **Each SVG is parsed, not copied**, with `encoding/xml` and no DTD
  (a `<!DOCTYPE>` or an entity other than XML's five fails). The root
  must be `svg` with Lucide's attributes and `viewBox="0 0 24 24"`. Inside
  it only these, with only these attributes, each value matching its
  grammar; anything else fails the generator:

  | Element | Attributes | Value grammar |
  |---|---|---|
  | `path` | `d` | path data: `[MmLlHhVvCcSsQqTtAaZz0-9.,+\- ]+` |
  | `circle` | `cx`, `cy`, `r` | a number |
  | `ellipse` | `cx`, `cy`, `rx`, `ry` | a number |
  | `line` | `x1`, `y1`, `x2`, `y2` | a number |
  | `polyline`, `polygon` | `points` | numbers, commas, spaces |
  | `rect` | `x`, `y`, `width`, `height`, `rx`, `ry` | a number |

  A number is `-?[0-9]*\.?[0-9]+`. The shapes are written back from the
  parsed values, never from the file's text.
- **No registry**: each icon is its own function, so the linker drops the
  ones an app doesn't call. The catalogue's list is in an internal
  package nothing else imports.

### 5.3 Breaks

- The 22 functions return `IconSpec` instead of `templ.Component`.
  `@icons.X()`, `actions.New(…).Icon(icons.X())` and any call that passes
  an icon where a `templ.Component` is wanted compile as before. What
  breaks:
  - a `func() templ.Component` given an icon function (`var f func()
    templ.Component = icons.X`);
  - a variable inferred from an icon then given another component
    (`c := icons.Check(); if busy { c = spinner }`): declare it
    `var c templ.Component`;
  - a generic call that infers one type from an icon and a component.

  ui itself has none of these; Confab has none.

What Confab gains: pencil, trash, external-link, upload, download and
calendar for its actions.

## 6. `Reply` and the flash (U1, F25)

### 6.1 The flash ui declares

Mirrors handlers' draft amendment A7 (`handlers/middleware.Flash`, neozp
D4), whose method set satisfies it; ui imports no handlers.

```go
package components

// Flasher carries messages to the next page across a redirect, and gives
// the ones this request delivers. handlers/middleware's Flash is one.
// Levels are "info", "success", "warning" and "error".
type Flasher interface {
	Add(ctx context.Context, level string, msg i18n.Ter, args ...any)
	Messages(ctx context.Context) iter.Seq2[string, i18n.Ter]
}

// Flash is the request's Flasher, set once per request as actions.Gate is:
// ctx = components.Flash.With(ctx, flash).
var Flash *ui.Provider[Flasher]
```

- **Wiring.** Set `components.Flash` inside handlers' `Flash.Middleware`
  (whose `Add` panics outside it), in the same middleware that sets
  `actions.Gate`. ui treats a nil interface as no flash; it can't see a
  typed nil inside the interface, so don't put one there.
- **Tones and levels:** success ↔ `success`, info and no tone ↔ `info`
  (the same toast: an info toast has no `data-tone`), warning ↔
  `warning`, danger ↔ `error`; an unknown level reads as info. A message
  crosses as its level and text only: a toast's `Attrs` don't travel.

### 6.2 What `Reply` does

| | With htmx | Without |
|---|---|---|
| a change, a toast, `Fail` | unchanged (§4.4) | 303 to `Else`; **the toast, or `Fail`'s danger message, goes to the flash** |
| `Go(l)` | `HX-Redirect`; **a toast (or `Fail`'s message) goes to the flash** | 303 to `l`; the same |

- `Go` no longer panics after a toast, a change or `Fail`. A change before
  `Go` is dropped (the page it goes to shows the result); its toast is
  flashed. `Fail(err)` then `Go` flashes `err`'s text as an error.
- Still panics, all before anything is written: two changes,
  `Fail(nil)`, a blank id.
- **No flash in the context** (none set, or nil): a toast on a redirect
  is dropped, and the redirect is sent **[Q5]**. `Reply` never fails after
  the handler's work for want of a flash.
- `Toast(t)`'s text travels as handlers' `Add` decides: a key keeps the
  reader's language, other text is rendered in the writer's. `Fail(err)`'s
  message is `errors.T(ctx, err)`: a private error reads as errors'
  generic text, never `Error()`. A `Toast` with `Fail` is replaced by
  `Fail`'s message, as with htmx (§4.4).
- **Which column.** A request with `HX-Request: true`, boosted or not,
  takes the htmx column, as today; `Go` answers it with `HX-Redirect`,
  and the page it loads delivers the flash.
- `Else` and `Go` still refuse an address that isn't local
  (`ErrNotLocal`, nothing written, nothing flashed), so `HX-Redirect`
  only ever names this site.
- An app that flashed by hand before `Fail(err).Else(l)` now shows two
  messages; it drops its own.

### 6.3 Showing flashes

- `Toasts()` (the region the shells render) renders, inside it, one
  `Toast` per message of `Flash.Value(ctx).Messages(ctx)`, tone from the
  level, after the toasts it already holds. Without a flash it renders as
  today.
- An app that renders `Toasts()` itself gets the same. Nothing else reads
  the flash.
- A boosted page whose swap target isn't the body (`hx-target="#main"`)
  leaves the region out of the swap, so a flash it delivers doesn't show.
  ui's shells boost the body.

What Confab deletes: `internal/admin/flash.go` (`goWith`, `takeFlash`) and
the `flash` parameter of its shell; `Reply(w, r).Toast(saved).Go(link)`
works.

## 7. Everything that breaks

| What | Who it hits | Why |
|---|---|---|
| `NavItem` panics with no `actions.Gate` in the context | a page that renders nav items without a gate | Q1's default |
| `NavItem` refused by the gate renders nothing | an app whose gate refuses a nav link it wanted shown | U3 |
| `PostCode` no longer panics for an unknown country | code that relied on the panic | U5 |
| `Bool` fields render as a switch when the script runs | every app's three-state fields: the look changes, the posted values don't | U6 |
| icon functions return `IconSpec` | `func() templ.Component` values holding an icon function; a variable inferred from an icon | U8 |
| `PostCode` matches values in upper case | an app that wanted lower case refused | U5 |
| `Fail` without htmx flashes its message | an app that flashed it by hand shows it twice | U1 |
| `Go` after a toast or change redirects instead of panicking | none that worked | U1 |

## 8. Questions for the lock

Each has a default (recommended); an unanswered one takes it.

- **Q1. A `NavItem` with no gate in the context.** *Default:* panics, as
  `actions.Allowed` does (S9): a forgotten gate shows up on the first
  page, not as links that leak. *Alternative:* shows the item, as today.
- **Q2. The three-state switch's element and order.** *Default:*
  `<ui-tristate>`; No at the start edge, no answer in the middle, Yes at
  the end; Space goes no answer → Yes → No. *Alternative:* a name of
  your choice; another order.
- **Q3. Icon sizes.** *Default:* four named sizes as data attributes
  (`SizeText` 1.25em, `SizeS` 1rem, `SizeM` 1.5rem, `SizeL` 2rem).
  *Alternative:* a free CSS length, which needs inline `style`
  (a CSP exception) or `width`/`height` attributes that ui's CSS would
  have to stop overriding.
- **Q4. Icon colours.** *Default:* `Color` with the six token colours of
  §5.1, as data attributes. *Alternative:* `Color(token string)` taking
  any `--ui-*` custom property.
- **Q5. A redirect with a toast and no flash.** *Default:* the toast is
  dropped and the redirect sent. *Alternative:* panic before writing,
  which is loud but fires after the handler's work, the problem of F25.
- **Q6. Lucide pin.** *Default:* lucide-static 1.77.0 (ADR 0007's), or
  the newest release at least two weeks old when C5 runs, vendored as the
  checked shapes (about 520 kB). *Alternative:* fetch at `go generate`
  time with the checksum, nothing vendored (generation and CI's drift
  check then need the network).
- **Q7. Icons as Go, not `.templ`.** *Default:* the generator writes plain
  Go: one function per icon returning an `IconSpec`, which is a
  `templ.Component`, rendered by one shared `Render` (about 10,000
  generated lines). *Alternative:* a `.templ` component per icon, as
  zpui's tool wrote for Flowbite (about 100,000 generated lines, the same
  markup).
