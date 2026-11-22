# ui: interface

The contract of `github.com/epistemer/ui` for v0.1.0: the kernel (`ui`,
`ui/uitest`) and forms (`ui/forms`, `ui/forms/rule`, `ui/forms/formstest`).
The designs and their reasons are [brief.md](brief.md) and
[forms-design.md](forms-design.md); this document states what code may rely
on. Once epistemer locks it, a change to anything here that alters what an
app writes or gets back is an amendment (§10).

**Status: locked by epistemer on 2026-11-21 (M0.2).** Items marked *(new)*
were not spelled out in either design and were proposed with this document:
formstest's API (§7), the catalogue's error keys and texts (§6.3), forms'
own error texts (§5.9), the condition operator names on the wire (§5.7),
and §9. They are locked with the rest.

## 1. Module

| | |
|---|---|
| Module | `github.com/epistemer/ui`, Go 1.27 |
| Packages | `ui` (kernel), `ui/uitest`, `ui/forms`, `ui/forms/files` *(A7)*, `ui/forms/rule`, `ui/forms/formstest` |
| Kernel imports | `github.com/a-h/templ` (v0.3.1118) and the standard library |
| uitest imports | the kernel, templ, `golang.org/x/net/html` (ADR 0003, M3.2) |
| forms imports | the standard library, `github.com/epistemer/errors`, `github.com/epistemer/i18n`, `github.com/epistemer/models/listing` *(A7)*, `github.com/epistemer/models/option`, `ui/forms/files` *(A7)*, `ui/forms/rule` |
| files imports | the standard library *(A7)* |
| rule imports | the standard library, `github.com/epistemer/errors`, `github.com/epistemer/i18n`, `ui/forms/files` *(A7)* |
| formstest imports | forms, its internal hook, `github.com/epistemer/errors`, the standard library |
| Never imported | by forms: templ, any web framework, any decimal library. By the kernel: i18n, errors, forms |

No exported signature in forms or rule contains `any`, `interface{}`, `[]any`
or `map[string]error` (I9).

## 2. Kernel: context values

```go
type Provider[T any] struct{ /* unexported */ }

func NewProvider[T any](def T) *Provider[T]

func (p *Provider[T]) Provide(v T) templ.Component
func (p *Provider[T]) With(ctx context.Context, v T) context.Context
func (p *Provider[T]) Value(ctx context.Context) T
```

- `Provide(v)` renders its block with `v` visible to everything inside it.
  `With` does the same for Go code. `Value` returns the nearest provided
  value, else the default given to `NewProvider`.
- Values live in the context, keyed by the provider (K2). Two providers of
  the same type never see each other's values (K8).
- `Provide` written directly in a slotted component's block, around fills,
  returns `ErrInBlock` (§4.3).

## 3. Kernel: element IDs

```go
type ID struct{ /* unexported; copies refer to the same handle */ }

func NewID(name string) ID

func Scope(ids ...ID) templ.Component
func (id ID) Get(ctx context.Context) string
func (id ID) Part(ctx context.Context, parts ...string) string
func (id ID) Key(parts ...string) string

type IDSource interface{ Next() string }

func WithIDs(ctx context.Context, s IDSource) context.Context
func Sequence(prefix string) IDSource
```

- **Names.** `NewID` panics unless `name` matches
  `[a-z][a-z0-9]*(-[a-z0-9]+)*`. The panic names the bad value.
- **Generated ids.** `Scope(ids...)` draws one token per id from the
  context's `IDSource` and renders its block with them. `Get` returns
  `<name>--<token>`; the nearest enclosing `Scope` of that id wins. `Get`
  outside any `Scope` of that id panics with
  `ui: <name>.Get outside ui.Scope(<name>)` and the fix. `Part` appends
  `-<escaped part>` per part.
- **Keyed ids.** `Key(parts...)` returns `<name>-<escaped part>-…`, needing
  no context. With no parts it returns `<name>`.
- **Escaping.** Every byte outside `[A-Za-z0-9]` becomes `_` and two
  lowercase hex digits; an empty part becomes `_`. Escaping is injective:
  different parts give different ids. A keyed id never contains `--`; a
  generated id always does. Every output is a valid HTML id and CSS
  identifier.
- **Sources.** The default source is one per process: a counter in base 36
  after a prefix of 6 random base-36 characters fixed at start-up.
  `Sequence(prefix)` yields `prefix-1`, `prefix-2`, … and is safe for
  concurrent use. Tokens never contain `--`; a source whose `Next` returns
  one makes `Scope` return an error.
- `Scope` written directly in a slotted component's block returns
  `ErrInBlock` (§4.3).

## 4. Kernel: slots

```go
type Slot struct{ /* unexported */ }
type SlotOf[T any] struct{ /* unexported */ }
type SlotRef interface{ /* unexported method */ }

func NewSlot() *Slot
func NewSlotOf[T any]() *SlotOf[T]

func Slotted(layout templ.Component, slots ...SlotRef) templ.Component

func (s *Slot) Render(ctx context.Context, w io.Writer) error       // fills the slot
func (s *SlotOf[T]) Render(ctx context.Context, w io.Writer) error  // fills the slot
func (s *Slot) Place() templ.Component
func (s *SlotOf[T]) Place(v T) templ.Component
func (s *Slot) Filled(ctx context.Context) bool
func (s *SlotOf[T]) Filled(ctx context.Context) bool
func (s *SlotOf[T]) Value(ctx context.Context) T
```

`*Slot` and `*SlotOf[T]` implement `templ.Component` and `SlotRef`. Each
captures the file and line of its `NewSlot`/`NewSlotOf` call, which names it
in every message.

