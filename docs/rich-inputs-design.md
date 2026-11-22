# The remote select, uploads and list fields: design draft

| | |
|---|---|
| Status | Draft, amended by the D8.2 spike's findings (§8.1). Every question in §9 (R1–R28) has a default and stands on it until epistemer answers or the lock (D8.4) takes the defaults |
| Date | 2026-11-22 |
| Roadmap item | D8.1 |
| Review | [0019](reviews/0019-rich-inputs-design.md) |
| Inputs | [ui-inventory.md](ui-inventory.md), [forms-design.md](forms-design.md) §8.3 and §10.3, [interface.md](interface.md) §5.3, §5.8 and §5.10 (locked), [visual-interface.md](visual-interface.md) §6 and §10.6 (locked), [server-state-design.md](server-state-design.md) §6.5 and §13, [server-state-interface.md](server-state-interface.md) (locked, A6), ADR [0005](adr/0005-client-behaviour.md), models' `listing.Labeler` and `option.SelectOption`, handlers' `WithMaxBody`, htmx 2.0.11 |

This is the second round of D7 (S6): **the remote select, uploads and list
fields**, and with the first of them **a table filter on a link column**.
ADRs [0011](adr/0011-remote-select.md), [0012](adr/0012-uploads.md) and
[0013](adr/0013-list-fields.md) record the three choices that shape the
rest. A slice spike (D8.2, §8) tries every part, and every pair that meets
on a page, before any text is locked.

Examples use the toy shop: an order form with a customer (a remote select
over `customer_list_view`), a receipt (an upload), and lines (a list whose
rows each pick a product, another remote select).

## 1. What the record of zp asks for

Read from zp on 2026-11-22, as needs. Counts and shapes only.

**The remote select** (18 search handlers, about 10 form call sites, and
every link-column filter of the server tables):

| Kept | Why |
|---|---|
| A field holds a record's id and shows its name | The id is what the handler needs; the name is what people read |
| Suggestions come from the server as people type, a page at a time | The lists are too long to preload |
| The chosen name is given to the widget, not looked up by the client | The first paint is right, with no request |
| The same widget filters a table by a link column | One widget for "pick a record" |

| Removed by design | What it let through |
|---|---|
| The search endpoint and its parameter name were strings written at each call site | Nothing checked that the route existed |
| The chosen name was passed beside the field, as a separate pointer | A form re-rendered after an error lost it unless every page remembered |
| A search error rendered an empty list, the same as "no matches" | People retyped their search against a database that was down |
| Suggestions were clickable list items with no roles, no keyboard and no active option | The widget was mouse-only |
| Infinite scroll loaded the next page when its last item came into view | No end, nothing announced, and a page of ten |
| A form to add a new record sat in a modal inside the dropdown | A form inside a form's control; its own errors had nowhere to go |

**Uploads** (11 forms, single and multiple files, mostly images):

| Kept | Why |
|---|---|
| A drop zone and a file button | People drag files from their desktop |
| A preview of what was chosen, with its name and size, and a way to take one out | People check before they send |
| An edit page shows the files already stored | Otherwise nobody knows what saving does |
| Storage behind one interface for local disk and object storage, with thumbnails | A storage concern, not a form's (out of ui's scope) |

| Removed by design | What it let through |
|---|---|
| On an edit page, the script downloaded every stored file and put it back into the file input | Every save uploaded every file again; a large file made a small edit slow |
| Previews were `blob:` image addresses with no `img-src` rule for them | Strict CSP was impossible |
| The form had no notion of "keep this file" or "remove that one" | Removing a file was a separate page, or not possible |

**List fields** (no repeatable rows in markup at all):

| Kept | Why |
|---|---|
| Nested shapes: a record with a list of sub-records, each with typed fields | The domain has them |

| Removed by design | What it let through |
|---|---|
| A list travelled as JSON in one hidden input, built by two scripts | No field errors per row, nothing without script, and the server parsed a second format |
| The client guessed numbers: a value that parsed as a number was sent as one | A code like `007` arrived as `7` |

forms already removed the last two: a list is a repeatable group posting
`list[i].field` (forms-design §8), with a template row for a client to
clone (§8.3). D8 renders it.

## 2. Principles

These add to [visual-design.md](visual-design.md) §2 and
[server-state-design.md](server-state-design.md) §2.

