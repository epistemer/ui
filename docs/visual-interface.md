# ui: interface of the visual packages

| | |
|---|---|
| Status | **Locked by epistemer on 2026-11-22 (D6), with the defaults of V1–V18.** Amendment A5 (links, §2.7 and §13.4) was locked the same day, and A6 (actions and tables, §13.5) after it. It stays a document of its own beside [interface.md](interface.md), whose §10 records the lock and amendment A4 (uitest, §13.1). A change to anything here that alters what an app writes or gets back is an amendment |
| Date | 2026-11-22 |
| Roadmap item | D5 |
| Inputs | [visual-design.md](visual-design.md) §2–§11 (Q1–Q9 and the spike's findings), ADRs [0004](adr/0004-own-css.md), [0005](adr/0005-client-behaviour.md), [0006](adr/0006-goja-for-script-tests.md), reviews [0002](reviews/0002-visual-design.md)–[0004](reviews/0004-d4-part2.md), the spike in `internal/spike`, and review [0005](reviews/0005-visual-interface.md) of this draft |

This is the contract of the visual packages for the first lock, the scope
of Q9's default: theme (with `themeschema`), icons, elements, inputs (the
field, without lists), components (dialog, confirmation, menu, navigation
tabs, toasts, detail list), layout (app shell, auth shell, page header) and
the catalogue. Tables, server tables, pagination, filters, the remote
select, uploads and list fields wait for D7, which gets its own design and
an amendment to the lock.

The locked interface.md stays as it is until then. Nothing here is
exported yet: the spike in `internal/spike` is the evidence, and the
milestones after D6 (V1…) build the packages to this text.

**How to read the marks.**
- **(spiked)**: the D4 spike built it and tests or the browser pane checked
  it (visual-design.md §11).
- **(not spiked)**: this draft goes beyond what the spike proved. The
  milestone that builds it may find the design wrong; a fix before v1.0.0
  is an amendment, as for any locked text.
- **[V*n*]**: an API choice that visual-design.md didn't settle. Each is a
  question for epistemer with a default, collected in §14. Taking the
  default means locking the text as written.

## 1. Module