### 4.1 Behaviour

1. `Slotted(layout, slots...)` reads and clears its children (K3), renders
   the block once into a sink that keeps nothing (the collect pass), then
   renders `layout` once to the writer.
2. In the collect pass, rendering a slot records its block as that slot's
   fill and writes nothing. Output other than white space is
   `ErrStrayContent`.
3. `Place` renders the fill if the slot was filled, even if the fill renders
   nothing; otherwise the placement's own block; otherwise nothing.
4. A fill renders in the placement's context: the layout's providers, ID
   scopes and, for `SlotOf[T]`, the placed value are visible.
5. Inside fill content, `Place` and `Filled` refer to the component whose
   author wrote that content (lexical), not to the component being filled.
6. The block runs once per render of `Slotted`; a fill once per placement;
   a fallback only when its slot is unfilled.
7. `Value` returns the value of the nearest placement of that slot; outside
   one it panics. `Filled` reports whether the current component's caller
   filled the slot; outside a layout it panics.

### 4.2 Errors

Plain sentinels (`errors.New` from the standard library), wrapped with the
slot's declaring position: test them with `errors.Is`.

| Error | Returned when |
|---|---|
| `ErrStrayContent` | The block writes anything but white space outside a fill |
| `ErrFilledTwice` | A block fills the same slot twice |
| `ErrNotDeclared` | A block fills, or a layout places, a slot not listed in `Slotted` |
| `ErrNoComponent` | A slot is filled outside any slotted component's block, or placed outside a layout |
| `ErrInBlock` | `Provide`, `Scope` or `Place` is rendered directly in a block during the collect pass |

Messages read `ui: slot declared at <file>:<line>: <what happened>`.

### 4.3 Wrappers in a block

`Provide`, `Scope` and `Place` rendered directly in a slotted component's
block return `ErrInBlock`; opened around the component call they work. A
wrapper of the app's own that only renders its block is allowed there.

## 4A. Kernel: links (amendment A5)

```go
// Link is anything with a URL: a route link, or URL's adapter.
type Link = interface{ URL() string }

// URL makes a Link of a literal or external address.
func URL(s string) Link
```

- `Link` is an alias of the bare interface, so a function another package
  declares over `interface{ URL() string }` is assignable where ui wants
  one over `Link`, with no import either way.
- `URL(s).URL()` is `s`. `URL("")` is a link with no address.
- The visual packages take a `Link` wherever they take an address, and
  say what no address means ([visual-interface.md](visual-interface.md)
  §2.7). The kernel's imports don't change.

## 5. Forms (`ui/forms`)

### 5.1 Form and lifecycle

```go
type Form struct{ /* unexported */ }

func New() *Form

func (f *Form) Bind(src Source)
func (f *Form) Validate(ctx context.Context) bool
func (f *Form) Submit(ctx context.Context, r *http.Request) bool

func (f *Form) Valid() bool
func (f *Form) Err() error
func (f *Form) Errors() iter.Seq[*FieldError]
func (f *Form) Fail(err error)
func (f *Form) Check(fn func(ctx context.Context) error)

func (f *Form) Control() Control
func (f *Form) Controls() iter.Seq[Control]

// On a sub-form (a group); legal on the root but meaningless there.
func (f *Form) Label(l i18n.Ter) *Form
func (f *Form) When(c Cond) *Form
func (f *Form) Active() bool
```

- **Declare.** Constructors read nothing. Programmer mistakes panic with a
  message naming the field: a duplicate or malformed name, a condition
  cycle, a builder call after `Bind`, a handle used after `As`, a second
  `Bind`.
- **Bind** records each field's submitted text in declaration order,
  decides presence, runs the conversion chain; a conversion failure becomes
  that field's error. No rule runs. Undeclared names are ignored.
- **Validate**: resolve which nodes are active (an inactive node's value,
  text and error are cleared); an active field with a conversion error keeps
  it; an absent active field gets its required error if required, else its
  rules are skipped; a present active field runs its rules in order and
  stops at the first failure; if no field failed, `Check` functions run in
  order and the first failure is the form-level error. "Failed" means any
  node: a field, a group's check or a list's limits (§9.2). Returns
  `Valid()`.
- **Submit** is `FromRequest`, `Bind`, `Validate`. An unreadable request
  records `ErrUnreadable` on the form and returns false.
- `Fail(err)` attaches a form-level error at any time; `Valid()` is then
  false.
- A form is not safe for concurrent use. One instance serves one request.

### 5.2 Source

```go
type Source interface {
	Values(name string) []string
	Files(name string) []*multipart.FileHeader
	Names() iter.Seq[string]
}

func FromRequest(r *http.Request) (Source, error)
func FromValues(v url.Values) Source
```

`FromRequest` parses the body once (multipart or urlencoded) and, for a GET,
the query.

### 5.3 Fields

```go
type Field[T any] struct{ /* unexported */ }

func (f *Form) Text(name string) *Field[string]
func (f *Form) Texts(name string) *Field[[]string]
func (f *Form) Int(name string) *Field[int]
func (f *Form) Float(name string) *Field[float64]
func (f *Form) Bool(name string) *Field[bool]
func (f *Form) Checkbox(name string) *Field[bool]
func (f *Form) Time(name, layout string) *Field[time.Time]
func (f *Form) Date(name string) *Field[time.Time]
func (f *Form) File(name string) *Field[*multipart.FileHeader]
func (f *Form) Files(name string) *Field[[]*multipart.FileHeader]
func (f *Form) Upload(name string, stored []files.Stored) *Field[files.Upload]  // amendment A7
func (f *Form) Uploads(name string, stored []files.Stored) *Field[files.Upload] // amendment A7

func (f *Form) Number[T Integer | Float](name string) *Field[T]
func (f *Form) Enum[E EnumValue](name string, values []E) *Field[E]
func (f *Form) Parsed[T any, PT interface {
	*T
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string) *Field[T]

type Integer interface{ ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 }
type Float interface{ ~float32 | ~float64 }

type EnumValue interface {
	~string
	option.SelectOption
}
```