1. **A field is declared once** (forms' I1), including what it needs to be
   drawn. `inputs.Field` renders groups, list rows and a table's filter
   form generically (`components/table.templ` renders each filter control
   with it), so a widget that needs an argument the generic path can't
   pass is a widget nobody can put in a list. This reverses two bullets of
   forms-design §10.3; R1 and R14 put it to epistemer.
2. **ui stores nothing.** No files, no temporary uploads, no flash. A file
   goes to the handler with its form; where it is kept is the app's.
3. **Every request a widget makes goes to a link the gate admits.** A
   search link is a route like any other.
4. **Works without script where the platform allows it, and says what it
   lacks where it doesn't.** A file input and a form are native. Searching
   as you type and adding a row are not; their markup without script is
   specified (R3, R20).
5. **The ui script does not depend on htmx** (ADR 0005). It may use htmx
   when it is there, but nothing D8 adds breaks without it.
6. **A clone is a new element.** A row cloned from the template has fresh
   ids, its own index in every name and condition, and behaviours that set
   themselves up.

## 3. The remote select (ADR 0011)

### 3.1 Declaring

A remote select is a field whose value is a record's id. The field says
so, with the link that searches and the `Labeler` that names a chosen id
(R1, R2):

```go
type Order struct {
	Customer *forms.Field[int64]
	// …
}

f.Customer = f.Number[int64]("customer_id").Label(models.L10nOrder_CustomerId).
	Required().
	Remote(adminroutes.CustomerSearch(), customers) // a ui.Link, a listing.Labeler
```

- `Remote(search interface{ URL() string }, l listing.Labeler)` on
  `Field[T]` gives the field `KindRemote`. forms keeps importing no kernel
  and no templ: the link is the literal interface `ui.Link` aliases.
  `listing` imports nothing but the standard library (checked).
- `Remote` takes a field whose kind is `KindText` or an integer
  `KindNumber` (`Text`, `Int`, `Number[int64]`, a `Parsed` id type). On any
  other field, with a nil link or `Labeler`, or combined with `Options` or
  `Hint` in either order, it panics at declaration (I10); so does `Hint`
  with any of the new kinds. One value only (R11).
- `Control.Remote() (search interface{ URL() string }, l listing.Labeler, id string, ok bool)`
  hands both to a renderer, with `id`: the field's value formatted, or
  `""` when the field is absent or its text didn't convert.
- **forms never calls the `Labeler`** (R2). Binding and validating a remote
  field is binding and validating its type: an `int64` that doesn't parse
  is the field's usual error. Whether the id names a row this user may
  pick is the app's check (`f.Check`), as it is for any id today: a row
  found is not a row allowed (R9).

### 3.2 What it renders

`inputs.Field` on a `KindRemote` control renders, inside the usual
`ui-field` wrapper (label, help, error, ids and `aria-describedby` as
visual-interface §6.2):

The server renders the shape that works without script (R3):

```html
<ui-combobox class="ui-combobox" data-ui-search="/admin/customers/search"
             data-ui-texts='{"results":"%d results","none":"No matches",…}'>
  <input id="field--…" class="ui-input" type="text" name="customer_id" value="7"
         inputmode="numeric" aria-describedby="field--…-chosen …">
  <span class="ui-combobox-chosen" id="field--…-chosen">Ann Lee</span>
</ui-combobox>
```

Someone without script types an id they know, and sees the current
choice. On `connectedCallback` the script upgrades it to:

```html
<ui-combobox class="ui-combobox" data-ui-search="…" data-ui-texts="…">
  <input type="hidden" name="customer_id" value="7" data-ui-value>
  <input id="field--…" class="ui-input" role="combobox" type="text"
         value="Ann Lee" autocomplete="off" spellcheck="false"
         aria-autocomplete="list" aria-expanded="false"
         aria-controls="field--…-options" aria-describedby="…">
  <button type="button" class="ui-combobox-clear" tabindex="-1" aria-label="Clear Customer">…X icon…</button>
  <div class="ui-combobox-popup" popover="manual" id="field--…-popup">
    <ul class="ui-combobox-options" role="listbox" id="field--…-options" aria-label="Customer"></ul>
    <p class="ui-combobox-status" role="status"></p>
  </div>
</ui-combobox>
```

- The name moves to the hidden input; the visible input keeps the field's
  id, so the label and `aria-describedby` still point at it, and its value
  becomes the chosen name. The upgrade can flash the id for a frame before
  the module runs; the spike measures it, and CSS on
  `ui-combobox:not(:defined)` is the fallback.
- `inputmode="numeric"` only for an integer field; a text id (a uuid)
  takes none.
- Every text the script shows or announces is rendered by the server into
  `data-ui-texts`, translated (ADR 0005: the script holds no catalogue).
- The clear button is out of the Tab order (`tabindex="-1"`): the keyboard
  clears by emptying the text (§3.4). It shows only while there's a value.

**The chosen name.** At render, `inputs` calls `LabelOf(ctx, id)` once
when `Control.Remote` gives an id, and shows the answer as an
`i18n.Static` text. Text that didn't convert (`abc` for an `int64`) is
never looked up: it shows as typed, with the field's conversion error.
Otherwise:
- found: its name;
- not found (a row since deleted, or a tampered id): the id itself, so the
  field is never blank while it posts a value (R8);
- an error: the render fails with it, as any render error does. The
  database is down; a page that pretends otherwise misleads (R8).

A `Labeler` answers `false`, not an error, for an id that can't name a
row (`abc` for a uuid column): the design asks that of every app's
`Labeler`, and the contract says so beside `Remote`.

One call per distinct id per request: `inputs` memoizes `LabelOf` by
`Labeler` and id in the request's context, through a provider the form
render sets (R10). A list of 50 lines picking 12 products makes 12 calls;
50 different products still make 50. A batch method waits for a
measurement and would be a models amendment.

**The search link and the gate.** `inputs` asks `actions.Allowed(ctx,
search)` at render. A refused link renders the shape without script,
with no `data-ui-search`, so the script leaves it as it is (R7): an id can
still be typed, but nothing is searched. The gate on the search
link governs searching only. What a form may post, and whether this user
may pick that record, is the form handler's check (R9); markup can't make
a field read-only against a crafted request.

### 3.3 Searching

The search is a GET of the search link with one parameter, `q`, the text
typed, added to the link's own query. It belongs to the search link, so it
never meets a table's reserved `q`: the combobox's visible input has no
`name`, and the request is the script's own (R4).

**The script fetches; htmx doesn't** (R4). `<ui-combobox>` calls
`fetch` itself, debounced (250 ms), cancelling the request it supersedes
(`AbortController`), with `Accept: text/html` and `X-Requested-With:
ui-combobox`. Reasons, each found in this repository:
- the actions script treats any htmx 422 as a swap, tracks an opener for
  any request outside an open dialog, and drops a pending refocus on any
  other htmx request (`theme/js/actions.js`); a search must touch none of
  that;
- htmx 2.0.11 has no mutation observer, so `hx-*` on a row `<ui-repeat>`
  clones needs `htmx.process`, which a script that doesn't depend on htmx
  can't count on;
- a search answer is a list of options, not a region swap.

This changes ADR 0005's list of what emits `hx-*`: the remote select
leaves it (§6).

**The answer** is HTML, rendered by the handler with
`components.Suggestions`:

```go
func (h *H) CustomerSearch(w http.ResponseWriter, r *http.Request) error {
	text := r.URL.Query().Get("q")
	rows, _, err := h.q.ListCustomerListView(r.Context(), listing.Query{
		Search: text, SearchIn: []string{"name", "email"},
		Sort:   []listing.Sort{{Column: "name"}, {Column: "id"}},
		Limit:  components.SuggestionLimit + 1,
	})
	if err != nil {
		return err
	}
	opts := make([]components.Suggestion, 0, len(rows))
	for _, c := range rows {
		opts = append(opts, components.Suggestion{Value: strconv.FormatInt(c.ID, 10), Label: i18n.Static(c.Name)})
	}
	return components.Suggestions(opts...).Render(r.Context(), w)
}
```

- A suggestion is ui's `components.Suggestion{Value string; Label
  i18n.Ter}` (R5). Not models' `option.SelectOption`, whose `Label()` is
  an `i18n.TKer`, a translation key: a row's name is data and has none
  (found by the spike, §8.1).
  ui adds no type and no request parser: `q` is one line to read.
- `Suggestions` renders `<ul data-ui-suggestions [data-ui-more]>` holding
  `<li data-value="…">label</li>` per option, at most `SuggestionLimit`
  (20); given more, it renders 20 and sets `data-ui-more` (R6). No paging
  and no infinite scroll.
- **The script never inserts the answer's markup.** It parses it in an
  inert `<template>`, reads each item's `data-value` and text, and builds
  its own `<li role="option" id="<input id>-opt-<n>">` with `textContent`.
  So option ids are unique per combobox (two comboboxes on a page never
  share one), and nothing in an answer can add elements, attributes or
  script to the page.
- **What counts as an answer**: status 200, not redirected
  (`response.redirected` is false: a session that expired redirects to a
  sign-in page), and a root `ul[data-ui-suggestions]`. Anything else, or a
  network error, says "Couldn't search. Try again." in the status and
  keeps the previous choice.
- Given none, the status says "No matches". `data-ui-more` adds "Keep
  typing to narrow these down" to the status, outside the listbox, which
  holds options only.
- The search is a GET with no side effects; it carries the page's cookies
  (`credentials: "same-origin"`) and needs no CSRF token.
- An empty `q` is a search like any other; most handlers answer their
  first page. The popup opens on typing, on Alt+Down, or on Down.

### 3.4 Keyboard and focus

The APG editable combobox with list autocomplete, focus staying in the
input, the active option by `aria-activedescendant`:

| Key | Does |
|---|---|
| typing | marks the text uncommitted, searches, opens the popup when the answer arrives |
| Down, Up | open the popup; move the active option (stops at the ends) |
| Alt+Down, Alt+Up | open; close |
| Enter | with an active option: chooses it (hidden value, visible text) and closes. With uncommitted text and no active option, including while a search is pending: does nothing, and never submits. With committed text and the popup closed: the form's own Enter |
| Esc | with the popup open: closes it. With it closed and uncommitted text: restores the last choice's text. Either way the key stops there (`preventDefault`, `stopPropagation`). With the popup closed and nothing uncommitted, Esc passes on, so it closes an enclosing dialog |
| Tab, a click outside | closes; empty text clears the choice; other text that isn't a choice is replaced by the last choice (no free text) |
| Clear button (pointer only) | empties both, focuses the input |

A pointer chooses by clicking an option; the popup prevents the default
of `pointerdown`, so the input never loses focus to the click.

**Events.** The committed value lives on the hidden input. Whenever it
changes (a choice, a clear, a restore), the script dispatches bubbling
`input` and `change` events on it, so the condition evaluator (forms' I7)
and a table's filter form hear the field as they hear any other. The
visible input's own `input` and `change` events stop at `<ui-combobox>`:
its text isn't the field's value. The status line counts the
options shown ("12 results") once each answer settles, politely. The popup
is a manual popover, positioned under the input by anchor positioning
where it exists, and otherwise by the script, as `<ui-menu>`'s is
(ADR 0005); inside the modal action dialog it opens above the dialog,
since a popover shown later is higher in the top layer (spiked, §8).

A clone (§5) sets itself up in `connectedCallback`, so a remote select in
a list row works with no extra wiring.

### 3.5 A table filter on a link column

```go
tables.Col(c.CustomerId).Filter().Remote(adminroutes.CustomerSearch(), customers)
```

- `Remote` on `tables.Column` makes the column's filter a remote select
  with `listing.Eq` (R13). It takes a `Text` or `Integer` column; any other
  kind panics at the table's first use, as `Check` reports today.
- The filter form's field is `Text` or `Number[int64]` with `Remote`, so it renders
  through `inputs.Field` like every other filter.
- The chip shows the chosen name through the same `LabelOf` call, once per
  link filter per request, as `listing.Labeler`'s doc says. A miss shows
  the id; the filter still applies (no rows, most likely). An error fails
  the request (500), as a listing error does.
- The chip is shown only for a valid state (as today), so a value that
  didn't convert is never looked up.
- The search link's gate decides as in §3.2: refused, the filter is a
  text input of the id.

## 4. Uploads (ADR 0012)

### 4.1 Scope

**The file travels with its form** (R14), as `multipart/form-data`, to the
form's own handler. ui adds no upload endpoint, no temporary storage and
no direct-to-storage upload: those need a storage package neozp doesn't
have, and come with it.

### 4.2 Declaring, with the files already stored

forms' `File` and `Files` stay as they are for a form that only adds
files. An edit form, which shows the files it has and lets people keep or
remove them, uses a new field type (R15, R16):

```go
type StoredFile struct {
	Key   string                  // the app's key; posted back to keep it
	Name  string                  // shown
	Href  interface{ URL() string } // where it opens or downloads; nil for none
	Image bool                    // Href is an image: shown as a thumbnail
}

