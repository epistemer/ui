# ui: interface of the remote select, uploads and list fields

| | |
|---|---|
| Status | **Draft** of amendment A7, for epistemer to lock (roadmap D8.4). Its questions are R1–R28 of the design (§9 there) and L1–L6 below; each stands on its default until answered. Once locked, it amends [interface.md](interface.md) (forms), [visual-interface.md](visual-interface.md) (inputs, components, CSS, the script) and [server-state-interface.md](server-state-interface.md) (tables) |
| Date | 2026-11-22 |
| Roadmap item | D8.4 |
| Inputs | [rich-inputs-design.md](rich-inputs-design.md) (R1–R28, the spike's findings §8.1), ADRs [0011](adr/0011-remote-select.md)–[0013](adr/0013-list-fields.md), reviews [0019](reviews/0019-rich-inputs-design.md) and [0020](reviews/0020-d8-2-spike.md), the spike in `internal/spike`, handlers' ADR 0006 (a route's own body limit, drafted with this) |

The contract of what forms, inputs, components, tables, the CSS and the
ui script gain for the remote select, uploads and list fields. It
follows the design and the spike: a part the spike didn't try is marked
*(not spiked)*, as the visual lock did.

The conventions of visual-interface.md §2 hold: builders, variants as
data attributes, `Attrs` on every spec, programmer mistakes panic (at
declaration in forms and tables, at render in inputs), `Ter` for every
text, logical CSS.

## 1. Module

| Package | Gains imports | Spiked as |
|---|---|---|
| `ui/forms` | models' `listing` (standard library only) and `ui/forms/files` | `internal/spike` (`remote`, `upload`, by path) |
| `ui/forms/files` *(new, L1)* | the standard library | `internal/spike` (`storedFile`, `upload`) |
| `ui/forms/rule` | `ui/forms/files` | not spiked |
| `ui/inputs` | `ui/actions` | `internal/spike/spike.templ` |
| `ui/components` | nothing new | `internal/spike` (`suggestions`) |
| `ui/tables` | nothing new | a stand-in form in `<ui-table>` |

## 2. forms (amends interface.md)

### 2.1 Remote fields (§5.4, §5.10)

```go
// Remote makes the field a remote select (KindRemote): its value is a
// record's id, searched at search and named by l. forms never calls l.
func (f *Field[T]) Remote(search interface{ URL() string }, l listing.Labeler) *Field[T]

// Remote gives a remote field's search link and Labeler, and id: the
// value formatted, or "" when the field is absent or its text didn't
// convert. ok is false for any other field.
func (c Control) Remote() (search interface{ URL() string }, l listing.Labeler, id string, ok bool)
```

- Legal on a field whose kind is `KindText` or an integer `KindNumber`
  (`Text`, `Int`, `Number[T]` of an integer type, a `Parsed` id type).
  It panics on any other field, with a nil link or `Labeler`, combined
  with `Options` or `Hint` in either order, and after `Bind`. `Hint` with
  `KindRemote`, `KindUpload` or `KindUploads` panics.
- `Control.Attrs` of an integer remote field gives `inputmode="numeric"`
  (for the text input without script); a text one gives none.
- Binding and validating are the field's type's. A row found is not a
  row allowed: whether this user may pick the id is the app's `Check`.
- **A `Labeler` answers `false`, not an error, for an id that can't name
  a row** (`abc` for a uuid column). Every `Labeler` an app passes must;
  an error means the store failed.

### 2.2 Uploads (§5.3, §5.10) *(single-file replacement, `Required` and the counts not spiked)*

```go
package files // ui/forms/files

// Stored is a file the record already has.
type Stored struct {
	Key   string                   // the app's key, posted back to keep it
	Name  string                   // shown
	Href  interface{ URL() string } // where it opens; nil for none
	Image bool                     // Href is an image: shown as a thumbnail
}

// Upload is an upload field's value.
type Upload struct {
	Kept  []Stored                // stored files still kept, in the order given
	Added []*multipart.FileHeader // files chosen now
}
```

```go
package forms

func (f *Form) Upload(name string, stored []files.Stored) *Field[files.Upload]  // KindUpload: one file in all
func (f *Form) Uploads(name string, stored []files.Stored) *Field[files.Upload] // KindUploads

// Stored yields each stored file with its keep state: posted, after Bind;
// true before.
func (c Control) Stored() iter.Seq2[files.Stored, bool]

// Lost is true for a file field (File, Files, Upload, Uploads) that was
// posted files in a form that failed: the browser won't send them again.
func (c Control) Lost() bool
```

- **Presence** (§5.3's table gains a row): an upload field is present
  when `Kept` and `Added` hold at least one file between them. Before
  `Bind` its value is `Upload{Kept: stored}` (a copy), present when
  `stored` isn't empty.
- **Binding**: kept files post as `<path>.keep=<key>`; new files under
  `<path>`. A posted key not among `stored` is dropped; a repeated key
  counts once; `Kept` keeps `stored`'s order.
- `Upload` holds one file: with `len(stored) > 1` it panics at
  declaration; of several files posted it takes the first, as `File`
  does; an added file empties the value's `Kept`, while the control still
  reports the posted keep state.
- `Required` holds with one kept or added file.
- `Set`, `SetPtr`, `Options` and `Hint` on an upload field panic; the
  value comes from `stored` and the request only. `Options` on `File` or
  `Files` panics too.
- `Upload` or `Uploads` declared in a list's item panics (R26); `File`
  and `Files` are legal there.
- `stored` with an empty or repeated `Key`, or an empty `Name`, panics.
  `Image` with a nil `Href` draws no thumbnail.
- `Control.Text()` of an upload field is nil. `Control.Stored()` of any
  other control, or of one with no stored files, is an empty iterator,
  never nil.
- **`Lost()`** is evaluated when called: true for an active file field
  (`KindFile`, `KindFiles`, `KindUpload`, `KindUploads`) that was posted
  at least one file, has no error of its own, and whose form has an
  error (`Form.Err() != nil`, so a `Fail` after `Bind` counts).
- **An oversized body**: `Submit` reports it as `ErrUnreadable` wrapping
  the `*http.MaxBytesError`. A handler returns that error as is rather
  than re-rendering, so handlers' `Status` answers 413 and the script
  keeps the person's files (design §4.5):

  ```go
  if !f.Submit(ctx, r) {
  	var big *http.MaxBytesError
  	if errors.As(f.Err(), &big) {
  		return f.Err() // 413
  	}
  	// … re-render with 422
  }
  ```

### 2.3 Lists (§5.8, §5.10)

```go
// Bounds gives a list's effective bounds: max is DefaultMaxItems when
// MaxItems wasn't set. Zero for any other control.
func (c Control) Bounds() (min, max int)
```

- **A new form's list** (R27): `Control().Children()` of a list that was
  neither bound nor given items by `Add` yields `MinItems` blank items,
  built at indices `0..MinItems-1` as the template row is. The list
  itself isn't changed: `Items()` stays empty and `Add` appends as
  before. *(not spiked)*
- **An item that posts nothing** (L7): an item whose fields are all
  unchecked checkboxes posts no name, and binding would drop it. Each
  item renders `<input type="hidden" name="<path>[i]">` (no member), and
  forms counts an index posted bare like that as an item. A change to
  the locked rule of §5.8, which counts an index only through a member's
  name. *(not spiked)*

### 2.4 Rules (§6.2) *(not spiked)*

```go
func Files(rules ...Rule[*multipart.FileHeader]) Rule[files.Upload] // each added file
func MinFiles(n int) Rule[files.Upload]                             // kept plus added
func MaxFiles(n int) Rule[files.Upload]
```

- `MaxSize` adds `data-ui-max-size` (bytes) to `Control.Attrs`, as
  `ContentType` adds `accept`; `Files` passes its inner rules'
  attributes on.
- Errors, declared in `rule` with the others: `ErrMinFiles`
  (`errors.forms.rule.min_files`, "choose at least %d") and
  `ErrMaxFiles` (`errors.forms.rule.max_files`, "choose at most %d").
  forms' own `ErrMinItems` and `ErrMaxItems` stay the list's.

### 2.5 Kinds

`KindRemote`, `KindUpload` and `KindUploads` join `Kind`, after
`KindList`, so existing values keep their numbers.

## 3. inputs (amends visual-interface.md §6)

```go
// WithLabelCache returns ctx with a cache of LabelOf answers by Labeler
// and id, for the request (R10). Without one, each field calls once. A
// Labeler whose dynamic type isn't comparable (a func adapter) is never
// cached.
func WithLabelCache(ctx context.Context) context.Context

// Previews shows thumbnails of chosen images (KindUpload, KindUploads,
// KindFile, KindFiles), marking the field data-ui-previews; the app's CSP
// must allow img-src blob: (R17). On any other kind it panics.
func (f FieldSpec) Previews() FieldSpec
```

Rendering a `KindRemote` field, or a stored file with an `Href`, asks
`actions.Allowed`; with no `actions.Gate` in the context it panics (A6's
S9).

### 3.1 What each new kind renders

| Kind | Element |
|---|---|
| `KindRemote` | `<ui-combobox class="ui-combobox" data-ui-search="…" data-ui-texts="…">` holding the text input of the id (`name` = path, `Control.Attrs` applied) and `<span class="ui-combobox-chosen" id="…-chosen">` with the chosen name, named by the input's `aria-describedby`. Text that didn't convert: the input holds it, the span is empty. A refused search link: no `data-ui-search` (R7) |
| `KindUpload`, `KindUploads`, `KindFile`, `KindFiles` | `<div class="ui-field ui-upload" data-ui-texts="…">`: label, help, the lost note (`<p class="ui-field-help ui-upload-lost">`, when `Lost`, named by the input's `aria-describedby`, not an error), for an upload field `<ul class="ui-upload-files">` of `<li class="ui-upload-file">` each holding a thumbnail (`Image` with an admitted `Href`), the checkbox `name="<path>.keep"` checked by its keep state, `<label>` around "Keep" and the name as text only, and the name as a link beside it when the gate admits `Href`; then the file input (`ui-upload-input`, `multiple` for `KindUploads` and `KindFiles`, `Control.Attrs` applied), `<ul class="ui-upload-chosen">`, `<progress class="ui-upload-progress" max="100" hidden>`, a visually hidden status, the error. `File` and `Files` change from visual-interface §6.1's plain input *(not spiked for them)* |
| `KindList` | `<fieldset class="ui-list">` with its legend and help; `<ui-repeat data-ui-path data-ui-next data-ui-min data-ui-max data-ui-texts>` (bounds from `Control.Bounds`) holding `<ol class="ui-list-items">` of `<li class="ui-list-item">`, each the item marker of §2.3 and a `<fieldset class="ui-group">` with the legend "Item n", its fields and `<button type="button" class="ui-button ui-list-remove" data-ui-remove hidden>`; a `<template>` of the template row in the same markup; `<button type="button" class="ui-button ui-list-add" data-ui-add hidden>`; a visually hidden status; then the list's error. `inputs.Field` no longer returns an error for a list |

- `data-ui-texts` is a JSON object of the element's texts, translated
  (ADR 0005: the script holds no catalogue). What the server knows it
  fills in: the field's or list's label (`clear`, `add`) and the maximum
  size (`tooLarge`). What only the script knows stays a verb, one per
  text: a count or a file name. The script fills it in a single pass, so
  a `%s` inside a file name stays as it is. Keys per element are §7's
  second column.
- Ids are generated per field as today; the template row's are its own,
  and the script suffixes a clone's.

## 4. components (amends visual-interface.md §7)

```go
// Suggestion is one answer of a search: an id and its name (R5).
type Suggestion struct {
	Value string
	Label i18n.Ter
}

// SuggestionLimit is how many suggestions an answer shows. A handler
// fetches one more, so Suggestions can tell there were more.
const SuggestionLimit = 20

// Suggestions renders a search's answer: <ul data-ui-suggestions> of
// <li data-value> with the label as text, the first SuggestionLimit, and
// data-ui-more when there were more. No ids, no roles: the script builds
// the options.
func Suggestions(s ...Suggestion) templ.Component
```

## 5. tables (amends server-state-interface.md §6) *(chips and a real table not spiked)*

```go
// Remote makes the column's filter a remote select with listing.Eq; the
// chip names the value through l (R13).
func (c Column[R]) Remote(search ui.Link, l listing.Labeler) Column[R]
```

- Legal on a `Text` or `Integer` column with `Filter()`; otherwise the
  table's `Check` reports it, as for other column options. A nil link or
  `Labeler` panics at once.
- Rendering the filter needs an `actions.Gate`, as any remote field
  does. The chip and the filter field each ask `LabelOf` once; with
  `inputs.WithLabelCache`, the field's answer is cached (the chip is
  computed by `tables`, which doesn't import `inputs`).
- The filter field is `Text` or `Number[int64]` with `Remote`; its
  parameter is owned by the table like any filter's.
- The chip asks for a valid state only: a miss shows the id, an error
  fails the request.

## 6. The script (amends visual-interface.md §10)

| Element or behaviour | Does |
|---|---|
| `<ui-combobox>` | Upgrades the shape of §3.1 when it has `data-ui-search`: the hidden input takes the name; the visible one becomes `role="combobox"` (`aria-autocomplete="list"`, `aria-expanded`, `aria-controls`), with a clear button (`tabindex="-1"`), a manual popover holding the listbox and the visible status, and a visually hidden live region outside the popover. On upgrade it removes `inputmode` and the chosen span, and that span's id from `aria-describedby`; unconverted text stays in the visible input. Fetches the search link with `q` set through `URLSearchParams` (keeping the link's own query), itself, no htmx, debounced 250 ms, cancelling the last; accepts only a 200, not redirected, whose root is `ul[data-ui-suggestions]`; builds each option from `data-value` and text. Keyboard and events as design §3.4 |
| `<ui-repeat>` | Shows its own Add and Remove buttons; clones the template, rewriting its own path prefix in `name`, `data-ui-path` and `data-ui-when` at every depth, suffixing this level's ids and every reference to them; renumbers legends and Remove buttons; evaluates conditions before focusing the first visible, enabled control; announces; marks Add and Remove unavailable at the bounds with the reason (`aria-disabled`) |
| Uploads | Keep cards over the checkboxes; the chosen list with Remove (through `DataTransfer`); size and type checks with `setCustomValidity` and a capturing `submit` refusal (forms are `novalidate`); the drop highlight; the type check defers to the server for a file the system gives no type; progress from the request's own `xhr.upload` (`htmx:beforeSend`); a 413 with files chosen puts `body_too_large` on the upload fields and keeps the form (a send error is left to htmx); choosing files again clears it |

Announcements go through a timer, never an animation frame.

## 7. Keys

| Key | English | Where (`data-ui-texts` key) |
|---|---|---|
| `ui.inputs.combobox.clear` | Clear %s | `clear`, label filled |
| `ui.inputs.combobox.results` | Results: %d | `results` |
| `ui.inputs.combobox.none` | No matches | `none` |
| `ui.inputs.combobox.failed` | Couldn't search. Try again. | `failed` |
| `ui.inputs.combobox.more` | Keep typing to narrow these down | `more` |
| `ui.inputs.upload.keep` | Keep %s | server, in the label |
| `ui.inputs.upload.remove` | Remove %s | `remove` |
| `ui.inputs.upload.undo` | Undo | `undo` |
| `ui.inputs.upload.chosen` | %s chosen | `chosen` |
| `ui.inputs.upload.again` | Choose the files again | server |
| `ui.inputs.upload.too_large` | %s is larger than %s | `tooLarge`, size filled |
| `ui.inputs.upload.wrong_type` | %s isn't a type this field takes | `wrongType` |
| `ui.inputs.upload.body_too_large` | These files are too large to send together. Send fewer at a time. | `bodyTooLarge` |
| `ui.inputs.list.item` | Item %d | server, and `item` |
| `ui.inputs.list.add` | Add to %s | server, label filled |
| `ui.inputs.list.remove` | Remove item %d | server, and `remove` |
| `ui.inputs.list.added` | Item %d added | `added` |
| `ui.inputs.list.removed` | Item %d removed | `removed` |
| `ui.inputs.list.at_most` | Maximum: %d | `atMost` |
| `ui.inputs.list.at_least` | Minimum: %d | `atLeast` |

## 8. Questions of the lock

R1–R28 are the design's (§9 there). These are the contract's own:

| # | Question | Default | Alternative |
|---|---|---|---|
| L1 | Where `Stored` and `Upload` live | A leaf package `ui/forms/files` (`files.Stored`, `files.Upload`), because `rule` can't import forms, which imports it | Upload rules as methods on a field type of its own (`f.Upload(…).MaxFiles(3)`), keeping the types in forms |
| L2 | `Control.Remote`'s shape | One method giving link, `Labeler`, id and ok | Three methods (`Search()`, `Labeler()`, `ID()`) |
| L3 | `Column.Remote` without `Filter()` | `Check` reports it | `Remote` implies `Filter()` |
| L4 | The script-off upload's keep control | A checked checkbox per stored file, `name="<path>.keep"` | A hidden input per kept file, with a remove checkbox |
| L5 | What `Suggestions` takes | `...Suggestion` | `[]Suggestion` and a `More(bool)` builder, so a handler that pages can say so without a 21st row |
| L6 | The parameter the combobox sends | `q`, fixed | A `data-ui-param` per field |
| L7 | An item that posts no name (only unchecked checkboxes) | A bare marker per item, `<path>[i]`, which forms counts (a change to locked §5.8) | No marker; such an item is dropped, and an item must hold a field that always posts |

## 9. Not spiked

Marked above, collected: a single-file `Upload` replacing its stored
file; `Required`, `MinFiles` and `MaxFiles` on uploads; the lost note on
`File` and `Files`; a text (uuid) id; `MinItems` blank items in a new
list; the item marker; `Column.Remote` and its chip in a real table; goldens (the build
adds them); a screen-reader pass; the upgrade's flash, measured in a
visible pane.