- Names match `[a-z][a-z0-9_]*`. `Time` and `Date` parse in UTC; `Date`'s
  layout is `2006-01-02`.
- `Enum` accepts only a submitted text equal to some value's `Value()`, and
  supplies the field's options from `values`.
- M0.3's spike (Go 1.27.4) confirmed every generic method here: explicit
  instantiation on a method promoted through an embedded `*Form`
  (`f.Number[int32]("n")`), inference of `PT` in `Parsed`, `As[U]` on
  `Field[T]`, `Group` inferring `G`, method values (`is := f.Type.Eq`), and
  `Enum` over `~string` plus `option.SelectOption`.
- A bound's type is not inferred from the field: `f.Int("n").Is(rule.Min(0))`
  compiles, but a `Field[int32]` needs `rule.Min[int32](1)` or
  `rule.Min(int32(1))`.
- **Uploads** *(amendment A7, [rich-inputs-interface.md](rich-inputs-interface.md) §2.2)*:
  `ui/forms/files` (standard library only) declares:

  ```go
  package files // ui/forms/files

  // Stored is a file the record already has.
  type Stored struct {
  	Key   string                    // the app's key, posted back to keep it
  	Name  string                    // shown
  	Href  interface{ URL() string } // where it opens; nil for none
  	Image bool                      // Href is an image: shown as a thumbnail
  }

  // Upload is an upload field's value.
  type Upload struct {
  	Kept  []Stored                // stored files still kept, in the order given
  	Added []*multipart.FileHeader // files chosen now
  }
  ```

  Before `Bind`, an upload field's value is `files.Upload{Kept: stored}` (a
  copy), present when `stored` isn't empty. Kept files post as
  `<path>.keep=<key>`; new files under `<path>`. A posted key not among
  `stored` is dropped; a repeated key counts once; `Kept` keeps `stored`'s
  order. `Upload` (`KindUpload`) holds one file in all: with
  `len(stored) > 1` it panics at declaration; of several files posted it
  takes the first; an added file empties the value's `Kept`, while the
  control still reports the posted keep state. `Uploads` is `KindUploads`.
  `Required` holds with one kept or added file. `Set`, `SetPtr`, `Options`
  and `Hint` on an upload field panic; `Options` on `File` or `Files`
  panics too. `Upload` or `Uploads` declared in a list's item panics;
  `File` and `Files` are legal there. `stored` with an empty or repeated
  `Key`, or an empty `Name`, panics. An oversized body in `Submit` is
  reported as `ErrUnreadable` wrapping `*http.MaxBytesError`.

**Presence after Bind.**

| Constructor | Absent when |
|---|---|
| `Text` | the name is missing or the value is empty after trimming (white space is trimmed unless `.Raw()`) |
| `Texts`, `Files` | no non-empty item was submitted |
| `Int`, `Float`, `Number`, `Time`, `Date`, `Enum`, `Parsed`, `Bool` | the name is missing or the value is empty |
| `Checkbox` | never: a missing name is false; `Required` means it must be ticked |
| `File` | no file was uploaded under the name |
| `Upload`, `Uploads` *(A7)* | `Kept` and `Added` hold no file between them |

### 5.4 Builders

All return the field and are legal until `Bind`.

```go
func (f *Field[T]) Label(l i18n.Ter) *Field[T]
func (f *Field[T]) Placeholder(p i18n.Ter) *Field[T]
func (f *Field[T]) Help(h i18n.Ter) *Field[T]
func (f *Field[T]) Raw() *Field[T] // Text only: keep surrounding white space

func (f *Field[T]) Required() *Field[T]
func (f *Field[T]) RequiredWith(err error) *Field[T]
func (f *Field[T]) InvalidWith(err error) *Field[T]

func (f *Field[T]) Is(rules ...rule.Rule[T]) *Field[T]
func (f *Field[T]) When(c Cond) *Field[T]
func (f *Field[T]) Options(opts ...option.SelectOption) *Field[T]
func (f *Field[T]) Hint(k Kind) *Field[T]
func (f *Field[T]) Remote(search interface{ URL() string }, l listing.Labeler) *Field[T] // amendment A7

func (f *Field[T]) As[U any](c Conv[T, U]) *Field[U]
```

- Label defaults to `i18n.Static(name)`; placeholder to the label.
- `Options` validates membership by comparing the submitted text with each
  option's `Value()` (error `ErrNotAnOption`), and supplies the choices.
- `Raw` on a non-`Text` field panics.
- `Remote` *(amendment A7, [rich-inputs-interface.md](rich-inputs-interface.md) §2.1)*
  makes the field a remote select (`KindRemote`): its value is a record's
  id, searched at `search` and named by `l`. forms never calls `l`. Legal
  on a field whose kind is `KindText` or an integer `KindNumber` (`Text`,
  `Int`, `Number[T]` of an integer type, a `Parsed` id type); it panics on
  any other field, with a nil link or `Labeler`, combined with `Options`
  or `Hint` in either order, and after `Bind`. `Hint` with `KindRemote`,
  `KindUpload` or `KindUploads` panics. Every `Labeler` an app passes
  answers `false`, not an error, for an id that can't name a row.