type Upload struct {
	Kept  []StoredFile             // stored files still kept
	Added []*multipart.FileHeader  // files chosen now
}

func (f *Form) Upload(name string, stored []StoredFile) *Field[Upload]  // one file in all
func (f *Form) Uploads(name string, stored []StoredFile) *Field[Upload] // several
```

```go
// stored is what the record has: nil on a create form.
f.Receipt = f.Upload("receipt", stored).Label(…).Required().
	Is(rule.Files(rule.MaxSize(5<<20), rule.ContentType("image/png", "image/jpeg", "application/pdf")))
```

- **The stored files are an argument of the declaration**, not a `Set`
  before `Bind` (an earlier draft): a handler that declares the form for a
  POST can't forget them, and forgetting them would have read as "remove
  every file". A form is declared per request, so the record is at hand.
  Before `Bind`, every stored file is kept.
- **Binding.** Kept files post as `receipt.keep=<key>`. A posted key that
  isn't one of `stored` is dropped, and a repeated key counts once: a form
  can keep only what it was shown. `Kept` keeps `stored`'s order. New files
  post under `receipt`.
- **`Upload` holds one file in all.** When a file is added, the value's
  `Kept` is empty, whatever was posted: an added file replaces the stored
  one, which is how "replace" reads. The control still reports each stored
  file's posted keep state, so a form that fails shows the checkbox as the
  person left it, and the next submit, without the lost file (§4.4), keeps
  the stored one.
- **Required** means one kept or added file.
- **Rules**, typed for `Upload` (I4): `rule.Files(rules
  ...rule.Rule[*multipart.FileHeader])` runs the file rules forms already
  has on each added file; `rule.MinFiles(n)` and `rule.MaxFiles(n)` count
  kept plus added. `MaxSize` gains the attribute `data-ui-max-size` in
  `Control.Attrs`, as `ContentType` gives `accept`.
- **Not in a list** (R26): a list's items are built during `Bind` from the
  posted indices, so an item has no record to take stored files from.
  `Upload` or `Uploads` declared inside a `List` panics (I10). `File` and
  `Files` work in a list.
- **What the handler does**: delete the stored files not in `Kept`, store
  `Added`. ui never touches either.
- `StoredFile.Href` is the edit page's link to the file: forms holds it so
  `inputs.Field` can draw the field in any group or list (principle 1).

### 4.3 What it renders

```html
<div class="ui-field ui-upload" data-ui-texts='{"remove":"Remove %s","undo":"Undo",…}'>
  <label class="ui-field-label" for="field--…">Receipt</label>
  <p class="ui-field-help">PNG, JPEG or PDF, up to 5 MB</p>
  <ul class="ui-upload-files">
    <li class="ui-upload-file">
      <input type="checkbox" name="receipt.keep" value="k-81" checked id="field--…-keep-0">
      <label for="field--…-keep-0">Keep <a href="/files/k-81">receipt.pdf</a></label>
    </li>
  </ul>
  <input type="file" id="field--…" name="receipt" class="ui-input ui-upload-input" accept="image/png,image/jpeg,application/pdf" data-ui-max-size="5242880">
  <ul class="ui-upload-chosen"></ul>
  <progress class="ui-upload-progress" max="100" hidden></progress>
  <p class="ui-visually-hidden" role="status"></p>
  <p class="ui-field-error">…</p>