| | |
|---|---|
| Module | `github.com/epistemer/ui` (unchanged), plus the nested module `github.com/epistemer/ui/theme/themeschema` |
| New packages | `ui/theme`, `ui/icons`, `ui/elements`, `ui/inputs`, `ui/components`, `ui/layout`, `ui/catalog`, `ui/catalog/catalogtest` |
| theme imports | the standard library, templ, `github.com/epistemer/errors`, `github.com/epistemer/i18n` |
| themeschema imports | `github.com/epistemer/cfgx/dsl`, `ui/theme` (its own go.mod, so cfgx never enters ui's) |
| icons imports | templ, `github.com/epistemer/i18n` |
| elements imports | the kernel, templ, icons, i18n, `github.com/epistemer/models/option` |
| inputs imports | the kernel, templ, forms, elements, icons, i18n, errors |
| components imports | the kernel, templ, elements, icons, i18n |
| layout imports | the kernel, templ, components, elements, icons, i18n, `net/http` |
| catalog imports | the standard library, templ, theme |
| catalogtest imports | catalog, uitest, theme, `ui/internal/themehook`, `testing` |
| Internal packages | `ui/internal/themehook`: a variable through which `ui/theme` hands its joined stylesheet to catalogtest, set in theme's `init` (the way forms hands formstest its hook); `ui/internal/a11y`: the names and selector checks, shared by uitest and catalogtest. Neither is API |
| Never imported | by icons, elements, inputs, components, layout: `ui/theme` (they emit class names; the CSS lives in `theme/css/`, design §3). By any ui package: htmx (the app's dependency, ADR 0005), a CSS or JS framework. By forms and the kernel: anything in this document |

The layer rule (design §2) is the import rule: each package imports only
the public API of the packages listed for it. An app can therefore rebuild
any composite from public parts.

## 2. Conventions every visual package follows

### 2.1 Builders (Q2) [V1]

```go
// The shape every builder has. X is the component's name.
type XSpec struct{ /* unexported */ }

func X(required ...) XSpec                      // starts it: what it can't do without
func (x XSpec) Option(v T) XSpec                // each method returns a changed copy
func (x XSpec) Attrs(a templ.Attributes) XSpec  // extra attributes, merged (§2.2)
func (x XSpec) Render(ctx context.Context, w io.Writer) error // templ.Component
```

- **Immutable values.** Every method has a value receiver and returns a
  copy; no copy shares a map or slice with another. A spec can be kept in a
  package variable, refined per call and rendered from many goroutines:

  ```templ
  var cancel = elements.Button().Variant(elements.Ghost)

  @cancel.Href(back) { Cancel }
  ```

- **The name.** `elements.Button()` returns `elements.ButtonSpec`: a Go
  function and a type can't share a name (spike finding 7). The `Spec`
  suffix appears only in `go doc` and in the rare declared variable, never
  at a call site. [V1]
- **Required arguments go to the constructor** (a dialog's title, a menu's
  label, a field's control). Everything else is a method, and the bare
  constructor is the common look.
- **Booleans for states that follow data.** A state that commonly depends
  on data takes a bool, so a condition is an argument:
  `.Disabled(!canEdit)`, `.Current(path == here)`. A method that fixes what
  the thing is takes none: `.Submit()`, `.Danger()`.
- **Content.** A component with one content region takes templ children. A
  component with several takes one slot per region (the kernel's rule 2:
  loose content in a slotted component's block is `ErrStrayContent`).
  Slots whose fill must reach the component's markup with attributes
  (triggers) are `SlotOf[templ.Attributes]`. [V8]
- **Programmer mistakes panic at render**, with a message naming the
  package, the component and the mistake, as the kernel's `Get` does:
  `elements: Button with href "/x" can't submit`. A required slot left
  unfilled is one: `TooltipTrigger`, `DialogBody`, `MenuItems`, `AppMain`,
  `AuthBody`.
- **Every spec has `Attrs`**, even where this document lists no other
  method.
- **Text a person sees is an `i18n.Ter`** (design §2.6). Builders store the
  `Ter` and call `T(ctx)` when they render.

Builders without options are plain functions returning `templ.Component`:
icons, value renderers and the few regions listed as such. [V9]

### 2.2 Attributes and `Merge` (spiked)

```go
package elements

func Merge(own templ.Attributes, owned []string, caller templ.Attributes) templ.Attributes
```

`Merge` returns a new map, leaving its arguments alone:
- keys compare without regard to case and come out in lower case;
- `class`, `aria-describedby`, `aria-labelledby` and `aria-controls` are
  space-separated lists: the caller's tokens are added after the
  component's, without repeats. A value present on either side must be a
  string; any other type panics, so a caller's class can't silently
  replace the component's (`aria-controls` is new beside the spike);
- a key in `owned` keeps the component's value (or its absence), whatever
  the caller passed;
- any other key takes the caller's value.

Every builder's `Attrs` merges this way, and repeated `Attrs` calls
accumulate in order. Each component documents **the element that receives
them** and **its owned keys** (the tables in §5–§9). HTML keeps the first of
two copies of an attribute, so ui never renders two maps into one tag.

`Merge` stays in `ui/elements` (the backlog's gate): inputs and components
already import elements, and so can an app's own components. [V5]

### 2.3 Class names and variant attributes (Q3)

- One class per component, `ui-<component>`; its parts are
  `ui-<component>-<part>`. Every class listed in this document is public
  API: apps style them, so renaming or removing one is an amendment.
  Adding one isn't.
- Variants are data attributes, rendered only when they differ from the
  component's default, so the bare class is the default look:

  | Attribute | Values | Used by |
  |---|---|---|
  | `data-variant` | `outline`, `ghost` | Button |
  | `data-size` | `s`, `l` | Button, Avatar, Spinner, Heading |
  | `data-tone` | `neutral`, `brand`, `danger`, `warning`, `success`, `info` | Button, Badge (all six); Alert, Toast (`info`, `success`, `warning`, `danger`); MenuItem (`danger`) |
  | `data-mirror` | (presence) | directional icons |

- State is styled from HTML and ARIA, never from classes (ADR 0004):
  `:disabled`, `[aria-disabled="true"]`, `[aria-current]`,
  `:user-invalid`, `[aria-invalid="true"]`, `:popover-open`, `[open]`,
  `[aria-expanded="true"]`.
- `data-ui-*` attributes are hooks for the ui script (§10), not styles.

Shared value types live in elements and are used by every package above
it:

```go
package elements

type Variant string // Solid (""), Outline, Ghost
type Size string    // Small ("s"), Medium (""), Large ("l")
type Tone string    // Neutral, Brand, Danger, Warning, Success, Info

const (
	Solid   Variant = ""
	Outline Variant = "outline"
	Ghost   Variant = "ghost"

	Small  Size = "s"
	Medium Size = ""
	Large  Size = "l"

	Neutral Tone = "neutral"
	Brand   Tone = "brand"
	Danger  Tone = "danger"
	Warning Tone = "warning"
	Success Tone = "success"
	Info    Tone = "info"
)
```

The zero `Tone` means the component's own default: brand for Button,
neutral for Badge and MenuItem, info for Alert and Toast. A component
renders `data-tone` only when the tone it resolves differs from its
default, so `Button().Tone(Brand)` renders no attribute, and a neutral
outline button (`Button().Variant(Outline).Tone(Neutral)`, a Cancel) is
told apart by `data-tone="neutral"`. Neutral reads the text, surface and
border tokens, so it needs no accent of its own. Alert and Toast take only
the four message tones; `Brand` or `Neutral` on them panics at render,
because a message's level must have an icon and a word. [V7]

### 2.4 Ids (spiked)

Components draw generated ids from the kernel (`ui.NewID`, `ui.Scope`,
`Get`, `Part`), so fragments that htmx swaps in never collide. The
singletons a page holds once have fixed ids, listed with them:
`ui-confirm` (§7.2), `ui-toasts` (§7.5), `ui-main` (§8.1). No component
renders a generated id an app is expected to target; an app that targets
one gives it a keyed id of its own through the slot or `Attrs`.

### 2.5 ui's own text (Q5) [V3, V4]

ui declares its texts with `i18n.Key` under
`ui.<package>.<component>.<name>`, with English defaults, as errors and
forms do. The key variables are unexported; the keys and their English
texts are the contract (§11). An app changes a text the way it changes any
translation, including in the source locale, so no builder has a method per
text. The exceptions carry the caller's own text, because it differs per
call: the confirmation's question, a spinner's label, an icon-only
button's label.

Declared errors a person can see (an admin's theme value, §3.2) follow
forms' precedent: `errors.<package>.<name>`.

### 2.6 Accessibility (spiked for the checks)

Each interactive component's doc comment names the ARIA Authoring Practices
pattern it follows and lists its keyboard behaviour (design §7). Keyboard
and focus wait for the browser job; the names check (§13.1) and `CheckIDs`
run on every story.

### 2.7 Addresses (amendment A5)

An address is a `ui.Link`, the kernel's `interface{ URL() string }`
(interface.md §4A): a route link from handlers' generated builders, or
`ui.URL("/docs")` for a literal or external one.

- Components render an address through `templ.URL`, which lets `http`,
  `https`, `mailto`, `tel` and relative addresses through.
- **No address** is a nil link, one whose `URL()` is empty, or one
  `templ.URL` refuses. It is what an empty `templ.SafeURL` was before A5:
  no `href` on a Button or MenuItem, `Empty` from `Link` and `Image`, no
  image in an Avatar, the current page in a `Crumb`.

## 3. Theme (`ui/theme`) (spiked)

### 3.1 Tokens (the CSS contract)

Every token is a custom property in the `ui.tokens` layer on `:root`. Its
name is public API; its value isn't. Adding a token isn't an amendment;
renaming or removing one is.

| Group | Tokens |
|---|---|
| Neutral colours | `--ui-color-surface`, `-surface-raised`, `-text`, `-text-muted`, `-border`, `-border-strong` (a control's edge, 3:1, finding 15), `-focus` |
| Accents, per role (`brand`, `danger`, `warning`, `success`, `info`) | `--ui-color-<role>` (the fill), `--ui-color-on-<role>`, `--ui-color-<role>-hover`, `-<role>-active`, `-<role>-text`, `-<role>-subtle` |
| Space | `--ui-space-1` … `--ui-space-6` |
| Radius | `--ui-radius` (the base), `--ui-radius-s`, `-m`, `-l`, `-full` |
| Lines | `--ui-border-width`, `--ui-focus-width`, `--ui-focus-offset` |
| Type | `--ui-text-xs`, `-s`, `-m`, `-l`, `-xl`, `--ui-font-sans`, `--ui-font-mono` |
| Shadow | `--ui-shadow-1` … `-3` |
| Motion | `--ui-duration-s`, `-m`, `--ui-ease` |

Colours use `light-dark()`. Fill, on-colour and shades are the same in both
schemes; text and subtle are derived per scheme (design §4). Reduced motion
zeroes the duration tokens (finding 6). The layers, lowest first:
`ui.reset, ui.tokens, ui.base, ui.elements, ui.inputs, ui.components,
ui.layout` (ADR 0004).

The design's `--ui-space-7`, `-8` and `--ui-text-2xl` are left out until a
component needs them.

### 3.2 Config and its checks

```go
type Config interface {
	ThemeBrandColor() string
	ThemeDangerColor() string
	ThemeWarningColor() string
	ThemeSuccessColor() string
	ThemeInfoColor() string
	ThemeFontSans() string
	ThemeFontMono() string
	ThemeRadius() string
}

type Defaults struct{} // the token table's values; satisfies Config

func CheckColor(s string) error  // #rgb or #rrggbb
func CheckFont(s string) error   // a font-family list
func CheckRadius(s string) error // 0, or px ≤ 32, or rem/em ≤ 2

var (
	ErrColor  error // errors.theme.color
	ErrFont   error // errors.theme.font
	ErrRadius error // errors.theme.radius
)
```

- `Config` is what an admin may change at run time (Q4): the five accent
  fills, the two font families and the radius base. Its getters carry the
  group's prefix, because cfgx's generated settings satisfy it through
  `themeschema` (§3.4), pinned with `GoName("Theme")`.
- The checks are top-level functions of `func(string) error`, the shape
  cfgx's `.Validate` takes, in a package that doesn't import `cfgx/dsl`, as
  cfgx requires. Each returns its declared error (errors' `Key`), which an
  admin sees when cfgx refuses a value:
  - `CheckColor`: `#` and 3 or 6 hex digits. No names, no functions: a
    value reaches a stylesheet the CSP trusts.
  - `CheckFont`: 1 to 12 comma-separated families, each either a quoted
    name (`"Noto Sans"`, `'Inter'`) of 1–64 ASCII letters, digits,
    spaces, `.`, `_` and `-`, or a bare word matching
    `[A-Za-z][A-Za-z0-9-]{0,63}` (`system-ui`, `sans-serif`) other than a
    CSS-wide keyword (`inherit`, `initial`, `unset`, `revert`,
    `revert-layer`). Nothing else gets through: no `;`, `}`, `(`, `\` or
    `/`.
  - `CheckRadius`: `0`, or an unsigned number with at most two digits
    before the point and three after, in `px` (at most 32), `rem` or `em`
    (at most 2). The scale follows the base: `--ui-radius-s` is half of
    `--ui-radius`, `-m` equal, `-l` double (`calc()` in `ui.tokens`), and
    `-full` is fixed.
- A value that fails its check when the theme reads it (a config that
  bypassed cfgx) falls back to the default, silently. The check at the key
  is what tells the admin.

### 3.3 Theme: assets, run-time stylesheet, head

```go
type Theme struct{ /* unexported */ }

func New(prefix string, c Config) (*Theme, error)

func (t *Theme) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (t *Theme) Head() templ.Component
```

- `New` builds the bundle once: it renders the token layer from the table
  with the defaults, joins `theme/css/*.css` in layer order (each file
  wrapped in its layer), and reads the script modules. `prefix` is a path
  on the app's own origin (`/ui`): one leading slash, no `//`, `\`, `?`,
  `#`, quotes, `<`, `>` or space; anything else is an error.
- `ServeHTTP` serves, under the prefix:
  - `<prefix>/<release>/ui.css`, `<prefix>/<release>/ui.js` and the
    modules `ui.js` imports, where `<release>` is a hash of every file, so
    a relative `import "./cond.js"` stays in one release (ADR 0005).
    Cached `immutable`.
  - `<prefix>/theme/<hash>.css`: the run-time theme, one
    `@layer ui.tokens { :root { … } }` block with the derived accent
    tokens, the fonts and the radius from `Config`. The current hash is
    cached `immutable`; any other hash gets the current theme with
    `Cache-Control: no-cache` (design §4).
  - An unknown `<release>` is a 404, as spiked. Serving the current files
    under an old release would mix modules of two releases: a cached old
    `ui.js` importing a new `cond.js`. Review 0005 turned down the draft's
    first answer here. Anything else is a 404 too.
- The theme reads `Config` on every `Head` render and every theme request,
  and rebuilds the stylesheet only when one of the eight values differs
  from the last ones seen. It needs no change callback from cfgx. Safe for
  concurrent use.
- `Head` renders, in order, `<link rel="stylesheet">` for `ui.css`, for
  the current theme, and `<script type="module">` for `ui.js`. The theme's
  tokens come after ui's in the same layer, so they win. Nothing inline:
  the page works under `script-src 'self'; style-src 'self'`.
- Derivation (design §4, findings 1 and 2): the on-colour is black or
  white, whichever contrasts more with the fill; hover and active mix the
  fill in OKLab away from the on-colour (88%, 76%); text is shifted in
  OKLCh lightness, gamut-mapped, until it reaches 4.55:1 on the stricter
  surface in each scheme; subtle is a 14% OKLab tint of the raised surface
  that keeps text contrast; the focus ring is the brand's text colour.
  The derivation is internal: its results are tokens, not Go API.

An app that doesn't use cfgx passes `theme.Defaults{}` or its own `Config`.

### 3.4 themeschema (nested module) (not spiked)

The spike's `Config` had no prefix (`BrandColor()`), and it had no schema;
the check functions are its validators renamed.

```go
package themeschema // github.com/epistemer/ui/theme/themeschema

func Group() *dsl.GroupDef
```

`Group` returns the cfgx group `theme`, pinned with `GoName("Theme")`:

| Key | Getter | Default | Check |
|---|---|---|---|
| `brand_color` | `ThemeBrandColor` | `#2f5bd3` | `theme.CheckColor` |
| `danger_color` | `ThemeDangerColor` | `#c62828` | `theme.CheckColor` |
| `warning_color` | `ThemeWarningColor` | `#f2b01e` | `theme.CheckColor` |
| `success_color` | `ThemeSuccessColor` | `#1e8a4c` | `theme.CheckColor` |
| `info_color` | `ThemeInfoColor` | `#0f7c9e` | `theme.CheckColor` |
| `font_sans` | `ThemeFontSans` | the table's system stack | `theme.CheckFont` |
| `font_mono` | `ThemeFontMono` | the table's monospace stack | `theme.CheckFont` |
| `radius` | `ThemeRadius` | `0.5rem` | `theme.CheckRadius` |

- Every key is a `String`, `Mutable`, with its default read from
  `theme.Defaults`, so the table stays the one source. No key name holds an
  initialism, so the getters don't depend on the app's initialisms.
- Its descriptions become cfgx's labels (`settings.theme.<key>`), which the
  admin's settings page shows.
- A test in ui keeps `theme.Config`, `Defaults` and `Group` in step: the
  same keys, defaults and checks.
- Use: `var Theme = themeschema.Group()` and
  `dsl.Command("Serve").Uses(Theme, …)`; the generated settings satisfy
  `theme.Config`.
- The module is tagged `theme/themeschema/vX.Y.Z` with ui's releases.
  Values per tenant come later, with neozp's tenancy.

## 4. Icons (`ui/icons`) [V10]

```go
func X() templ.Component
func Plus() templ.Component
func Minus() templ.Component
func Check() templ.Component
func ChevronDown() templ.Component
func ChevronUp() templ.Component
func ChevronStart() templ.Component // mirrors
func ChevronEnd() templ.Component   // mirrors
func ArrowStart() templ.Component   // mirrors
func ArrowEnd() templ.Component     // mirrors
func More() templ.Component         // a row of three dots
func Menu() templ.Component         // three lines
func Search() templ.Component
func Info() templ.Component
func Warning() templ.Component
func CircleCheck() templ.Component
func CircleX() templ.Component
func Eye() templ.Component
func EyeOff() templ.Component
func Sun() templ.Component
func Moon() templ.Component
func User() templ.Component

func Labelled(icon templ.Component, text i18n.Ter) templ.Component
```

- Every icon is `<svg class="ui-icon" viewBox="0 0 24 24"
  aria-hidden="true">` drawn in `currentColor`, sized `1.25em` by CSS, so
  it follows the text around it. Directional icons carry `data-mirror` and
  flip under `:dir(rtl)` (spiked with `ChevronEnd`).
- An icon is decorative: the control around it carries the name. An icon
  that is the only carrier of meaning goes through `Labelled`, which wraps
  it in `<span class="ui-icon-labelled" role="img" aria-label="…">`.
- An app's own icons follow the same contract: a `templ.Component` that
  renders `svg.ui-icon`.
- *(not spiked beyond Plus, X, ChevronDown, ChevronEnd.)* The set is what
  this document's components use, plus `Search`, `Eye`, `EyeOff`, `User`
  for the layout and auth pages. The paths are vendored from one open
  icon set with its licence notice (default: Lucide, ISC). [V10]

## 5. Elements (`ui/elements`)

### 5.1 Button (spiked)

```go
type ButtonSpec struct{ /* unexported */ }

func Button() ButtonSpec

func (b ButtonSpec) Variant(v Variant) ButtonSpec
func (b ButtonSpec) Size(s Size) ButtonSpec
func (b ButtonSpec) Tone(t Tone) ButtonSpec
func (b ButtonSpec) Href(l ui.Link) ButtonSpec // A5
func (b ButtonSpec) Submit() ButtonSpec
func (b ButtonSpec) Disabled(d bool) ButtonSpec
func (b ButtonSpec) Label(t i18n.Ter) ButtonSpec // aria-label, for an icon-only button
func (b ButtonSpec) Attrs(a templ.Attributes) ButtonSpec
```

- Children are the content (text, icons). It renders:
  - `<button type="button">`, or `type="submit"` with `Submit`;
  - `<a href>` with `Href`;
  - with `Href` and `Disabled(true)`: `<a role="link"
    aria-disabled="true">` without `href` (design §5), which the ui script
    keeps inert (§10.2).
- `Href` with `Submit` panics at render.
- Attrs go to the `<button>` or `<a>`. Owned: `type`, `role`, `href`,
  `disabled`, `aria-disabled`, `data-variant`, `data-size`, `data-tone`,
  and `aria-label` when `Label` is set.
- Class `ui-button`. *(`Label` is not spiked; the spike's dialog close
  button passed `aria-label` through `Attrs`.)*

### 5.2 Badge, Alert, Card (not spiked)

```go
type BadgeSpec struct{ /* unexported */ }

func Badge() BadgeSpec
func (b BadgeSpec) Tone(t Tone) BadgeSpec
func (b BadgeSpec) Attrs(a templ.Attributes) BadgeSpec

type AlertSpec struct{ /* unexported */ }

func Alert() AlertSpec
func (a AlertSpec) Tone(t Tone) AlertSpec
func (a AlertSpec) Title(t i18n.Ter) AlertSpec
func (a AlertSpec) Attrs(at templ.Attributes) AlertSpec

type CardSpec struct{ /* unexported */ }

var (
	CardHeader *ui.Slot
	CardBody   *ui.Slot
	CardFooter *ui.Slot
)

func Card() CardSpec
func (c CardSpec) Attrs(a templ.Attributes) CardSpec
```

- **Badge**: `<span class="ui-badge">` around its children, in the subtle
  colours of its tone, neutral (surface and border) by default. Attrs go to
  the span.
- **Alert**: an inline, static message: `<div class="ui-alert">` with the
  tone's icon (`Info`, `Warning`, `CircleCheck`, `CircleX`) and a visually
  hidden tone word (`ui.elements.alert.info`, `…warning`, `…success`,
  `…danger`), so the level isn't shown by colour alone. The word's text
  ends with its own punctuation ("Error:"), so each locale punctuates it
  its own way, and ui renders a space after it, so a screen reader doesn't
  run it into the message (§11). Then an optional
  `ui-alert-title` (a `<p>`, not a heading) and the children in
  `ui-alert-body`. No live role: a message that appears later is a toast
  (§7.5). Attrs go to the outer div.
- **Card**: `<section class="ui-card">` with `ui-card-header`,
  `ui-card-body` and `ui-card-footer` for the filled slots only (the
  kernel's `Filled`, M4.5's product card). Attrs go to the section. [V8]

### 5.3 Avatar, Spinner, Tooltip, Separator, Heading (not spiked)

```go
type AvatarSpec struct{ /* unexported */ }

func Avatar(name i18n.Ter) AvatarSpec
func (a AvatarSpec) Src(l ui.Link) AvatarSpec // A5
func (a AvatarSpec) Size(s Size) AvatarSpec
func (a AvatarSpec) Decorative() AvatarSpec
func (a AvatarSpec) Attrs(at templ.Attributes) AvatarSpec

type SpinnerSpec struct{ /* unexported */ }

func Spinner() SpinnerSpec
func (s SpinnerSpec) Size(sz Size) SpinnerSpec
func (s SpinnerSpec) Label(t i18n.Ter) SpinnerSpec
func (s SpinnerSpec) Attrs(a templ.Attributes) SpinnerSpec

var TooltipTrigger *ui.SlotOf[templ.Attributes]

type TooltipSpec struct{ /* unexported */ }

func Tooltip(text i18n.Ter) TooltipSpec
func (t TooltipSpec) Attrs(a templ.Attributes) TooltipSpec

func Separator() templ.Component
func VerticalSeparator() templ.Component

type HeadingSpec struct{ /* unexported */ }

func Heading(level int) HeadingSpec
func (h HeadingSpec) Size(s Size) HeadingSpec
func (h HeadingSpec) Attrs(a templ.Attributes) HeadingSpec
```

- **Avatar**: `<img class="ui-avatar" alt="name">` with `Src`; without
  it, `<span class="ui-avatar" role="img" aria-label="name">` showing the
  first letter of the name's first two words. `Decorative` (an avatar next
  to the name it shows) renders `alt=""`, or `aria-hidden="true"` on the
  span. Attrs go to the img or span; owned: `alt`, `role`, `aria-label`,
  `aria-hidden`, `src`, `data-size`.
- **Spinner**: `<span class="ui-spinner" role="status">` holding a
  visually hidden label, `ui.elements.spinner.loading` ("Loading") unless
  `Label` gives one. Its animation reads the duration tokens, so it stops
  under reduced motion. It fits `.htmx-indicator` (ADR 0005).
- **Tooltip**: the trigger's fill receives `aria-describedby` naming the
  text, which renders as `<span class="ui-tooltip-text" role="tooltip">`
  beside it, inside `<span class="ui-tooltip">`. CSS shows it on hover and
  on focus within. Esc hides it through the ui script, which sets `hidden`
  on the text until the pointer and focus leave, and stops the key there so
  a dialog or popover around the trigger stays open (§10.2, WCAG 1.4.13).
  A tooltip only adds a description: the trigger must have a name of its
  own. The fill is required: unfilled, `Tooltip` panics at render. Attrs go
  to the outer span.
- The text is positioned against its wrapper, not in the top layer, so a
  scrolling container can clip it. The V milestone checks that in the
  pane; if it clips in a dialog body or a table, the text becomes a manual
  popover placed as the menu's is.
- **Separator**: `<hr class="ui-separator">`. `VerticalSeparator`: `<div
  class="ui-separator" role="separator" aria-orientation="vertical">`.
- **Heading**: `<h1>`…`<h6>` with `class="ui-heading"`; `Size` sets the
  look apart from the level. A level outside 1–6 panics at render.

### 5.4 Value renderers (not spiked) [V9]

```go
func Text(t i18n.Ter) templ.Component
func Bool(v bool) templ.Component
func Date(t time.Time) templ.Component
func DateTime(t time.Time) templ.Component
func Enum(o option.SelectOption) templ.Component
func Link(to ui.Link, text i18n.Ter) templ.Component   // A5
func Image(src ui.Link, alt i18n.Ter) templ.Component // A5
func Amount(text i18n.Ter) templ.Component
func Empty() templ.Component
```

Table cells (D7) and detail rows (§7.6) both render values through these
(design §5, inventory finding 3):
- `Text`: the text, in `<span class="ui-text">`.
- `Bool`: `<span class="ui-bool" data-value="true|false">` with the
  `Check` or `X` icon and a visually hidden `ui.elements.bool.yes` or
  `…no`.
- `Date`, `DateTime`: `<time class="ui-date" datetime="…">` with the
  RFC 3339 date or date-time, and as text `i18n.StaticDate`'s form for
  `Date` and `2006-01-02 15:04` for `DateTime`, in the time's own
  location. Per-locale formats wait for i18n to offer them; a caller who
  needs one now passes its own text to `Text`.
- `Enum`: a neutral Badge with the option's label.
- `Link`: `<a class="ui-link" href>`.
- `Image`: `<img class="ui-image" src alt>`; a nil `alt` renders `alt=""`
  (decorative), never no `alt`. This is the one nil that isn't `Empty`.
- `Amount`: already formatted text (ui imports no decimal library) in
  `<bdi class="ui-amount">`, isolated so a currency stays in place in RTL,
  with tabular numerals.
- `Empty`: `<span class="ui-empty">—</span>` with a visually hidden
  `ui.elements.empty.none` ("None").

These render `Empty` instead: a nil `Ter` (but `Image`'s `alt`), or one
whose `T(ctx)` is empty or white space; a nil option; the zero
`time.Time`; a `Link` or `Image` with no address (§2.7).

### 5.5 Base classes

`ui.base` also styles, as public classes:
- `ui-visually-hidden`, text for assistive technology only;
- `.htmx-indicator`, which apps style through ui because htmx's own
  indicator style is an inline style (ADR 0005).

## 6. Inputs (`ui/inputs`) (spiked)

```go
type FieldSpec struct{ /* unexported */ }

func Field(c forms.Control) FieldSpec
func (f FieldSpec) Attrs(a templ.Attributes) FieldSpec
```

### 6.1 What each kind renders

| Kind | Element |
|---|---|
| `KindText` | `<input type="text">`, or the rule's `type` (`email`, `url`, `tel`), with `dir="ltr"` for those three and `dir="auto"` for text *(A8)* |
| `KindMultiline` | `<textarea dir="auto">` *(A8)*, with a newline before the value so a leading newline survives the parser |
| `KindPassword` | `<input type="password">`, never with its value, and a reveal button (§6.3) |
| `KindNumber` | `<input type="number">` |
| `KindDate`, `KindDateTime` | `<input type="date">`, `"datetime-local"` |
| `KindCheckbox` | `<input type="checkbox" value="true">` before its label, `checked` by the server's ticked rule |
| `KindToggle` (`!TwoState()`, `Bool`) | a `<select>` with an empty option, Yes and No wrapped in `<ui-tristate class="ui-tristate" data-ui-unset="…">` with `<template class="ui-tristate-marks">` holding `icons.Check()`, `icons.Minus()`, `icons.X()` (yes, unset, no); ui's script upgrades it to a three-position switch *(A9)* |
| `KindToggle` (`TwoState()`, `Checkbox` with `Hint(KindToggle)`) | `<input type="checkbox" role="switch" class="ui-input ui-switch" value="true">` in the checkbox's `ui-field-row`, `checked` by the server's ticked rule, styled by CSS without script *(A9)* |
| `KindSelect`, `KindMultiSelect` | `<select>` (`multiple`); a single select gets an empty first option, `ui.inputs.field.choose`, unless its options have an empty value of their own (finding 11) |
| `KindFile`, `KindFiles` | `<input type="file">` (`multiple`) |
| `KindHidden` | `<input type="hidden">`, wrapped only when it carries a condition |
| `KindGroup` | `<fieldset class="ui-group">` with `<legend class="ui-group-legend">` and its children's fields |
| `KindList` | not in this lock: `Render` returns an error naming the path. Lists come with `<ui-repeat>` in D7 |

### 6.2 Structure and wiring

Every field but a hidden one renders, in this order:

```html
<div class="ui-field" [data-ui-when="…"] [hidden]>
  <label class="ui-field-label" for="field--…">Label<span class="ui-field-required" aria-hidden="true">*</span></label>
  <p class="ui-field-help" id="field--…-help">Help</p>
  <input class="ui-input" id="field--…" name="path" aria-describedby="field--…-help field--…-error" …>
  <p class="ui-field-error" id="field--…-error"><svg class="ui-icon" …/><span class="ui-visually-hidden">Error: </span>message</p>
</div>
```

- *(not spiked)* The help sits **above** the control, so autocomplete and
  the on-screen keyboard don't cover it, and an error message starts with
  the `CircleX` icon and the hidden word `ui.inputs.field.error` ("Error:",
  punctuation in the text and the space rendered by ui, as Alert's, §11),
  so an error isn't shown by colour
  alone (the guidance check, finding 20). The spike put help below and had
  no icon: this is the first change the V milestone makes, in the goldens.
- **The other shapes.** A checkbox puts `<input>` and `<label>` in
  `<div class="ui-field-row">`, with help above the row and the error
  below it. A password field puts its `<input>` and the reveal button in a
  `ui-field-row` too. A group's fieldset holds its legend, its help (`<p
  class="ui-field-help">`, named by the fieldset's `aria-describedby`), its
  children's fields, then its error (`ui-field-error`); its condition
  and `hidden` sit on the fieldset itself. A hidden field with a condition
  is wrapped in a `<div data-ui-when>` with no class.
- Ids are generated (`ui.Scope` per field). `aria-describedby` names the
  help and the error that render. An error sets `aria-invalid="true"`.
  Error text is `errors.T(ctx, err)`.
- The asterisk follows `Required()` for every kind and is hidden from
  assistive technology; `required` carries the meaning.
- **Placeholders.** forms' `Placeholder` falls back to the label (finding
  12). A field renders `placeholder` only on text-like inputs and
  textareas, and only when its text differs from the label's, comparing
  `T(ctx)` results. See [V15] for the forms question.
- **Conditions.** A control with a condition carries it as
  `data-ui-when` (forms' JSON wire format) on its wrapper. An inactive
  control renders its wrapper `hidden`, and its control (or fieldset)
  `disabled` with the marker `data-ui-when-off`. A field in a group without
  a condition of its own follows its group's fieldset (finding 18).
- Attrs go to the control element (`<input>`, `<select>`, `<textarea>`,
  or the `<fieldset>` of a group), never the wrapper, and are merged after
  the control's own rule attributes (`maxlength`, `min`, `pattern`…).
  Owned: `id`, `name`, `type`, `required`, `disabled`, `aria-invalid`,
  `value`, `checked`, `multiple`, `data-ui-when-off`.
- Classes: `ui-field`, `ui-field-row`, `ui-field-label`,
  `ui-field-required`, `ui-field-help`, `ui-field-error`,
  `ui-field-reveal`, `ui-input`, `ui-switch` *(A9)*, `ui-tristate` *(A9)*,
  `ui-tristate-track` *(A9)*, `ui-tristate-knob` *(A9)*,
  `ui-tristate-state` *(A9)*, `ui-tristate-marks` *(A9)*, `ui-group`,
  `ui-group-legend`.
- Kinds the spike rendered: all of §6.1 but lists; the password reveal and
  the `ui-field-row` wrappers are new. File inputs are forms' `KindFile`
  as they are; D7's uploads (previews, progress) are a component of their
  own.

### 6.3 Password reveal (not spiked)

A password field renders, after its input, `<button type="button"
class="ui-field-reveal" aria-controls="…" aria-pressed="false"
aria-label="…" data-ui-reveal>` holding the `Eye` and `EyeOff` icons; CSS
shows one by `aria-pressed`. The label is `ui.inputs.field.show_password`
and stays the same while pressed, as the toggle-button pattern wants. The
ui script switches the input's type and `aria-pressed`, and puts both back
on the form's `reset`. Until the script has run (it sets
`<html data-ui-ready>`), CSS hides the button, so nobody meets a button
that does nothing.

## 7. Components (`ui/components`)

### 7.1 Dialog (spiked)

```go
var (
	DialogTrigger *ui.SlotOf[templ.Attributes]
	DialogBody    *ui.Slot
	DialogActions *ui.Slot
)

type DialogSpec struct{ /* unexported */ }

func Dialog(title i18n.Ter) DialogSpec
func (d DialogSpec) Attrs(a templ.Attributes) DialogSpec

func DialogClose(ctx context.Context, value string) templ.Attributes
```

- APG pattern: dialog (modal). It renders the trigger, then
  `<dialog class="ui-dialog" aria-labelledby>` holding
  `ui-dialog-header` (an `<h2 class="ui-dialog-title">` and a ghost close
  button labelled `ui.components.dialog.close`), `ui-dialog-body` and,
  only when filled, `<footer class="ui-dialog-actions">`.
- **The trigger.** `DialogTrigger`'s placed value is `commandfor` (the
  dialog), `command="show-modal"` and `aria-haspopup="dialog"`; the fill
  spreads it on its own button. Unfilled, the dialog renders an outline
  Button reading the title.
- **Opening and closing.** Invoker commands open it, with the ui script's
  fallback where they're missing (§10.2). Esc and the close button close
  it, and the browser returns focus to the trigger. The close button sends
  an empty `returnValue`.
- **`DialogClose(ctx, value)`** returns `commandfor`, `command="close"` and
  `value`: a button in the body or actions spread with it closes the
  dialog around it, with `value` as its `returnValue` (finding 14). Outside
  a dialog's fill it panics (the kernel's `Get`).
- *(not spiked)* CSS gives the dialog and its `::backdrop`
  `overscroll-behavior: contain`, so scrolling never reaches the page
  behind (finding 20). A tall dialog
  scrolls its body; header and actions stay.
- Attrs go to the `<dialog>`. Owned: `id`, `aria-labelledby`.
- A dialog never sits inside a menu's popover or inside the form it
  confirms (ADR 0005).

### 7.2 Confirmation (spiked) [V6, V11]

```go
func Confirm(ctx context.Context, question i18n.Ter) templ.Attributes
func ConfirmDialog() templ.Component
```

- `Confirm` returns `data-ui-confirm` holding the translated question.
  Spread on any button, link, menu item, form or element carrying `hx-*`,
  it makes the action ask first: `@elements.Button().Submit().
  Attrs(components.Confirm(ctx, q))`.
- `ConfirmDialog` is the page's one confirmation dialog, `<dialog
  id="ui-confirm" class="ui-dialog" aria-labelledby="ui-confirm-title"
  aria-describedby="ui-confirm-question" data-ui-confirm-dialog>`, with
  the title `ui.components.confirm.title` in `#ui-confirm-title`, the
  question in `#ui-confirm-question`, and two buttons: Cancel
  (`ui.components.confirm.no`, `value="cancel"`, `autofocus`, not
  spiked) and Confirm
  (`ui.components.confirm.yes`, danger tone, `value="confirm"`). The app
  and auth shells render it; a page without a shell renders it once.
- The flow is the script's (§10.3). Without the dialog on the page, the
  script falls back to `window.confirm`.

**Actions (Q6).** The spike found that confirmation needs no action type:
it is an attribute on whatever element starts the action, and one
capture-and-replay path serves links, submit buttons, htmx requests and the
app's own listeners (finding 13). So this lock has no `Action` type.
Design §5's "described once, placed anywhere" is what a table's row
actions need, placed as a toolbar button and as a menu item from one
description; it goes with D7, built on Button, MenuItem and `Confirm`.
[V6]

### 7.3 Menu (spiked) [V2]

```go
var (
	MenuTrigger *ui.SlotOf[templ.Attributes]
	MenuItems   *ui.Slot
)

type MenuSpec struct{ /* unexported */ }

func Menu(label i18n.Ter) MenuSpec
func (m MenuSpec) Attrs(a templ.Attributes) MenuSpec

type MenuItemSpec struct{ /* unexported */ }

func MenuItem() MenuItemSpec
func (m MenuItemSpec) Href(l ui.Link) MenuItemSpec // A5
func (m MenuItemSpec) Disabled(d bool) MenuItemSpec
func (m MenuItemSpec) Danger() MenuItemSpec
func (m MenuItemSpec) Submit(form string) MenuItemSpec // submits the form with that id
func (m MenuItemSpec) Attrs(a templ.Attributes) MenuItemSpec
```

- APG pattern: menu button. It renders `<ui-menu class="ui-menu">` holding
  the trigger and `<div class="ui-menu-popover" popover role="menu"
  aria-labelledby>`, whose content is `MenuItems`' fill.
- **The trigger.** `MenuTrigger`'s placed value is an id,
  `popovertarget` (the popover), `aria-haspopup="menu"` and
  `aria-controls`. Unfilled, the menu renders an outline Button with the
  label and `ChevronDown`. An icon-only trigger fills the slot with
  `Button().Label(…)` and `icons.More()`.
- **Opening.** `popovertarget` opens the popover natively (light dismiss,
  Esc, focus back to the trigger) and makes the trigger its implicit
  anchor. Where anchor positioning exists (`@supports (position-area:
  block-end)`), CSS places it under the trigger at the start edge, flipping
  on both axes near an edge (finding 9). Elsewhere `<ui-menu>` places it
  through CSSOM, flipping and **clamping to the viewport**, and follows
  scrolling and resizing while open. *(The script's placement has never
  run in a browser: the pane's Chromium anchors natively (finding 17).
  Clamping isn't written yet: the spike's fallback flips but doesn't
  clamp (finding 20).)* Firefox takes this path, so the V milestone tests
  it there, or with `@supports` forced off.
- **`<ui-menu>`** (§10.4) keeps the trigger's `aria-expanded` in step with
  the popover's `toggle` event (the server never renders it), moves focus
  to the first item on open (the last when opened with ArrowUp), handles
  ArrowUp/Down with wrapping, Home, End, Tab (closes) and typeahead,
  skips disabled items, and closes after an item is chosen.
- **MenuItem**: `<button type="button" class="ui-menu-item"
  role="menuitem" tabindex="-1">`, or `<a>` with `Href`; children are its
  text and icon. `Disabled(true)` sets `aria-disabled="true"` (and drops a
  link's `href`), so arrow keys skip it and the script stops its clicks.
  `Danger` sets `data-tone="danger"`. `Submit(form)` renders
  `type="submit" form="…"`, so the item submits a form outside the
  popover (the spike did this through `Attrs`); with `Href` it panics.
  Attrs go to the button or link. Owned: `type`, `form`, `role`,
  `tabindex`, `href`, `aria-disabled`, `data-tone`.
- `Confirm` on an item closes the menu before the dialog opens, so focus
  returns to the trigger and then to the dialog (review 0004, finding 3).
- Attrs on `MenuSpec` go to the `<ui-menu>` element.
- Menus in rows that htmx swaps in work with no setup (spiked: a table of
  menus and a row inserted after load).

### 7.4 Navigation tabs (not spiked)

```go
type TabsSpec struct{ /* unexported */ }

func Tabs(label i18n.Ter) TabsSpec
func (t TabsSpec) Attrs(a templ.Attributes) TabsSpec

type TabSpec struct{ /* unexported */ }

func Tab(to ui.Link) TabSpec // A5
func (t TabSpec) Current(c bool) TabSpec
func (t TabSpec) Attrs(a templ.Attributes) TabSpec
```

- The navigation pattern, not APG Tabs (design §6): `<nav
  class="ui-tabs" aria-label="label"><ul class="ui-tabs-list">` around
  its children. A Tab renders its own `<li class="ui-tabs-item"><a
  class="ui-tab" href>` (templ can't wrap each child); `Current(true)` adds
  `aria-current="page"`, which styles it. Children of a Tab are its text,
  a count badge included.
- Tabs scroll on the inline axis when they overflow. In-page tabs (APG
  Tabs, with panels) wait for a page that needs them.
- Attrs go to the `<nav>` and to each `<a>`; a Tab owns `href` and
  `aria-current`.

### 7.5 Toasts (not spiked) [V12, V18]

```go
func Toasts() templ.Component

type ToastSpec struct{ /* unexported */ }

func Toast(text i18n.Ter) ToastSpec
func (t ToastSpec) Tone(tone elements.Tone) ToastSpec
func (t ToastSpec) Attrs(a templ.Attributes) ToastSpec

func ToastSwap(toasts ...templ.Component) templ.Component
```

- **`Toasts`** is the region, rendered once by the shell, outside the
  landmark htmx swaps, and kept across swaps: `<ui-toasts id="ui-toasts"
  class="ui-toasts" popover="manual" aria-live="polite">`.
- **`Toast`**: `<ui-toast class="ui-toast" [data-tone]>`, a custom
  element, with the tone's icon and hidden tone word (as Alert), the text,
  and a close button `ui-toast-close` whose name is the `aria-label`
  `ui.components.toast.close`. Success and info toasts carry
  `data-ui-timeout="6000"`; warning and danger toasts stay until closed
  (WCAG 2.2.1, 2.2.4). Tones as Alert's: the four message tones only.
- **Where a toast renders.** Anywhere: in an htmx response through
  `ToastSwap`, or in a full page (a message after a redirect, from the
  app's own flash storage), where the page renders `Toast` wherever it
  likes. Each `<ui-toast>` moves itself into the region when it connects,
  so a full page's toast is inserted after load and announced like one
  that arrives by htmx. Without the script it stays where it was rendered,
  as static text.
- **`ToastSwap`** wraps toasts for an htmx response:
  `<div hx-swap-oob="beforeend:#ui-toasts">…</div>`, which htmx 2
  appends to the region. ADR 0005 says ui emits `hx-*` only where a
  component exists to make requests; a toast is how a request answers, so
  `ToastSwap` is an exception the lock would write into ADR 0005. [V18]
- **Above dialogs.** A modal `<dialog>` sits in the top layer and makes
  the rest of the page inert, so a region in the page would show behind an
  open dialog. The region is therefore a manual popover: `<ui-toasts>`
  shows it while it holds a toast, hides it when empty, and hides and
  shows it again when a toast arrives, so it sits above any modal opened
  since. Whether a popover above a modal dialog can be reached and is
  announced in every browser is the first thing the V milestone checks;
  if not, the script places arriving toasts in the open dialog instead.
- **`<ui-toasts>` and `<ui-toast>`** (§10.5) time toasts out, pausing
  while the pointer is over the region or focus is in it, and remove a
  toast whose close button is pressed. Focus then goes to the next toast's
  close button, else the previous one's, else `#ui-main`, which carries
  `tabindex="-1"` for that (§8.2).
- The guidance's `ariaNotify()` isn't used: Baseline only since September
  2026.

### 7.6 Detail list (not spiked)

```go
type DetailListSpec struct{ /* unexported */ }

func DetailList() DetailListSpec
func (d DetailListSpec) Attrs(a templ.Attributes) DetailListSpec

type DetailRowSpec struct{ /* unexported */ }

func DetailRow(label i18n.Ter) DetailRowSpec
func (d DetailRowSpec) Attrs(a templ.Attributes) DetailRowSpec
```

- `<dl class="ui-detail-list">` holding its children; each `DetailRow` is
  `<div class="ui-detail-row"><dt class="ui-detail-label">label</dt><dd
  class="ui-detail-value">children</dd></div>`. Children are a value,
  usually from §5.4.
- Rows lay out label beside value where the container is wide enough (a
  container query), label above value where it isn't.
- A detail page is a Card holding a DetailList, with actions in the
  card's header or the page header (inventory: detail page).

## 8. Layout (`ui/layout`) (not spiked)

### 8.1 Document

```go
type DocumentSpec struct{ /* unexported */ }

func Document(title i18n.Ter) DocumentSpec
func (d DocumentSpec) Head(c ...templ.Component) DocumentSpec
func (d DocumentSpec) ColorScheme(s ColorScheme) DocumentSpec
func (d DocumentSpec) BodyAttrs(a templ.Attributes) DocumentSpec
func (d DocumentSpec) Attrs(a templ.Attributes) DocumentSpec

type ColorScheme string

const (
	SystemScheme ColorScheme = ""
	LightScheme  ColorScheme = "light"
	DarkScheme   ColorScheme = "dark"
)

func ColorSchemeOf(r *http.Request) ColorScheme
```

- `Document` renders `<!doctype html><html lang dir [data-color-scheme]>`,
  a `<head>` with charset, viewport, `<title>` and the components passed
  to `Head`, in order (an app passes its `theme.Head()` and anything of
  its own), and a `<body>` holding the children. Repeated `Head` calls
  append.
- `lang` and `dir` come from `i18n.Code(ctx)`: its code, and `rtl` when
  the locale's `RTL` is set (ADR 0004).
- `ColorSchemeOf` reads the cookie `ui-color-scheme` the ui script writes
  (§10.2); anything but `light` or `dark` is the system's. Rendering the
  attribute on the server means no page flashes the wrong scheme.
- Attrs go to `<html>` (owned: `lang`, `dir`, `data-color-scheme`);
  `BodyAttrs` to `<body>`, where an htmx app puts `hx-boost` or
  `hx-headers`.

### 8.2 App shell

```go
var (
	AppBrand *ui.Slot
	AppNav   *ui.Slot
	AppBar   *ui.Slot
	AppMain  *ui.Slot
)

type AppSpec struct{ /* unexported */ }

func App() AppSpec
func (a AppSpec) Attrs(at templ.Attributes) AppSpec

type NavItemSpec struct{ /* unexported */ }

func NavItem(to ui.Link) NavItemSpec // A5
func (n NavItemSpec) Current(c bool) NavItemSpec
func (n NavItemSpec) Attrs(a templ.Attributes) NavItemSpec

type NavSectionSpec struct{ /* unexported */ }

func NavSection(label i18n.Ter) NavSectionSpec
func (n NavSectionSpec) Attrs(a templ.Attributes) NavSectionSpec

func ColorSchemeMenu(current ColorScheme) templ.Component
```

- `App` renders `<div class="ui-app">` holding, in order:
  - a skip link `<a class="ui-app-skip" href="#ui-main">`
    (`ui.layout.app.skip`), visible on focus;
  - `<header class="ui-app-bar">` with `ui-app-brand` (the `AppBrand`
    fill), the navigation toggle and `ui-app-bar-end` (the `AppBar` fill:
    search, profile menu, toggles);
  - `<nav id="ui-app-nav" class="ui-app-nav"
    aria-label="ui.layout.app.navigation">` with the `AppNav` fill;
  - `<main id="ui-main" class="ui-app-main" tabindex="-1">` with the
    `AppMain` fill;
  - then, outside `main` so a swap of `#ui-main` keeps them,
    `ConfirmDialog` and `Toasts`.
- **Narrow screens** [V17]: the toggle is `<button type="button"
  class="ui-app-nav-toggle" aria-controls="ui-app-nav"
  aria-expanded="false" aria-label="ui.layout.app.open_navigation"
  data-ui-nav-toggle>` with `icons.Menu`. CSS shows the nav as a drawer
  from the start edge while the toggle is `[aria-expanded="true"]` (with
  `:has()`, so the state is ARIA's). The ui script keeps `aria-expanded`,
  moves focus to the first item on open, and closes the drawer on Esc
  (focus back to the toggle), on a click outside it, on choosing an item,
  on `htmx:afterSwap`, and when the screen widens. It is a non-modal
  disclosure: the page behind stays reachable. Until the script has run
  (`<html data-ui-ready>`), the nav renders as a plain block above the
  content, so a page without the script still has its navigation.
- **Wide screens** (a media query): the nav is a static column and the
  toggle is hidden.
- **NavItem**: `<a class="ui-nav-item" href>` with children as its text
  and icon; `Current(true)` adds `aria-current="page"`. At render, `NavItem`
  asks `actions.Allowed(ctx, to)`: refused renders nothing; no gate in the
  context panics, naming `actions.Gate` *(A9)*. A `NavItem` with no address
  still panics first. **NavSection**:
  `<div class="ui-nav-section" role="group" aria-labelledby>` with
  `<p class="ui-nav-section-label">` and its children, the items. `NavSection`
  renders its children into a buffer and renders nothing when they wrote
  only white space *(A9)*.
- **`ColorSchemeMenu(current)`** [V13]: a Menu in the bar labelled
  `ui.layout.scheme.label`, with three items, system, light and dark
  (`ui.layout.scheme.*`), each `role="menuitemradio"` with
  `data-ui-color-scheme` and `aria-checked` rendered from `current`
  (pass `ColorSchemeOf(r)`). The script sets `<html data-color-scheme>`
  and the cookie, and keeps `aria-checked` in step.
- Attrs on `App` go to the outer `<div class="ui-app">`.

### 8.3 Auth shell

```go
var (
	AuthBrand  *ui.Slot
	AuthBody   *ui.Slot
	AuthFooter *ui.Slot
)

type AuthSpec struct{ /* unexported */ }

func Auth(title i18n.Ter) AuthSpec
func (a AuthSpec) Attrs(at templ.Attributes) AuthSpec
```

- A centred card for sign-in and its siblings: `<div class="ui-auth">`
  holding `<main id="ui-main" class="ui-auth-card" tabindex="-1">` with
  `ui-auth-brand`, `<h1 class="ui-auth-title">`, `ui-auth-body` (a form of
  `inputs.Field`s) and, when filled, `ui-auth-footer` (other links); then,
  outside `main`, `Toasts` and `ConfirmDialog`. Attrs go to the outer
  div.

### 8.4 Page header

```go
type Crumb struct {
	Href ui.Link // A5; no address (§2.7) for the current page
	Text i18n.Ter
}

type PageHeaderSpec struct{ /* unexported */ }

func PageHeader(title i18n.Ter) PageHeaderSpec
func (p PageHeaderSpec) Description(t i18n.Ter) PageHeaderSpec
func (p PageHeaderSpec) Breadcrumbs(c ...Crumb) PageHeaderSpec
func (p PageHeaderSpec) Attrs(a templ.Attributes) PageHeaderSpec
```

- `<header class="ui-page-header">` with, when given, `<nav
  class="ui-breadcrumbs" aria-label="ui.layout.page.breadcrumb">` holding
  `<ol>` of `<li class="ui-breadcrumbs-item">` links, the last one
  `aria-current="page"`, with directional separators (`ChevronEnd`,
  mirrored, hidden from assistive technology); then `<h1
  class="ui-page-title">`, `<p class="ui-page-description">`, and its
  children, the actions, in `ui-page-actions`, which renders only when
  the children write something.
- One content region, so the actions are children, not a slot (review
  0005 found the draft's `PageActions` slot broke §2.1's own rule).

## 9. Catalogue (`ui/catalog`) (spiked)

```go
type Variant struct {
	Name   string
	Render func(ctx context.Context) templ.Component
}

type Story struct {
	Name     string // "<package>/<name>", such as "elements/button"
	Variants []Variant
}

type Catalog struct{ /* unexported */ }

func New(t *theme.Theme, stories ...Story) *Catalog
func (c *Catalog) ServeHTTP(w http.ResponseWriter, r *http.Request)
func (c *Catalog) Stories() []Story
```

- `New` panics on a story name that isn't `<segment>/<segment>` of
  `[a-z0-9-]`, a duplicate story, a story without variants, or two
  variants of one story with the same name.
- The handler serves an index, a page per story and one frame per variant,
  direction and scheme: a two-by-two grid (LTR and RTL, light and dark)
  with a width switcher and a token panel that re-themes every frame
  through CSSOM, from tokens derived in Go (finding 3). Its own URLs are
  relative, so an app mounts it anywhere under `http.StripPrefix`. The
  frames link the app's theme through `t.Head()`, so the app mounts the
  theme too. It works under `script-src 'self'; style-src 'self'`.
- ui's own stories live in `internal/stories` and are served by
  `go run ./cmd/catalog`. An app registers its own stories with ui's, or
  alone.

### 9.1 catalogtest (not spiked as a package)

The spike ran these checks from a test in `internal/stories`; the
package, its options and the reverse check's split are new.

```go
package catalogtest

func Check(t *testing.T, stories []catalog.Story, opts ...Option)

type Option func(*options)

func CSS(stylesheets ...string) Option // the app's own CSS, checked with ui's
func Prefix(prefixes ...string) Option // class prefixes the selector checks cover; "ui-" always
func External(ids ...string) Option    // ids a variant may reference outside itself
```

`Check` runs, as a subtest per variant named `<package>-<name>-<variant>`:
- `uitest.Render` (with `ui.Sequence`), then `uitest.Golden` at
  `testdata/<package>-<name>-<variant>.golden.html`;
- `uitest.CheckIDs`, with `External`;
- `uitest.CheckNames` (§13.1);
- `uitest.CheckSelectors` against ui's CSS and the `CSS` stylesheets,
  for the prefixes (§13.1).

Then, once for all variants, it fails for each rule in the `CSS`
stylesheets that selects a class of a covered prefix no story renders. ui's
own catalogue test also runs that reverse check on ui's CSS, by setting a
switch in `ui/internal/themehook`; an app's catalogue can't, because it
renders only some of ui. A variant whose markup differs in RTL is its own variant,
rendering with an RTL locale in its context. [V14]

## 10. The ui script contract

`ui.js` (ES2022 modules, served by `theme`, budget 32 kB gzipped for all
modules, measured in CI *(A9, Q8)*) reads the attributes and elements below. They are
public API: an app may write them without ui's Go helpers, and changing
their meaning is an amendment.

### 10.1 Loading

- `Head` loads it as a module. It sets itself up on load and works on
  content htmx swaps in, through delegated listeners and custom elements'
  `connectedCallback`; it never needs an init call.
- When it has run it sets `<html data-ui-ready>`, which CSS reads to
  hide controls that need it (the password reveal) and to turn the
  narrow-screen navigation into a drawer.
- It emits and needs nothing inline. It listens to htmx's
  `htmx:confirm` and `htmx:afterSwap` when htmx is on the page, and works
  without htmx.
- *(A8)* htmx swaps only the body of a boosted page, so after a boosted
  swap the script copies `lang` and `dir` from the response's `<html>`
  start tag to the page's (nothing else, and an attribute the response
  lacks is removed). It does the same when htmx loads a page for
  back/forward, and puts back what it remembered for a path that htmx's
  history cache restores. *(Neither htmx path has run in a browser: the catalogue
  loads no htmx (finding 17). The V milestone adds an htmx story.)*

### 10.2 Delegated listeners (stateless)

| Hook | Behaviour |
|---|---|
| `[aria-disabled="true"]` | a capturing click listener stops the click on it or inside it before any other listener, htmx included (spiked) |
| `button[commandfor][command]` | where `commandfor` is missing, does what the browser would for `show-modal`, `close`, `request-close`, `toggle-popover`, `show-popover`, `hide-popover`, passing the button's `value` to `close` and
`request-close` as the `returnValue`; returns early where it exists, leaves custom commands alone, and ignores a target that can't take the command (spiked; not run in a browser without commands, finding 17) |
| `[data-ui-when]` | the condition evaluator (forms' semantics, passes `forms/testdata/cond` through goja, ADR 0006): on load, `input`, `change`, `reset`, `pageshow` (a page restored from
the back-forward cache, *not spiked*) and `htmx:afterSwap` it shows or hides each wrapper and disables what it hides, marking it `data-ui-when-off`; it enables only what carries the marker (spiked) |
| `[data-ui-reveal]` | switches the password input named by `aria-controls` between `password` and `text`, and `aria-pressed` with it; the form's `reset` puts both back (not spiked) |
| `[data-ui-color-scheme]` | sets `<html data-color-scheme>` and the cookie `ui-color-scheme` (path `/`, a year, `SameSite=Lax`); `system` clears both; keeps `aria-checked` on the scheme items in step (not spiked) |
| `[data-ui-nav-toggle]` | the app shell's drawer (§8.2): `aria-expanded`, focus, Esc, outside click, choosing an item, `htmx:afterSwap`, and a media query turning wide (not spiked) |
| `.ui-tooltip` | Esc, while a tooltip shows for the focused or hovered trigger, sets `hidden` on its text until the pointer and focus leave, and stops the key, so a dialog or popover around it stays open (not spiked) |

### 10.3 Confirmation (spiked; the htmx path not run in a browser)

- A capturing click listener stops a click on an element carrying
  `data-ui-confirm` (not a form), closes a menu around it, asks the
  question in `dialog[data-ui-confirm-dialog]`, and on yes dispatches the
  click again with its modifier keys, which it then lets through. So a
  link follows (in a new tab with its modifiers), a submit button submits
  with its submitter, htmx requests, and the app's own listener runs.
- A form carrying the question asks on submit, unless htmx sends it.
- `htmx:confirm` asks for requests a click doesn't start
  (`hx-trigger="change"`, a form's submit); a click that already asked
  isn't asked again.
- The dialog closes before the action runs. Its own buttons answer; Esc
  cancels.

### 10.4 `<ui-menu>` (spiked)

A light-DOM custom element holding one `[popover]` and its trigger (the
element whose `popovertarget` names it). Items are its `[role^="menuitem"]`
descendants. Behaviour in §7.3. It keeps state in private fields, never in
properties that reflect attributes (finding 10).

### 10.5 `<ui-toasts>` and `<ui-toast>` (not spiked)

Light-DOM custom elements (§7.5). `<ui-toast>`, when it connects outside
the region, moves itself into it; inside, it starts its timeout if it
carries `data-ui-timeout` (milliseconds). `<ui-toasts>` pauses every
timeout while the pointer is over it or focus is in it, shows its popover
while it holds a toast and raises it when one arrives, and removes a toast
whose `.ui-toast-close` is pressed, moving focus as §7.5 says. Both set up
in `connectedCallback`, so toasts an out-of-band swap appends work: each
toast is an element of its own, not a child the region has to notice.

### 10.6 `<ui-tristate>` *(A9)*

A light-DOM custom element (`theme/js/tristate.js`) wrapping a `Bool`
field's `<select>` and `<template class="ui-tristate-marks">` (holding the
`Check`, `Minus` and `X` icons in yes, unset, no order), with
`data-ui-unset` carrying the "no answer" word (`ui.inputs.toggle.unset`).
When it connects:
- It adds `<button type="button" class="ui-tristate-track" role="checkbox"
  aria-checked="true|false|mixed" data-state="yes|no|unset">` holding
  `<span class="ui-tristate-knob" aria-hidden="true">` (with a clone of the
  state's mark) and `<span class="ui-tristate-state">` (visually hidden,
  holding the state's word), plus a polite live region
  (`<span class="ui-visually-hidden" role="status">`).
- The button copies the select's `aria-describedby` plus the state word's
  id, `aria-invalid`, and `aria-required="true"` (removing `required` from
  the select); the `<label>`'s `for` moves to the button. The select stays
  in the form with `ui-visually-hidden`, `tabindex="-1"`,
  `aria-hidden="true"`, and a `MutationObserver` mirrors its `disabled`
  attribute onto the button.
- Positions run No (start), no answer (middle), Yes (end), mirrored in RTL;
  pointer clicks pick the third (label clicks cycle to the next state),
  Space cycles unset → yes → no → unset, ArrowLeft/ArrowRight move towards
  their side and stop at the ends (RTL-aware), Home is No, End is Yes, and
  Delete/Backspace clear to unset.
- Each change sets the select's value, dispatches bubbling `input` and
  `change` on it, and updates the live region; the switch re-reads the
  select on `input`/`change` and after the form's `reset` (in a task).
  Upgrading already-upgraded markup (an htmx history snapshot) replaces the
  track and live region idempotently, and disconnecting cleans up form and
  label listeners and the observer.

### 10.7 Not in this lock

`<ui-repeat>` (list rows) and `<ui-combobox>` (the remote select) come
with D7.

## 11. ui's i18n keys [V3]

| Key | English |
|---|---|
| `ui.elements.alert.info` | Information: |
| `ui.elements.alert.success` | Success: |
| `ui.elements.alert.warning` | Warning: |
| `ui.elements.alert.danger` | Error: |
| `ui.elements.bool.yes` | Yes |
| `ui.elements.bool.no` | No |
| `ui.elements.empty.none` | None |
| `ui.elements.spinner.loading` | Loading |
| `ui.inputs.field.choose` | Choose… |
| `ui.inputs.field.yes` | Yes |
| `ui.inputs.field.no` | No |
| `ui.inputs.toggle.unset` *(A9)* | No answer |
| `ui.inputs.field.error` | Error: |
| `ui.inputs.field.show_password` | Show password |
| `ui.components.dialog.close` | Close |
| `ui.components.confirm.title` | Are you sure? |
| `ui.components.confirm.yes` | Confirm |
| `ui.components.confirm.no` | Cancel |
| `ui.components.toast.close` | Dismiss |
| `ui.layout.app.skip` | Skip to content |
| `ui.layout.app.navigation` | Main |
| `ui.layout.app.open_navigation` | Open navigation |
| `ui.layout.page.breadcrumb` | Breadcrumb |
| `ui.layout.scheme.label` | Theme |
| `ui.layout.scheme.system` | System |
| `ui.layout.scheme.light` | Light |
| `ui.layout.scheme.dark` | Dark |

The hidden level words end with their punctuation, so each locale
writes its own ("Erreur :"); ui renders a space after them.

Declared errors: `errors.theme.color` ("must be a colour such as
#2f5bd3"), `errors.theme.font` ("must be a list of font families"),
`errors.theme.radius` ("must be 0 or a length up to 32px or 2rem"), in
errors' convention (lower case, no final period; A1). Key names are
permanent from the lock: renaming one orphans its translations.

The spike's keys (`ui.dialog.close`, `ui.confirm.*`, `ui.inputs.choose`,
`ui.inputs.yes`, `ui.inputs.no`) are renamed to these; nothing has
shipped with them.

## 12. Behaviour in detail

### 12.1 Rendering

- Every spec renders the same markup for the same inputs and context, so
  goldens are stable under `ui.Sequence`.
- A component never renders a state the browser changes later
  (`aria-expanded` on a menu trigger, a popover's open state). The states
  it does render are true when rendered and change only through ui's
  script: `aria-pressed` on a reveal button, `aria-expanded` on the app
  shell's navigation toggle, `aria-checked` on the colour scheme items.
- Every `<button>` ui renders states its `type`.
- A disabled link renders no `href`, so neither the browser nor htmx
  follows it; the script stops its clicks.

### 12.2 CSS

- The checks of ADR 0004 (token-only values, logical sides, `!important`
  once, at-rule statements refused) and the selector check (§13.1) run on
  ui's CSS in `go test`.
- Variant attributes and classes in the goldens all have a rule (spiked
  for Button, field, Dialog, Menu).

### 12.3 Browser support

As ADR 0004: popover (newly available) is used without a fallback; anchor
positioning (limited) sits behind `@supports` with the script's placement
elsewhere; invoker commands have the script's fallback. Keyboard and focus
behaviour waits for the browser test job (ADR 0005), and until then is
listed in each component's doc comment. Not yet exercised in a browser
(finding 17): the invoker fallback, the menu's own placement, and the htmx
paths.

## 13. Amendments to locked parts

### 13.1 uitest (amendment A4, proposed)

```go
func CheckNames(t testing.TB, html string)
func CheckSelectors(t testing.TB, html, css string, opts ...SelectorOption)

type SelectorOption func(*selectorOptions)
func Prefix(prefixes ...string) SelectorOption
```

- **`CheckNames`** (spiked, design §7) fails for each element that needs an
  accessible name and has none: buttons (`type="image"` included), links
  with `href`, form controls other than hidden inputs, `<img>` without
  `alt` (an empty `alt` passes), `<dialog>`, and elements with a control
  role (`role="button"`, `menuitem`, `tab`, `checkbox`…). A name comes from
  text (alt and `aria-label` of what's inside included, form controls
  inside excluded), `aria-label`, `aria-labelledby` (its targets count even
  when hidden), a `<label>` (by `for` or around it), `alt`, or `title`. It
  also fails a `<fieldset>` without a non-empty `<legend>`, and two
  navigation landmarks (`<nav>`, `role="navigation"`) with the same name or
  none. It ignores `hidden` and `aria-hidden="true"` subtrees and
  `<template>` contents. A whole page or a fragment, as `Golden` reads.
- **`CheckSelectors`** (spiked) fails for each compound selector the markup
  asks of the CSS that the CSS never selects: every class of a covered
  prefix (default `ui-`), alone and with each variant attribute of §2.3 on
  the same element by value (`.ui-button[data-variant="outline"]`), and
  with other `data-*` attributes by presence (`.ui-icon[data-mirror]`).
  `data-ui-*` hooks are skipped. The CSS is read with nesting resolved.
- They join §8, and `uitest`'s imports don't change.

### 13.2 forms: `Placeholder` (no change proposed) [V15]

`Control.Placeholder` falls back to the label, so a renderer can't tell an
explicit placeholder from none (finding 12). ui works around it by
comparing texts at render. A forms amendment could add
`Control.HasPlaceholder() bool`, or change `Placeholder` to return
`(i18n.Ter, bool)` (breaking). The default is no change.

### 13.3 Backlog

`Merge` moves from the backlog into `ui/elements` (§2.2). The backlog's
gate for moving it into the kernel ("two ui packages need it") is met by
inputs and components, which both import elements anyway; [V5] decides.

### 13.4 Links (amendment A5)

Locked by epistemer on 2026-11-22, from
[server-state-design.md](server-state-design.md) §11 and ADR
[0008](adr/0008-links.md). The roadmap's item A5 builds it, before
v0.2.0 is tagged. Built on 2026-11-22 (review 0013).

- The kernel gains `Link` and `URL` (interface.md §4A).
- Eight signatures take a `ui.Link` where they took a `templ.SafeURL`,
  each marked `// A5` above: `ButtonSpec.Href` (§5.1), `AvatarSpec.Src`
  (§5.3), `Link` and `Image` (§5.4), `MenuItemSpec.Href` (§7.3), `Tab`
  (§7.4), `NavItem` (§8.2) and `Crumb.Href` (§8.4). The two that take an
  image's `src` are in on the design's S8, which stands on its default.
- §2.7 is new. Nothing these components render changes for the same
  address.

### 13.5 Actions and tables (amendment A6)

Locked by epistemer on 2026-11-22 with the defaults of T1–T11. The contract
is [server-state-interface.md](server-state-interface.md): `ui/actions`,
`ui/tables`, their test kits, and what components gain. It changes these
locked parts (its §5):

- `ButtonSpec` and `MenuItemSpec` gain `Unavailable(reason i18n.Ter)` (S21):
  `aria-disabled`, focusable, inert, described by the reason. Menus' arrow
  keys stop on unavailable items and still skip `Disabled` ones (§7.3).
- `MenuSpec` gains `ID(id string)`, a keyed id for its trigger (§7.3).
- The shells (§8.2, §8.3) render `ActionForm` and `ActionDialog` beside
  `ConfirmDialog` and `Toasts`.
- ADR 0005's list of what emits `hx-*` gains the table's state controls,
  the action placements and `Reply`.

The roadmap's T milestones build it, through v0.3.0.

### 13.6 The remote select, uploads and list fields (amendment A7)

Locked by epistemer on 2026-11-22. The contract is
[rich-inputs-interface.md](rich-inputs-interface.md). It changes these
locked parts:

- §6.1: `KindList` renders a repeatable list with `<ui-repeat>` instead of
  returning an error; `KindFile` and `KindFiles` render the upload shape;
  `KindRemote`, `KindUpload` and `KindUploads` join the table.
- §6: `inputs.WithLabelCache` and `FieldSpec.Previews`.
- §7: `components.Suggestion`, `SuggestionLimit` and `Suggestions`.
- §10.6: `<ui-combobox>` and `<ui-repeat>` are built; §10 gains the
  upload behaviours. The remote select stays on ADR 0005's list of what
  emits `hx-*` (its search goes through htmx, R4).
- §11 gains the `ui.inputs.combobox.*`, `upload.*` and `list.*` keys.

The roadmap's U milestones build it.

## 14. Questions for epistemer

epistemer took every default on 2026-11-22, so the text above is locked as written.

| # | Question | Default | Alternatives |
|---|---|---|---|
| V1 | The builder types' names (finding 7) | `…Spec`: `Button()` returns `ButtonSpec` | `…Builder`; or the type takes the noun (`Button`) and the constructor a verb (`NewButton()`), which lengthens every call site |
| V2 | The menu's trigger | `popovertarget`, as spiked: Baseline since 2024, so opening needs no script and the trigger is the implicit anchor; placement still needs `<ui-menu>`'s fallback where anchor positioning is missing (Firefox). Roles from the server, keys from `<ui-menu>`. Native `focusgroup` can come later without changing what callers write | the guidance's `commandfor` with `command="toggle-popover"` and `focusgroup="menu nomemory"`, keys from script only where `focusgroup` is missing: newer, and a browser without invoker commands then needs the script to open and to place |
| V3 | ui's key pattern | `ui.<package>.<component>.<name>` (§11): regular, and no two components' texts collide | `ui.<package>.<name>` (Q5's letter; `close` would collide between dialog and toast); `ui.<component>.<name>` (the spike's, shorter) |
| V4 | Overriding ui's texts per call | no methods: an app changes ui's texts through translations, the source locale included; only per-call texts (the question, a spinner's or icon button's label) are arguments | a method per text on each builder |
| V5 | Where `Merge` lives | `ui/elements` | the kernel, as `ui.Merge` (an amendment to the kernel's lock) |
| V6 | An `Action` type (Q6) | none in this lock: `Confirm` is attributes on any button, link, menu item or form; one action description with placements goes with the table (D7) | an `Action` now, before its first real placement |
| V7 | Tones | six constants, `Neutral` and the five accent roles; the zero tone is the component's default (Button brand, Badge and MenuItem neutral, Alert and Toast info); Alert and Toast take only the four message tones | no `Neutral` (the first draft's: review 0005 showed a neutral Cancel button couldn't be expressed); a `MenuItem.Tone` instead of `Danger` |
| V8 | Content regions | children for one region (PageHeader's actions, Badge, Alert), a slot per region for several (Card, Dialog, Menu, App, Auth), and a `SlotOf[templ.Attributes]` for a trigger (Dialog, Menu, Tooltip) | builder arguments taking `templ.Component` (`Card().Footer(c)`), which need a templ function per piece of content |
| V9 | Value renderers | plain functions returning `templ.Component`; `Date` shows `i18n.StaticDate`'s form until i18n formats per locale | builders; a date format argument now |
| V10 | The icon set | about 20 icons vendored from Lucide (ISC), with its notice; an ADR when they land | drawn ourselves; another set |
| V11 | The confirmation's texts | one title and one Confirm label for every question | a per-action confirm label (`Delete`), which needs a second attribute and a second argument |
| V12 | Toasts | success and info time out after 6 s, pausing on hover and focus; warning and danger stay until closed; one polite region | every toast times out; an assertive region for danger |
| V13 | The colour scheme switch | `ColorSchemeMenu(current)` in the app bar, with `menuitemradio` items, a cookie `ui-color-scheme`, read by `layout.ColorSchemeOf` | a two-state toggle with no "system"; the app's own control |
| V14 | Catalogue goldens | one per variant; a variant that differs in RTL is its own variant | an RTL golden for every variant |
| V15 | forms' `Placeholder` fallback | no change; ui compares texts | `Control.HasPlaceholder()` (additive); `Placeholder() (i18n.Ter, bool)` (breaking) |
| V16 | *(withdrawn)* Assets of an unknown release | a 404, as spiked (§3.3). The draft proposed serving the current files; review 0005 showed that mixes modules of two releases | — |
| V17 | The app shell's navigation on narrow screens | a non-modal drawer the ui script opens through the toggle's `aria-expanded`, closing on Esc, outside click, navigation and widening; a plain block without the script; a static column on wide screens | a modal `<dialog>` drawer (focus contained, background inert); the draft's popover drawer, which review 0005 showed keeps light dismiss armed after widening; a collapsible column of icons (zp's), which needs a stored state |
| V18 | `ToastSwap`'s `hx-swap-oob` | keep it, and write the exception into ADR 0005: a response fragment, not a request | no helper: apps write the out-of-band wrapper themselves |

**Not spiked**, collected:
- elements: Badge, Alert, Card, Avatar, Spinner, Tooltip, Separator,
  Heading, the value renderers, `Button.Label`, `Neutral`;
- icons beyond four, and the icon set's source;
- theme: the prefixed `Config`, `themeschema`, the check functions'
  errors;
- inputs: help above the control, the error icon and word, the
  `ui-field-row` wrappers, the password reveal;
- components: the dialog's overscroll containment, `autofocus` on Cancel,
  `MenuItem.Submit`, the menu's clamping, navigation tabs, toasts, the
  detail list;
- layout, all of it, with the drawer and the colour scheme switch;
- `catalogtest` as a package, `aria-controls` in `Merge`'s joined keys,
  `pageshow`, `<html data-ui-ready>`.

Spiked but not run in a browser (finding 17): the invoker-command
fallback, `<ui-menu>`'s own placement, and every htmx path. Their
milestones after D6 build them with stories, goldens and the checks, and a
browser check in the pane, before their packages are tagged.