### 5.5 State

```go
func (f *Field[T]) Set(v T) *Field[T]
func (f *Field[T]) SetPtr(p *T) *Field[T]
func (f *Field[T]) Value() T
func (f *Field[T]) Get() (T, bool)
func (f *Field[T]) Ptr() *T
func (f *Field[T]) Err() error
func (f *Field[T]) Fail(err error)
func (f *Field[T]) Control() Control
```

`Set` gives an initial value and marks the field present; `SetPtr(nil)` does
nothing. `Value` is the zero `T` when the field is absent, inactive or in
error; `Ptr` is nil then. After a failed conversion the field keeps the
submitted text for redisplay.

### 5.6 Conversion

```go
type Conv[T, U any] struct {
	Parse  func(T) (U, error)
	Format func(U) T
}
```

`As` appends a stage: rules added before it check `T`, rules after check `U`.
The handle `As` was called on is dead; using it panics. Label, placeholder,
help, required-ness, `When`, `Options` and the kind carry over. A `Parse`
error that implements `i18n.Ter` is shown as it is; any other is wrapped as
the cause of the field's invalid error.

### 5.7 Conditions

```go
type Cond struct{ /* unexported */ }

func (f *Field[T]) Eq(v T) Cond
func (f *Field[T]) In(vs ...T) Cond
func (f *Field[T]) Filled() Cond

func All(cs ...Cond) Cond
func Any(cs ...Cond) Cond
func Not(c Cond) Cond

func (c Cond) MarshalJSON() ([]byte, error)
```

- `Eq` holds when the field is active, present, and its text (what was
  submitted, trimmed as Bind trims it; before Bind, the formatted initial
  value) equals `v` formatted by the field's codec at the stage `Eq` was
  called on. This is the string a client evaluator sees, so the two sides
  agree ("01" is not `Eq(1)` on either). A field in conversion error is
  still compared by its text. `In` holds when `Eq` holds for one of `vs`.
  `Filled` holds when the field is active and present. On an inactive or
  absent field all three are false.
- `When(c)` makes a field, group or list active only while `c` holds.
  Conditions resolve after `Bind`, on demand, memoised. A cycle panics at
  the `When` call that closes it. A condition may read a field of the same
  form or an enclosing one, never into a list's items from outside (panic).
- **Wire format** *(operator names new)*:
  - a comparison is `{"typ":"c","k":"<path>","op":"eq","v":"<text>"}`;
  - `Filled` is `{"typ":"c","k":"<path>","op":"filled"}`;
  - `All`, `Any` and `Not` are `{"typ":"l","op":"and"|"or"|"not","args":[…]}`;
  - `In` is an `or` of `eq` nodes.

  `testdata/cond/*.json` holds conformance vectors (condition, submitted
  values, expected result); any client evaluator must pass them.

### 5.8 Groups and lists

```go
func (f *Form) Group[G any](name string, build func(g *Form) G) G
func (f *Form) List[G any](name string, build func(g *Form) G) *List[G]

type List[G any] struct{ /* unexported */ }

func (l *List[G]) Label(t i18n.Ter) *List[G]
func (l *List[G]) When(c Cond) *List[G]
func (l *List[G]) MinItems(n int) *List[G]
func (l *List[G]) MaxItems(n int) *List[G]
func (l *List[G]) Items() []G
func (l *List[G]) Add() G
func (l *List[G]) Err() error
func (l *List[G]) Control() Control

const DefaultMaxItems = 256
const IndexPlaceholder = "__i__"
```

- **Paths.** A root field posts as `name`, a group member as
  `group.name`, a list item's field as `list[i].name`, deeper as
  `a.b[2].c`.
- **Binding a list.** Indices are the distinct one-to-six-digit decimals
  after `<path>[` among `Source.Names()`, sorted numerically, gaps allowed.
  Each gets a sub-form built and bound, and items are renumbered `0..n-1`.
  More than `MaxItems` (or `DefaultMaxItems`) is `ErrMaxItems` on the list,
  and the extra items are not built.
- The list's control exposes a template row built at `IndexPlaceholder`.
- **A new form's list** *(amendment A7)*: `Control().Children()` of a list
  that was neither bound nor given items by `Add` yields `MinItems` blank
  items, built at indices `0..MinItems-1` as the template row is. The list
  itself isn't changed: `Items()` stays empty and `Add` appends as before.
- **An item that posts nothing** *(amendment A7)*: an index counts only
  through a member's name. A row whose controls all post nothing when
  empty (unchecked checkboxes, a multi-select with nothing chosen,
  controls a condition disabled) is dropped on bind, before its rules run.
  Text inputs, selects (`Bool`, `Enum`, a single `Options`), and file
  inputs always post; a list whose rows could hold only the first kind
  adds a field of the second.

### 5.9 Errors

```go
type FieldError struct {
	Path  string
	Label i18n.Ter
	Err   error
}

func (e *FieldError) Error() string // "<path>: <err>"
func (e *FieldError) Unwrap() error
```

`Form.Err()` is nil when valid, else a join of one `*FieldError` per failing
node: the form-level error (path `""`) first, then nodes in declaration
order. `Errors()` yields the same. Each node holds at most one error.

Forms' declared errors, all of kind `errors.Invalid` *(texts new)*:

| Variable | Key | Text |
|---|---|---|
| `ErrUnreadable` | `errors.forms.unreadable` | `the form could not be read` |
| `ErrInvalid` | `errors.forms.invalid` | `is not valid` |
| `ErrNotAnOption` | `errors.forms.not_an_option` | `is not one of the choices` |
| `ErrRequired` | `errors.forms.required` | `is required` |
| `ErrMinItems` (`Key1[int]`) | `errors.forms.min_items` | `add at least %d` |
| `ErrMaxItems` (`Key1[int]`) | `errors.forms.max_items` | `add at most %d` |

### 5.10 Control and Kind

```go
type Control struct{ /* unexported */ }

func (c Control) Kind() Kind
func (c Control) Path() string
func (c Control) Label() i18n.Ter
func (c Control) Placeholder() i18n.Ter
func (c Control) Help() i18n.Ter
func (c Control) Required() bool
func (c Control) Active() bool
func (c Control) When() (Cond, bool)
func (c Control) Text() []string
func (c Control) Options() iter.Seq[Choice]
func (c Control) Attrs() iter.Seq2[string, string]
func (c Control) Err() error
func (c Control) Children() iter.Seq[Control]
func (c Control) Template() (Control, bool)
func (c Control) Remote() (search interface{ URL() string }, l listing.Labeler, id string, ok bool) // amendment A7
func (c Control) Stored() iter.Seq2[files.Stored, bool]                                             // amendment A7
func (c Control) Lost() bool                                                                        // amendment A7
func (c Control) Bounds() (min, max int)                                                            // amendment A7

type Choice struct {
	Value    string
	Label    i18n.Ter
	Selected bool
}

type Kind uint8

const (
	KindText Kind = iota + 1
	KindMultiline
	KindPassword
	KindHidden
	KindNumber
	KindCheckbox
	KindToggle
	KindDate
	KindDateTime
	KindSelect
	KindMultiSelect
	KindFile
	KindFiles
	KindGroup
	KindList
	KindRemote  // amendment A7
	KindUpload  // amendment A7
	KindUploads // amendment A7
)

func (k Kind) String() string
```

Defaults: `Text`, `Parsed` → `KindText`; `Int`, `Float`, `Number` →
`KindNumber`; `Checkbox` → `KindCheckbox`; `Bool` → `KindToggle`; `Date` →
`KindDate`; `Time` → `KindDateTime`; `Enum` or any field with options →
`KindSelect` (`Texts` with options → `KindMultiSelect`); `File`, `Files` →
`KindFile`, `KindFiles`; `Upload`, `Uploads` → `KindUpload`, `KindUploads`
*(A7)*; `Remote` → `KindRemote` *(A7)*; groups and lists → `KindGroup`,
`KindList`. `Hint` overrides. `Text()` is the submitted text if bound,
else the formatted value (`nil` for file and upload fields). `Attrs` come
from the field's rules (`maxlength`, `minlength`, `min`, `max`, `pattern`,
`accept`, `type`, `data-ui-max-size`), plus `inputmode="numeric"` for an
integer `KindRemote` field *(A7)*.

- `Remote()` *(amendment A7)* gives a remote field's search link and
  `Labeler`, and `id`: the value formatted, or `""` when the field is
  absent or its text didn't convert. `ok` is false for any other field.
- `Stored()` *(amendment A7)* yields each stored file with its keep state:
  posted, after `Bind`; `true` before. For any other control, or one with
  no stored files, it is an empty iterator, never nil.
- `Lost()` *(amendment A7)* is evaluated when called: true for an active
  file field (`KindFile`, `KindFiles`, `KindUpload`, `KindUploads`) that
  was posted at least one file, has no error of its own, and whose root
  form has an error (`Form.Err() != nil`).
- `Bounds()` *(amendment A7)* gives a list's effective bounds (`max` is
  `DefaultMaxItems` when `MaxItems` wasn't set), and zeros for any other
  control.

## 6. Rules (`ui/forms/rule`)

### 6.1 The type

```go
type Rule[T any] struct{ /* unexported */ }

func New[T any](check func(ctx context.Context, v T) error) Rule[T]
func (r Rule[T]) Err(err error) Rule[T]
func (r Rule[T]) Attrs() iter.Seq2[string, string]
func Check[T any](ctx context.Context, v T, rules ...Rule[T]) error
```

A rule never sees an absent value. `Err` returns a copy reporting `err` on
failure. `Check` runs rules in order and returns the first failure.

### 6.2 Catalogue

| Applies to | Constructor | Attrs |
|---|---|---|
| `string` | `MinLen(n int)`, `MaxLen(n int)` (runes) | `minlength`, `maxlength` |
| `string` | `Matches(re *regexp.Regexp)` | `pattern` when `re` is valid in JavaScript too |
| `string` | `Email()`, `URL()`, `Phone()` | `type` = `email`, `url`, `tel` |
| `string` | `CountryCode()`, `CurrencyCode()`, `PostCode(country string)` | |
| `T cmp.Ordered` | `Min(v T)`, `Max(v T)`, `Between(lo, hi T)` | `min`, `max` for numbers |
| `T interface{ Cmp(T) int }` | `AtLeast(v T)`, `AtMost(v T)` | |
| `T comparable` | `OneOf(vs ...T)` | |
| `time.Time` | `NotBefore(t time.Time)`, `NotAfter(t time.Time)` | `min`, `max` as dates |
| `*multipart.FileHeader` | `MinSize(bytes int64)`, `MaxSize(bytes int64)`, `ContentType(types ...string)` | `data-ui-max-size` *(A7)*, `accept` |
| `files.Upload` *(A7)* | `Files(rules ...Rule[*multipart.FileHeader])`, `MinFiles(n int)`, `MaxFiles(n int)` | inner rules' attributes (`Files`) |
| `[]T` | `MinItems(n int)`, `MaxItems(n int)`, `Each(rules ...Rule[T])` | |