</div>
```

- **Keep or remove, with no script**: a "Keep" checkbox per stored file,
  checked as the control reports it (all of them before a submit). The script draws it as a file card with a Remove button that
  unchecks it and an Undo that checks it again; the checkbox is still what
  posts.
- **Drop zone** (R18): the file input itself, sized and styled as the
  zone. Browsers accept a drop on a file input natively; the script only
  adds `data-ui-dragging` for the highlight.
- **What was chosen**: the script lists each chosen file's name and size
  in `ui-upload-chosen`, with a Remove button that rebuilds the input's
  files through `DataTransfer`. Image thumbnails of chosen files need
  `img-src blob:` in the app's CSP, so they are opt-in:
  `inputs.Field(c).Previews()` (R17). Stored files' thumbnails use `Href`
  and need nothing. Each chosen file listed is announced in the field's
  status.
- **A stored file's link is asked of the gate** (principle 3): `inputs`
  asks `actions.Allowed(ctx, Href)`, and a refused or nil link shows the
  name as text, with no link and no thumbnail.
- **Size and type, before sending** (R22): the script checks
  `data-ui-max-size` and `accept` on change and shows ui's own message
  (`ui.inputs.upload.too_large`, `ui.inputs.upload.wrong_type`, from
  `data-ui-texts`) in the field's error, and sets `setCustomValidity`.
  ui's forms are `novalidate`, so that alone blocks nothing: a capturing
  `submit` listener refuses a form holding such a field, before htmx or
  the browser sends it, and focuses the field (found in review 0020). The server still checks: the script only saves a
  long upload that would fail.
- **Progress** (R19): with htmx, the script listens to `htmx:xhr:progress`
  on the form and fills the `<progress>` of each upload field that has
  files chosen, and no other. htmx sends the download's progress under
  the same name, so the script listens to the request's own
  `xhr.upload`, which `htmx:beforeSend` hands over before the body goes
  (spiked). The bar hides when the answer arrives, and
  nothing announces "uploaded": an upload done isn't a form saved, and the
  answer (a 422 or a reply) speaks for itself. The progress is never
  focused. Without htmx, the browser's own progress shows.

### 4.4 A form that fails

A browser never refills a file input from the server, so a form answered
with errors loses the files chosen in it (R20). A field (`File`, `Files`,
`Upload`, `Uploads`) that was posted files, in a form that failed, renders
a note, "Choose the files again", as help-styled text above the input,
named by its `aria-describedby`. It isn't an error: the field may be
valid, and it doesn't set `aria-invalid`. forms tells the renderer through
a new `Control.Lost() bool` (§6).

`hx-preserve`, which htmx 2.0.11 has, is not the answer: it matches by id,
and a row cloned in the browser has an id the server's re-render won't
give it (forms renumbers items on bind); a preserved input keeps its old
`aria-invalid` and `aria-describedby`; and the chosen-files list beside
it is re-rendered empty. The spike doesn't try it.

### 4.5 Body size

handlers limits every route's body at 1 MiB unless the app passes
`WithMaxBody` to `Register`, and the option is per registration, not per
route. An upload route needs more than its siblings. D8 asks handlers for
a per-route limit, a directive `//h:body 20MiB` (R21), as its own
amendment; until then an app raises the surface's limit.

## 5. List fields (ADR 0013)

### 5.1 What it renders

`inputs.Field` on a `KindList` control renders (it returns an error
today, visual-interface §6.1):

```html
<fieldset class="ui-list" aria-describedby="…-help">
  <legend class="ui-group-legend">Lines</legend>
  <p class="ui-field-help" id="…-help">…</p>
  <ui-repeat data-ui-path="lines" data-ui-next="2" data-ui-min="1" data-ui-max="50"
             data-ui-texts='{"item":"Item %d","remove":"Remove item %d","added":"Item %d added",…}'>
    <ol class="ui-list-items">
      <li class="ui-list-item">
        <fieldset class="ui-group">
          <legend class="ui-group-legend">Item 1</legend>
          …the item's fields, through inputs.Field…
          <button type="button" class="ui-list-remove" data-ui-remove hidden>Remove item 1</button>
        </fieldset>
      </li>
      …
    </ol>
    <template>…the template row at lines[__i__], the same markup…</template>
    <button type="button" class="ui-list-add" data-ui-add hidden>Add to Lines</button>
    <p class="ui-visually-hidden" role="status"></p>
  </ui-repeat>
  <p class="ui-field-error">…</p>
</fieldset>
```

- Item legends are `ui.inputs.list.item` ("Item %d"), inside the list's
  own legend, so a screen reader hears "Lines, group; Item 2, group" (R23).
  Remove buttons say which item.
- The Add and Remove buttons render `hidden` and the script shows them
  (R24): without script a list edits the items it has, and can neither
  add nor remove.
- **A new form's list** holds `MinItems` blank items: forms fills a list
  that was neither bound nor given items up to its minimum, so a list with
  `MinItems(1)` shows one row on first paint, with script or without
  (R27). Today such a list holds none.

### 5.2 Adding

`<ui-repeat>` clones the template's content and, in the clone, walking
every element **and the content of every `<template>` inside it**
(a `<template>`'s rows aren't its children, so `querySelectorAll` doesn't
reach them):
- replaces **its own segment only**: every occurrence of its prefix,
  `lines[__i__]`, becomes `lines[2]`, in `name`, in `data-ui-path` (so a
  nested `<ui-repeat>` learns its new path), and in `data-ui-when`, where a
  condition may name several fields of the row. A nested list keeps its
  own `__i__`: forms gives both levels the same placeholder
  (`orders[__i__].lines[__i__].sku`, checked), and the prefix
  `orders[__i__]` matches only the first;
- gives every `id` at this level a fresh suffix and rewrites each matching
  token of what refers to it: `for`, `list`, `commandfor`,
  `popovertarget`, and the space-separated lists `aria-controls`,
  `aria-describedby`, `aria-labelledby`. Ids inside a nested template are
  left for that list's own clones;
- sets `data-ui-next` to one more. Indices only grow: forms sorts them and
  renumbers the items `0..n-1` on bind (interface §5.8), so gaps are fine
  and a removed index is never reused in a page.

It appends the clone, renumbers the legends and Remove buttons, moves
focus to the new item's first visible, enabled control (not a hidden
input, nothing inside `[hidden]`), and says "Item 3 added" in its status.
Renumbering and button states touch only the list's own items
(`:scope > ol > li > fieldset > legend`), never a group's or a nested
list's legends, which share the class.