`Files` checks each added file and passes its inner rules' attributes on;
`MinFiles` and `MaxFiles` count kept plus added files *(amendment A7)*.
`Each` reports the first failing item's error. `MinItems`, `MaxItems` and
`Each` take `T` from no argument, so Go cannot infer it: write
`rule.MinItems[string](1)` (checked on Go 1.27.4).

### 6.3 Default errors *(keys and texts new)*

All of kind `errors.Invalid`. A bound is passed to the text as a string,
formatted with `fmt.Sprint`; a date as `i18n.StaticDate`, so it renders in
the request's locale.

| Variable | Key | Text |
|---|---|---|
| `ErrMinLen` (`Key1[int]`) | `errors.forms.rule.min_len` | `must be at least %d characters` |
| `ErrMaxLen` (`Key1[int]`) | `errors.forms.rule.max_len` | `must be at most %d characters` |
| `ErrMatches` | `errors.forms.rule.matches` | `is not in the expected format` |
| `ErrEmail` | `errors.forms.rule.email` | `must be an email address` |
| `ErrURL` | `errors.forms.rule.url` | `must be a URL` |
| `ErrPhone` | `errors.forms.rule.phone` | `must be a phone number` |
| `ErrCountryCode` | `errors.forms.rule.country_code` | `must be a country code` |
| `ErrCurrencyCode` | `errors.forms.rule.currency_code` | `must be a currency code` |
| `ErrPostCode` | `errors.forms.rule.post_code` | `must be a postal code` |
| `ErrMin` (`Key1[string]`) | `errors.forms.rule.min` | `must be at least %s` |
| `ErrMax` (`Key1[string]`) | `errors.forms.rule.max` | `must be at most %s` |
| `ErrBetween` (`Key2[string, string]`) | `errors.forms.rule.between` | `must be between %s and %s` |
| `ErrOneOf` | `errors.forms.rule.one_of` | `is not one of the allowed values` |
| `ErrNotBefore` (`Key1[i18n.Ter]`) | `errors.forms.rule.not_before` | `must not be before %s` |
| `ErrNotAfter` (`Key1[i18n.Ter]`) | `errors.forms.rule.not_after` | `must not be after %s` |
| `ErrMinSize` (`Key1[string]`) | `errors.forms.rule.min_size` | `must be at least %s` |
| `ErrMaxSize` (`Key1[string]`) | `errors.forms.rule.max_size` | `must be at most %s` |
| `ErrContentType` (`Key1[string]`) | `errors.forms.rule.content_type` | `must be one of these file types: %s` |
| `ErrMinFiles` (`Key1[int]`, A7) | `errors.forms.rule.min_files` | `choose at least %d` |
| `ErrMaxFiles` (`Key1[int]`, A7) | `errors.forms.rule.max_files` | `choose at most %d` |
| `ErrMinItems` (`Key1[int]`) | `errors.forms.rule.min_items` | `choose at least %d` |
| `ErrMaxItems` (`Key1[int]`) | `errors.forms.rule.max_items` | `choose at most %d` |

`AtLeast` and `AtMost` report `ErrMin` and `ErrMax`. File sizes are written
in the largest of B, KB, MB, GB that keeps the number whole (`5 MB`).

## 7. formstest *(new)*

```go
// Submit binds values (a query string such as "name=Gold&type=fixed") and
// validates. A malformed query string fails t.
func Submit(t testing.TB, ctx context.Context, f *forms.Form, values string) bool

// Errors lists the form's errors in Err's order, one "<path>: <key>" per
// failing node, with <key> the i18n key of the public error or "-".
func Errors(f *forms.Form) []string

// AssertLabelled fails t for every control, at any depth, whose label is
// still the default built from its wire name.
func AssertLabelled(t testing.TB, f *forms.Form)
```

Files are tested through `forms.Bind` with a `Source` of the test's own.

## 8. uitest

```go
func Render(t testing.TB, c templ.Component, opts ...Option) string
func RenderErr(c templ.Component, opts ...Option) (string, error)

type Option func(*options)
func Context(f func(context.Context) context.Context) Option

func Golden(t testing.TB, name, html string)
func CheckIDs(t testing.TB, html string, opts ...RefOption)

type RefOption func(*refOptions)
func External(ids ...string) RefOption
```

- `Render` and `RenderErr` render with `ui.Sequence("t")`; a render error
  fails `t` in `Render`.
- `Golden` compares the normalised HTML (one element per line, attributes
  sorted, text trimmed) with `testdata/<name>.golden.html`; with
  `go test -update` it rewrites the file. uitest registers `-update` unless
  a package initialised before it already defined a flag of that name, in
  which case it reads that one. A test package that imports uitest reads
  uitest's flag (`flag.Lookup("update")`) rather than declaring its own
  (A3).
- `CheckIDs` fails for a duplicate id and for any reference that does not
  resolve, naming the attribute and the id. It follows `for`, `form`,
  `list`, `headers`, `popovertarget`, `commandfor`, `aria-controls`,
  `aria-labelledby`, `aria-describedby`, `aria-owns`,
  `aria-activedescendant`, `aria-errormessage`, `aria-details`,
  `aria-flowto`, `href="#…"`, and `hx-target`, `hx-include`,
  `hx-indicator`, `hx-select`, `hx-disabled-elt` when the value is a plain
  `#id`. `External` lists ids a fragment may reference outside itself.

## 9. Behaviour in detail