### 5.3 Removing

Remove takes the item out of the DOM and renumbers the legends and Remove
buttons. Focus goes to the next item's first visible, enabled control,
else the previous item's, else Add; the status says "Item 2 removed". There's no undo; the form isn't saved yet.

At `data-ui-max` items Add is unavailable, and at `data-ui-min` every
Remove is: `aria-disabled` with a reason, the state Button gained in A6
(`Unavailable`), so it stays focusable and says why. The server still
checks both (`ErrMinItems`, `ErrMaxItems`).

Rows don't move (R25). An order that matters is a field of the item.

## 6. What other locked text D8 would change

Proposed here, edited by none of this PR: each is part of the contract
amendment A7 (D8.4), locked by epistemer.

**forms ([interface.md](interface.md), locked):**
- §5.4: `Field[T].Remote(search, labeler)`, with its panics and the rule
  that a `Labeler` answers `false` for an id that can't name a row;
  `Hint` panics with the new kinds. §5.10: `KindRemote` and
  `Control.Remote()`. forms imports models' `listing`.
- §5.3 and §5.10: `Form.Upload` and `Form.Uploads` taking the stored
  files, the `Upload` and `StoredFile` types, `KindUpload`, `KindUploads`,
  and how a renderer reads each stored file and its keep state from a
  control (`Control.Stored() iter.Seq2[StoredFile, bool]`); both panic in
  a list.
- §5.10: `Control.Lost() bool`, true for a file field that was posted
  files in a form that failed (§4.4).
- §5.8: an unbound list without items holds `MinItems` blank ones (R27).
- §5.10: `Control.Bounds() (min, max int)` for a list: `<ui-repeat>`
  needs both, and a control exposes neither today (found by the spike).
- §6.2: `rule.Files`, `rule.MinFiles`, `rule.MaxFiles` for `Upload`;
  `MaxSize` adds `data-ui-max-size` to `Control.Attrs`.
- forms-design §10.3's bullets on current-file URLs and search endpoints
  are reversed (R1, R16): both become the field's.

**visual-interface.md (locked):**
- §6.1: the rows of `KindList` (§5 here), `KindFile`/`KindFiles` (the
  drop zone styling and chosen-file list apply to them too), and the new
  kinds.
- §6.2: the shapes of §3.2, §4.3 and §5.1; the classes `ui-combobox*`,
  `ui-upload*`, `ui-list*`; the new keys of §7.
- §10.6: `<ui-repeat>` and `<ui-combobox>` join the script's elements;
  §10 gains the upload behaviours; `data-ui-texts` carries the texts each
  needs.

**server-state-interface.md (locked, A6):**
- §6: `Column.Remote`; the filter table's row for a link column; chips
  name a link filter through the `Labeler`. Its parameter is one the
  table owns (`data-ui-own`), like any filter's, or the live-address
  merge drops it (spiked).

**ADR 0005:** the remote select leaves the list of what emits `hx-*`; it
fetches by itself (R4).

**Other packages:**
- handlers: a per-route body limit (R21), its own amendment.
- models: nothing now, beyond the contract asking every `Labeler` to
  answer `false` for an id that can't name a row; `listing.Labeler`'s doc
  comment could say so in a patch. A batch method waits for a
  measurement (R10).

## 7. Accessibility

| Part | Pattern and checks |
|---|---|
| Remote select | APG combobox (editable, list autocomplete). The input has `role="combobox"`, `aria-expanded`, `aria-controls`; options are `role="option"` with `aria-selected` on the active one; results counted in a polite status; the clear button is named for its field. Checked: keyboard table of §3.4 in LTR and RTL; a screen-reader pass in the spike |
| Upload | A native file input with its label; keep checkboxes labelled with the file name; chosen files listed in a polite region; progress is a native `<progress>`, never focused (the web guidance's rule), shown only for fields with files |
| List | Nested fieldsets with legends; Add and Remove named for the item; focus moves as §5.2 and §5.3 say; min and max explained, not just disabled |

New keys, all `ui.inputs.*`: `combobox.clear` ("Clear %s"),
`combobox.results` ("Results: %d"), `combobox.none` ("No matches"),
`combobox.failed` ("Couldn't search. Try again."), `combobox.more` ("Keep
typing to narrow these down"), `upload.keep` ("Keep %s"), `upload.remove`
("Remove %s"), `upload.undo` ("Undo"), `upload.chosen` ("%s chosen"),
`upload.again` ("Choose the files again"), `upload.too_large` ("%s is
larger than %s"), `upload.wrong_type` ("%s isn't a type this field
takes"), `list.item` ("Item %d"), `list.add` ("Add to %s", the list's label),
`list.remove` ("Remove item %d"), `list.added`, `list.removed`,
`list.at_most` ("Maximum: %d"), `list.at_least` ("Minimum: %d"; texts
with no noun after the count, since i18n has no plurals). Texts are drafts until the lock.

## 8. The slice spike (D8.2)

In `internal/spike` again, on the toy shop, with the shapes above declared
locally (no forms amendment before the lock). Done when each of these runs
with script on and off, LTR and RTL, light and dark, and passes goldens,
`CheckIDs`, `CheckNames` and `CheckSelectors`:

1. **The order form**: a customer remote select, a receipt upload with a
   stored file, and a list of lines, each with a product remote select and
   a quantity.
2. **Composition**, which the parts alone don't show:
   - a remote select inside a cloned row: the clone's own ids, its own
     index, its search working;
   - a remote select inside a list inside a list (nested `__i__`);
   - a remote select inside the `ActionDialog`: the popup above the
     modal, Esc closing the popup and not the dialog;
   - a remote select as a filter in a `<ui-table>`: its search isn't a
     state request, the chip shows the name;
   - a plain `File` inside a list row, and the panic of `Upload` there;
   - a 422 from the order form: the keep checkboxes as left, the note on
     the file fields (R20), the stored receipt kept on the next submit;
   - the combobox's events reaching a `data-ui-when` condition, and its
     Enter and Esc inside the dialog and inside a form;
   - a search answered by a redirect to sign-in.
3. **Measurements**: the upgrade's flash of the id (§3.2), and `LabelOf`
   calls for a 50-line order (R10).
4. **The keyboard table** of §3.4 in a script test where the DOM allows,
   and in the browser pane otherwise.

Its findings amend this design before the contract is drafted, as D7.2's
did.

### 8.1 What the spike found

The spike ran on 2026-11-22 in `internal/spike`, and was reviewed
([0020](reviews/0020-d8-2-spike.md)); its script's fixes were checked
again in the pane. It was mounted by the catalogue at `/spike/order` and `/spike/table`, with htmx 2.0.11. Its Go tests check
the pages with `CheckIDs`, `CheckNames` and `CheckSelectors`, the
`Labeler` rules and the keep keys; the browser pane checked the rest, by
script and, for Esc, Enter and Tab, with real keys.

What held, as written:
- **The combobox**: the upgrade from the shape without script; searching,
  cancelling, counting results; Down, Up, Enter, Esc (three presses
  inside the modal dialog: popup, text, then the dialog), Tab clearing
  empty text and skipping the clear button, Enter submitting only a
  committed value; the popup above the modal dialog (a hit test lands in
  it); a redirected answer failing cleanly; no htmx request and no
  `aria-busy` inside a `<ui-table>`; the refused link left as typed; the
  hidden input's events reaching a filter form. LTR and RTL, light and
  dark: the popup lines up with the input either way.
- **The list**: adding into a nested list (the inner list's path and its
  template's `__i__`, rewritten as §5.2 says); fresh ids with every
  reference intact (none duplicated, none broken); a condition rewritten
  and evaluated in the clone; renumbered legends and Remove buttons; focus
  after add and remove; both bounds unavailable with their reason; a
  combobox in a clone upgrading itself.
- **The upload**: keep cards over the checkboxes; the chosen list and its
  status; the size check through `setCustomValidity`; after a 422, the
  keep state as left, the note, and the next save keeping only what was
  kept; a repeated or unknown key dropped.
- **`LabelOf`**: once per id per request (five lines of one product made
  one call); text that didn't convert never looked up; a miss shown as
  the id; a failing `Labeler` a 500.

What changed the design:

| # | Finding | Change |
|---|---|---|
| 1 | models' `option.SelectOption` has `Label() i18n.TKer`, a key; a row's name is data | R5's default is ui's own `components.Suggestion{Value, Label i18n.Ter}` (§3.3) |
| 2 | forms' `Control` exposes neither bound of a list, which `<ui-repeat>` needs | `Control.Bounds()` joins the contract list (§6) |
| 3 | In a nested list, two buttons named "Add item" can't be told apart | `list.add` is "Add to %s", with the list's label (§5.1, §7) |
| 4 | "At least 1 items": a count followed by a noun needs plurals, which i18n lacks | "Minimum: %d", "Maximum: %d" (§7) |
| 5 | A hidden page runs no animation frames, so a status set in one is never written | The script announces through a timer, never a frame |
| 6 | htmx 2.0.11 fires `htmx:xhr:progress` for the upload and the download alike, with nothing to tell them apart | The bar listens to the request's own `xhr.upload`, from `htmx:beforeSend` (§4.3; the first fix, following events until one reached its total, failed review 0020) |
| 7 | A filter the table doesn't own is dropped by the live-address merge | `Column.Remote`'s parameter is owned like any filter's (§6) |
| 8 | "1 results" has the plural problem of finding 4 | "Results: %d" (§7) |
| 9 | ui's forms are `novalidate`, so `setCustomValidity` blocks no submit (review 0020) | The script refuses the submit itself (§4.3) |

Not settled by the spike:
- The upgrade's flash of the id: the pane was hidden, so nothing painted
  to measure. It moves to the build, in a visible pane.
- A screen-reader pass: none was run; the roles, names and live regions
  were checked in the DOM only.
- A real `<ui-table>` filter: `tables` has no `Remote` yet, so a stand-in
  form inside a `<ui-table>` carried the check, and no chip was drawn.
- A single-file `Upload` replacing its stored file, and `Required`,
  `MinFiles` and `MaxFiles` on an upload: the spike's receipt is an
  `Uploads` bound by hand, keeping and adding only.
- A text id (a uuid) in a remote select: every spiked id is an integer.

## 9. Decisions

All stand on their defaults until epistemer answers. Each is API design.

| # | Question | Default | Alternative |
|---|---|---|---|
| R1 | Where "this field is a remote select" is declared | On the forms field: `Field[T].Remote(search, labeler)` and `KindRemote`, so groups, list rows and table filters render it generically. Reverses forms-design §10.3 ("widget parameters… are arguments of the widget call") | As §10.3 says: `inputs.Remote(c, search, labeler)`, plus an override in `inputs.Field` keyed by path that must match `lines[__i__].product` and `lines[3].product` alike, and a separate hook in `TableOf` for filters |
| R2 | Who calls the `Labeler` | `inputs`, at render. forms carries it and never calls it | forms calls it in `Validate`, and a miss is a field error: a models clarification first, since `LabelOf` is documented for display |
| R3 | A remote select without script | A text input of the id, with the chosen name beside it | Nothing usable: the field renders read-only |
| R4 | How the search is requested | The ui script's own `fetch`, debounced and cancelled; no `hx-*`. Removes the remote select from ADR 0005's `hx-*` list | htmx (`hx-get` on the input), with the actions and table scripts taught to ignore it and `htmx.process` called on clones |
| R5 | The suggestion type, and parsing the search request | ui's `components.Suggestion{Value, Label}` (a struct, `Label` an `i18n.Ter`); no parser, `q` is read by the handler. Models' `option.SelectOption` doesn't fit: its label is a key | A `Search(r)` helper, and a description line per suggestion |
| R6 | More suggestions than fit | The first 20, then "Keep typing to narrow these down" | A "Show more" option that fetches the next page |
| R7 | The gate refuses the search link | The shape without script, not upgraded: an id can be typed, nothing is searched. Markup can't make a field read-only against a crafted post; the form's handler decides what it accepts | The field renders read-only (an earlier draft), which only hides the control |
| R8 | `LabelOf` finds nothing, or fails, at render | Nothing: the id is shown. An error: the render fails. Text that didn't convert is never looked up | Both show the id, and the error is logged |
| R9 | Is a posted id checked against the `Labeler`? | No: the app's own check, which also knows who may pick it | Yes, in `Validate` (R2's alternative) |
| R10 | `LabelOf` calls in a render | Memoized per request by `Labeler` and id, so repeated ids cost one call | No memo (one call per field); or a batch method on `Labeler` (a models amendment) |
| R11 | Several values in one remote select | Not in D8 (models' `listing` has no `IN` either) | A multi-value remote select with chips |
| R12 | Creating a record from inside the select | Not in D8; an `Open` action beside the field | A "New…" option opening the action dialog, whose reply chooses the new record |
| R13 | A table filter on a link column | `tables.Col(c).Filter().Remote(search, labeler)`, `Eq` on a `Text` or `Integer` column | A kind of its own in models' `column` (`Link`, with its target), a models amendment |
| R14 | Where a file goes | With its form, to the form's handler; ui stores nothing | An upload endpoint per field that stores at once and posts back a key (needs a storage package) |
| R15 | Keeping and removing stored files | `Form.Upload`/`Uploads(name, stored)` with `Upload{Kept, Added}`; kept keys post back, deduplicated, and must be among `stored` | Keep `File`/`Files` and let the app declare a `Texts("x.keep")` field and check the keys itself |
| R16 | Where a stored file's name and link come from | `StoredFile`, given to the declaration. Reverses §10.3's bullet on current-file URLs | An argument of `inputs.Field`, with the same path-keyed override as R1's alternative |
| R17 | Thumbnails of chosen files | Opt-in (`Previews()`), since they need `img-src blob:` | Always on, and every app's CSP allows `blob:` images |
| R18 | Drag and drop | The native file input is the drop zone; the script only highlights | A script drop handler on a larger zone |
| R19 | Upload progress | A `<progress>` per upload field with files chosen, fed by `htmx:xhr:progress`, hidden at the answer, never announced; nothing without htmx | None |
| R20 | Files chosen in a form that fails | A note asking for them again, through `Control.Lost()`. `hx-preserve` is rejected (§4.4) | No note: the browser's empty file input says it |
| R21 | A larger body for an upload route | Ask handlers for `//h:body <size>` per route | The app raises its surface's limit |
| R22 | Checking size and type before sending | Yes, in the script, with the server's translated message | Server only |
| R23 | An item's legend | "Item %d", one key for every list | `List.ItemLabel(i18n.Ter)` on forms, "Line 2" (a forms amendment) |
| R24 | Adding and removing rows without script | Not possible; the buttons are hidden until the script shows them | Submit buttons the handler recognises, with a forms amendment for adding and removing items while binding without validating |
| R25 | Reordering rows | None | Move up and Move down buttons that renumber |
| R26 | `Upload` inside a list | Panics: an item built during `Bind` has no record to take stored files from. `File` and `Files` work there | Stored files keyed by a row id field the item declares |
| R27 | A new form's list with `MinItems(n)` | forms fills an unbound list without items to `n` blank items | The app calls `List.Add()` n times |
| R28 | Names | `KindRemote`, `Remote`, `Upload`, `Uploads`, `StoredFile`, `KindUpload(s)`, `Control.Stored`, `Control.Lost`, `rule.Files`, `MinFiles`, `MaxFiles`, `components.Suggestions`, `Suggestion` (a struct), `SuggestionLimit`, `Control.Bounds`, `data-ui-texts`, `<ui-combobox>`, `<ui-repeat>` | epistemer's choice at the lock |

## 10. Out of this round

- A storage package, direct-to-storage uploads, image resizing,
  thumbnails made on the server.
- Multi-value remote selects (with models' `IN`).
- Creating a record from a select.
- Reordering list rows; drag to reorder.
- A command palette for global search: it shares the combobox's keyboard,
  and can reuse `<ui-combobox>`'s module when it comes.