Settled by Claude after the cross-model review of this draft
([review 0001](reviews/0001-interface-draft.md)). Each point makes a rule
above precise without changing its intent.

### 9.1 Kernel

- **Ids.** `Sequence(prefix)` panics unless `prefix` matches the id-name
  pattern. A token that is empty or not `[a-z0-9]+(-[a-z0-9]+)*` makes
  `Scope` return an error naming the source. `Get`, `Part` and `Key` on a
  zero `ID` panic. `WithIDs(ctx, nil)` panics. `Scope(a, a)` returns an
  error. A `Provider` or `ID` declared twice with equal arguments is two
  handles (K8), whatever the size of `T`.
- **Slot messages.** An error about a slot names it by its declaring
  position. `ErrStrayContent` quotes up to 40 bytes of the stray output;
  `ErrInBlock` names the wrapper (`Provide`, `Scope`, or the slot of a
  `Place`). The first error met in the collect pass is returned.
- **White space** is HTML's: space, tab, line feed, form feed, carriage
  return. A non-breaking space is content.
- **Fallbacks** render in the placement's context: `Value` works in the
  fallback of a valued slot. `Filled` or `Place` of a slot the current
  component did not list panics (`Filled`) or returns `ErrNotDeclared`
  (`Place`).
- `Slotted(nil, …)`, a nil slot, or a slot listed twice panics.

### 9.2 Forms: lifecycle and nesting

- `Validate` before `Bind` panics; so does `Set` after `Bind`. `Bind` keeps
  no initial value: an absent field is absent even if `Set` gave it one.
- `Bind`, `Validate` and `Submit` on a sub-form (a group or list item)
  panic. `Valid`, `Err` and `Errors` on a sub-form report its subtree.
- `Check` on a group runs during the root's `Validate` when the group is
  active and no node inside it failed; its error sits at the group's path.
  `Fail` on a group does the same. The root's checks run when no node of
  the whole form failed.
- **Conversion stages.** `Bind` runs the whole chain and records where it
  stopped. `Validate` runs each stage's rules on that stage's value in
  order, up to the stage that failed to convert; the first rule failure
  wins, otherwise the conversion error stands.
- **Multiple values** for a single-valued field: the last non-empty one is
  bound (the hidden-input-before-checkbox pattern works).
- **Trimming.** Every field parsed from text is trimmed before parsing,
  except a `Text` marked `Raw`. `Raw` on any other constructor panics at
  declaration.
- **Numbers.** `NaN` and infinities are conversion errors.
- **Booleans.** `Checkbox` is true when its last value is non-empty and not
  one `strconv.ParseBool` reads as false; it formats as `true` or `false`,
  and `Filled()` on it means ticked. `Bool` accepts what `ParseBool`
  accepts plus `on`; it formats as `true` or `false`.
- **Files.** A part with an empty file name is no file. `Files` ignores such
  parts.
- **Options** run in `Validate` as the first check of the field's first
  stage, against its trimmed text (each item for `Texts`).
  `ErrNotAnOption` is not replaced by `InvalidWith`. `Options` on an `Enum`
  replaces its choices. `Enum` with no values, or two values with one
  `Value()`, panics; an enum formats as `Value()`.
- **`Eq` and `In`** on a field whose `T` is a slice or a file panic at
  declaration. A condition built from a handle stays valid after `As`.
  `All()` holds and `Any()` does not. A zero `Cond` passed to `When`
  panics; `When` twice on one node panics.
- **Before Bind**, conditions resolve against initial values, so a page
  rendered from `Set` shows the right nodes as active.
- **Lists.** An index is `0` or `[1-9][0-9]{0,5}`, followed by `]` and then
  `.` or the end of the name. An index none of whose names reaches a
  declared member is dropped. `Bind` replaces items given by `Add`; `Add`
  after `Bind` panics. An inactive list has no items. With too many items,
  the first `MaxItems` are bound and validated. `MinItems(n<0)`,
  `MaxItems(n<1)`, or a minimum above the maximum panics.
- **Declaration checks.** `nil` passed to a builder, to `rule.New` or to
  `rule.Matches` panics. `RequiredWith`, `InvalidWith` and `Rule.Err` panic
  on an error that people would see as Unknown (one that implements no
  `i18n.Ter`, such as a `Def1` not given its argument). Repeated `Label`,
  `Placeholder`, `Help`, `InvalidWith`: the last wins.
- `As` also carries `RequiredWith` and `InvalidWith`. A `Conv` with a nil
  function panics.

### 9.3 Forms: requests and controls

- `FromRequest` reads a POST, PUT or PATCH body (urlencoded or multipart,
  through `ParseMultipartForm` with net/http's 32 MB default memory) without
  merging the query; a GET or HEAD reads the query. Any other content type
  is unreadable. net/http removes multipart temporary files after the
  handler returns. Limiting body size is the app's (`http.MaxBytesReader`).
- `Controls()` and `Control().Children()` yield direct children. `Text()`
  is empty for an absent unbound field, for files, groups and lists.
- `Texts` without options is `KindText`. `Kind.String()` is the name
  without its prefix, in lower case (`multiselect`).
- `Attrs`: when two rules give one attribute, the later rule's value wins;
  attributes come in the order they first appear.
- `FieldError.Error()` is the error's text alone at path `""`, whose
  `Label` is nil. formstest writes the form-level path as `<form>`.

### 9.4 Rules

- `Matches` checks the whole value: the expression is anchored, as HTML's
  `pattern` is, and its source is the `pattern` attribute.
- `URL` accepts absolute `http` and `https` URLs with a host.
- `Email` accepts what `net/mail.ParseAddress` accepts with no display
  name. `Phone` accepts E.164 after removing spaces, dashes, dots and
  parentheses. `CountryCode` is ISO 3166-1 alpha-2, `CurrencyCode` ISO
  4217, both upper case. `PostCode(country)` panics for a country it has no
  pattern for.
- `ContentType` sniffs the file's first 512 bytes with
  `http.DetectContentType` and compares media types without parameters;
  `image/*` matches a family. A file that can't be read fails the rule.
  Types in its message are joined with `, `.
- `NotBefore` and `NotAfter` on a `Date` field compare calendar dates in
  UTC; the attributes are dates for `Date` and `datetime-local` values for
  `Time`.
- File sizes in messages use 1024 per unit.

### 9.5 uitest and formstest

- `Golden` and `CheckIDs` parse a fragment in the context its first element
  needs (`tr` in `tbody`, `td` in `tr`, `option` in `select`), so table rows
  and options are not dropped.
- `CheckIDs` splits ID-list attributes (`headers`, `aria-labelledby`,
  `aria-describedby`, `aria-controls`, `aria-owns`, `aria-flowto`,
  `aria-details`) on white space; ignores `href="#"`; fails on an empty
  `id` or reference; checks each `<template>`'s content as its own
  document.
- `Context` options apply in order; one that installs `WithIDs` overrides
  `Sequence("t")`.
- `formstest.Submit` fails with `t.Fatalf`. `AssertLabelled` checks fields,
  named groups and lists, not the root or list items, and reads whether a
  label was set through an internal package of forms, not by comparing
  text.

## 10. Amendments

Each amendment has an ID, a date, the sections it changes and why, and an
entry in status.md.

| ID | Date | Sections | Change |
|---|---|---|---|
| A1 | 2026-11-21 | §5.9, §6.3 | Error texts follow errors' convention: lower case, no final period (`must be at most %d characters`); the UI capitalizes them. A clarification by Claude (F1.1), to be overturned by epistemer if unwanted |
| A2 | 2026-11-21 | §5.1, §5.8, §5.10, §7, §9.2 | Clarifications by Claude (F5 review): a list item has no `When` or `Label` of its own (both panic; conditions go on its fields or on the list, and it takes its list's label); `When` and `Label` on the root are legal and have no effect. A list index counts only when one of its names reaches a member an item declares, so junk never builds items or uses up `MaxItems`. `MinItems` above the default limit raises it. A template row's fields report their own conditions, not their list's, so a client can clone it while the list is hidden. A check that fails a field or the form and returns nil stops the remaining checks. `AssertLabelled` checks a list's members once, on its template row |
| A3 | 2026-11-21 | §1, §5.3, §5.10, §6.2, §8, §9.2, §9.3 | From the release review (Claude): a list index counts only when one of its names reaches a declared field, at any depth; `FromRequest` reads a body without parsing the query, so a query net/http would refuse doesn't spoil it; a Checkbox is unticked, present and `false` before Bind too; an unreadable `Submit` leaves every form and sub-form invalid; an Enum accepts any option whose `Value()` is offered; `Options` makes any field `KindSelect` (`KindMultiSelect` for several values) unless `Hint` set the kind; `When` on the root is accepted and ignored; `Min`, `Max`, `Between` panic on a NaN bound, length, size and item limits on a negative one; `ContentType("*/*")` matches any file; `Matches` gives `pattern` only when the expression avoids Go-only syntax; `URL` needs a host name; uitest keeps a whole page's `<html>` and `<body>`, fails an id with white space, and its `-update` is shared with the test package (§8); formstest's imports |
| A4 | 2026-11-22 | §8 | uitest gains `CheckNames` and `CheckSelectors` (with `SelectorOption`, `Prefix`), as [visual-interface.md](visual-interface.md) §13.1 states them. Locked by epistemer with the visual packages |
| A5 | 2026-11-22 | §4A (new); visual-interface.md §2.7, §5.1, §5.3, §5.4, §7.3, §7.4, §8.2, §8.4, §13.4 | An address is a link: the kernel gains `Link` and `URL`, and eight signatures of the visual packages take a `ui.Link` where they took a `templ.SafeURL` (ADR 0008). Locked by epistemer. Built by the roadmap's item A5, before v0.2.0 is tagged |
| A6 | 2026-11-22 | visual-interface.md §5.1, §7.3, §8.2, §8.3, §13.5; [server-state-interface.md](server-state-interface.md) | Actions and tables: `ui/actions`, `ui/tables` and their test kits; `Unavailable` on Button and MenuItem, `MenuSpec.ID`, the shells' action form and dialog. Locked by epistemer with the defaults of T1–T11 |
| A7 | 2026-11-22 | §5.3, §5.4, §5.10, §6.2 (forms); visual-interface.md §6, §7, §10, §13.6; server-state-interface.md §6; [rich-inputs-interface.md](rich-inputs-interface.md) | The remote select, uploads and list fields: `Field[T].Remote`, `Form.Upload`/`Uploads` over the new `ui/forms/files`, `Control.Remote`/`Stored`/`Lost`/`Bounds`, a new list's blank items, `rule.Files`/`MinFiles`/`MaxFiles`; §5.8's rule for items stays. Locked by epistemer with R1–R28 and L1–L7 (some answered, the rest on defaults) |

**The visual packages.** Their contract is
[visual-interface.md](visual-interface.md), locked by epistemer on
2026-11-22 (D6, the defaults of V1–V18). It covers theme (with the nested
module `themeschema`), icons, elements, inputs, components, layout and the
catalogue.
