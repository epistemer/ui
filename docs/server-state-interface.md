# ui: interface of actions and tables

| | |
|---|---|
| Status | **Locked by epistemer on 2026-11-22, with the defaults of T1–T11** (reviewed in [0015](reviews/0015-d7-5-contract.md)). It is amendment A6 to [visual-interface.md](visual-interface.md), whose §13.5 records it. A change that alters what an app writes or gets back is an amendment |
| Date | 2026-11-22 |
| Roadmap item | D7.5 |
| Inputs | [server-state-design.md](server-state-design.md) (decisions S1–S22, the spike's findings §12.1), ADRs [0008](adr/0008-links.md)–[0010](adr/0010-tables.md), reviews [0012](reviews/0012-server-state-design.md) and [0014](reviews/0014-d7-2-spike.md), the spike in `internal/spike`, handlers v0.2.0 (`handlers.Link`, `handlers.Gate`) and models v0.4.0 (`column`) |

The contract of `ui/actions`, `ui/tables`, their test kits, and what
`ui/components`, `ui/elements`, `ui/layout`, the CSS and the ui script gain
for them. It follows the design and the spike: every section names the
spike's package that tried it, or says it wasn't tried.

The conventions of visual-interface.md §2 hold throughout: builders,
variants as data attributes, `Attrs` on every spec, programmer mistakes
panic at render (at declaration for actions and tables, §2 below), `Ter`
for every text, logical CSS.

## 1. Module

| Package | Imports (beyond the standard library) | Spiked as |
|---|---|---|
| `ui/actions` | the kernel, templ, i18n, errors | `internal/spike/actions` |
| `ui/actions/actionstest` | `ui/actions`, `testing` | `…/actions/actionstest` |
| `ui/tables` | the kernel, templ, i18n, errors, `ui/forms`, `ui/actions`, models' `column`, `listing` and `option` | `…/tables` |
| `ui/tables/tablestest` | `ui/tables`, `ui/actions`, `ui/actions/actionstest`, `testing` | `…/tables/tablestest` |
| `ui/components` | gains `ui/actions`, `ui/tables`, `github.com/epistemer/errors` | `…/tableui` |

ui's go.mod requires models v0.4.0. ui imports no handlers and no auth
(S7): a route link reaches ui as `ui.Link`, and the gate is a function.

## 2. Actions (`ui/actions`) (spiked)

```go
package actions

// Kind is how an item runs, taken from its link, never chosen by a caller.
type Kind int

const (
	Navigate Kind = iota // a GET link
	Request              // any other method: a form post, or htmx's verb
	Open                 // a GET whose answer shows in the action dialog
)

// Action is declared once, at package level.
type Action struct{ /* unexported */ }

func New(label i18n.Ter) Action
func (a Action) Icon(c templ.Component) Action
func (a Action) Danger() Action
func (a Action) Primary() Action
func (a Action) Confirm(question i18n.Ter) Action
func (a Action) Dialog() Action             // an Open action
func (a Action) Attrs(at templ.Attributes) Action
func (a Action) To(l ui.Link) Bound

// Bound is an action with its link, per record, in a request.
type Bound struct{ /* unexported */ }

func (b Bound) Unless(err error) Bound // unavailable, with err as the reason, when err != nil
func (b Bound) If(ok bool) Bound       // absent when false

// Item is what a placement reads, as forms.Control is for a field.
type Item struct {
	Label    i18n.Ter
	Icon     templ.Component // nil for none
	Kind     Kind
	Method   string        // GET, POST, PUT, PATCH, DELETE
	URL      templ.SafeURL // sanitised (visual-interface.md §2.7)
	Question i18n.Ter      // nil for none
	Danger   bool
	Primary  bool
	Reason   error // non-nil: the item is unavailable, and this says why
	Attrs    templ.Attributes
	// and unexported fields: the action's identity, for Is
}

func (it Item) Is(a Action) bool // whether the item was bound from a

// Gate is set once per request; Resolve reads it.
var Gate *ui.Provider[func(ui.Link) bool]

func AllowAll(ui.Link) bool                     // for an app with no permissions
func Allowed(ctx context.Context, l ui.Link) bool // asks the gate; for a nav item, a tab, a link in a cell

// Resolve asks the gate about each link and describes what it admits, in
// order. An empty result is an empty, non-nil slice.
func Resolve(ctx context.Context, bound ...Bound) []Item
```

- **Declaration mistakes panic** where the action is declared: a blank
  label or question (read with `context.Background()`), `Dialog` with
  `Confirm`, `Primary` with `Danger`.
- **Identity.** Every builder call returns a new action, so two actions
  built from one base are two actions for `Is`; the package-level variable
  is the action.
- **Absent** (no item): the gate refuses the link; `If(false)`; a link
  with no address (§2.7 of visual-interface.md). Binding happens per row
  in a request, so it never panics.
- **The method** comes from the link: a link with a method
  `Method() string` (handlers' `Link`) gives it; any other link is a GET.
  So `Kind` is `Open` for a `Dialog` action, else `Navigate` for GET, else
  `Request`. A `Dialog` action bound to a link that isn't a GET panics in
  `Resolve`, naming the action: it would send a GET to a route that
  changes something.
- **A `Request` or `Open` item needs a local address**, as `Reply`'s
  `Else` does (§4.4): with another, it is absent. Only a `Navigate` item may
  go to another site, as a plain link.
- **No gate in the context, or a nil one, panics** in `Resolve` and
  `Allowed`, naming `actions.Gate` (S9). With handlers:
  `actions.Gate.With(ctx, handlers.Gate(admit))`, `admit` from the guard's
  `Admit(r)`.
- `Reason`'s text for people is `errors.T(ctx, it.Reason)`. The function
  that returned it (`shop.CanRefund`) is the one the handler runs before it
  acts.

### 2.1 `actionstest` (spiked)

```go
type Option func(*options)

func Refuse(links ...ui.Link) Option // the gate refuses these addresses
func Context(t testing.TB, opts ...Option) context.Context // a gate that admits the rest
func Assert(t testing.TB, items []actions.Item, want ...actions.Action) // these actions, in order
func Unavailable(items []actions.Item, a actions.Action) error // the item's reason, or nil
```

## 3. Tables (`ui/tables`) (spiked)

### 3.1 Declaring

```go
package tables

type Dir bool

const (
	Asc  Dir = false
	Desc Dir = true
)

// Column is a generated column (models' column.Column) with what the page
// says about it (S22).
type Column[R any] struct{ /* unexported */ }

func Col[R any](c column.Column[R]) Column[R]
func (c Column[R]) Searchable() Column[R]
func (c Column[R]) Sortable() Column[R]
func (c Column[R]) Filter() Column[R]
func (c Column[R]) Hidden() Column[R]
func (c Column[R]) Link(f func(R) ui.Link) Column[R]
func (c Column[R]) Amount(f func(R) i18n.Ter) Column[R] // money, formatted by the app
func (c Column[R]) Image(src func(R) ui.Link, alt func(R) i18n.Ter) Column[R] // an image, its address from a route (S8)

type Table[R any] struct{ /* unexported */ }

func New[R any](name string, key column.Column[R], cols ...Column[R]) *Table[R]
func (t *Table[R]) Label(l i18n.Ter) *Table[R]                  // the caption; required
func (t *Table[R]) RowLabel(c column.Column[R]) *Table[R]       // names a row's controls
func (t *Table[R]) Sort(c column.Column[R], d Dir) *Table[R]    // when the URL names none
func (t *Table[R]) Sizes(n ...int) *Table[R]                    // default 20, 50, 100
func (t *Table[R]) Unpaged() *Table[R]
func (t *Table[R]) Prefixed() *Table[R]                         // parameters under "<name>."
func (t *Table[R]) Actions(f func(context.Context, R) []actions.Item) *Table[R]
func (t *Table[R]) Bulk(f func(context.Context) []actions.Item) *Table[R]
func (t *Table[R]) Check() error                                // what needs the whole declaration

// Keyed ids, the same in a page and in any later answer (§3.5).
func (t *Table[R]) ID() string
func (t *Table[R]) RowID(key string) string // "<name>-row-<key>", the key escaped
// Selected reads the checked rows' keys: from the body for a POST, from
// the query for a GET (a bulk Open), under the table's parameter [T6].
func (t *Table[R]) Selected(r *http.Request) []string
func (t *Table[R]) SelectionName() string // the parameter, for a form that carries the selection on
```

Panics at declaration (`tables: <name>: …`):
- a name not `[a-z][a-z0-9-]*`, no columns, a column twice;
- a key that is neither text nor an integer;
- a filter on a column named `q`, `sort`, `page`, `size` (S13) or
  `selected` (the selection's, T6; clarified in T2), or whose name forms
  can't take as a field's;
- `Searchable` on a column that isn't text; `Amount` on one that isn't an
  integer or a number (an integer of minor units is money too); more than
  one of `Link`, `Amount` and `Image` on one column;
- `RowLabel` or `Sort` naming a column that isn't the table's, or `Sort`
  one that isn't `Sortable`;
- `Sizes()` with no size, or a size below 1;
- any builder called after the table's first `Bind` or `Row`: a table is
  declared once and only read afterwards.

`Check` returns, and `tablestest.Check` fails a test for: no `Label`; a
column without a label; a table with actions or bulk actions and no row
label (the `RowLabel` column, shown and text, or else a shown text
column), since a row's controls are named by it. The first `Bind` or `Row`
panics on what `Check` reports.

A text key may hold any character: ids escape it (every byte outside
`[A-Za-z0-9-]` as `_xx`, hexadecimal; an underscore too, so an escape is
never ambiguous: clarified in T2), so an id stays one token and a valid
selector.

### 3.2 A request

```go
func (t *Table[R]) Bind(r *http.Request) *State[R]

type State[R any] struct{ /* unexported */ }

func (st *State[R]) Valid() bool                         // false when a filter's value doesn't parse
func (st *State[R]) ListingQuery() listing.Query         // so a State is a listing.Source
func (st *State[R]) Errors() iter.Seq[*forms.FieldError]
func (st *State[R]) Page(rows []R, total int64) View[R]  // total as models' List<Type> gives it [T8]
func (st *State[R]) Rejected() View[R]                   // the filters and their errors; no rows, empty state or pager (S12)
// Answer sets what the response needs and returns its status: 422 when the
// state isn't valid, else 200. A rejected state answering a request that
// targeted only the results retargets it to the whole table
// (HX-Retarget and HX-Reselect "#<name>"), so the error shows (finding 18). [T5]
func (st *State[R]) Answer(w http.ResponseWriter, r *http.Request) int

// Targeted reports whether an htmx request targets this table or its
// results, so a page of several tables can query only that one.
func Targeted[R any](r *http.Request, t *Table[R]) bool

// Row describes one row outside any page, for an action's answer.
func (t *Table[R]) Row(ctx context.Context, row R) Row[R]
```

A handler:

```go
st := Orders.Bind(r)
if !st.Valid() {
	return page(w, st.Answer(w, r), ordersPage(st.Rejected()))
}
rows, total, err := q.ListOrderListView(r.Context(), st)
if err != nil {
	return err
}
return page(w, st.Answer(w, r), ordersPage(st.Page(rows, total)))
```

### 3.3 The URL (spiked)

| Parameter | Meaning | Junk |
|---|---|---|
| `q` | the search, trimmed, at most 200 runes | — |
| `sort` | a sortable column's name, `-` before it for descending | the table's own order, silently |
| `page` | from 1 | page 1, silently |
| `size` | one of `Sizes` | the first, silently |
| `<column>` | an equality filter (text, enum, bool) | an error on the filter |
| `<column>.from`, `<column>.to` | a range (integer, number, date, time), both ends inclusive; a time's `to` takes the whole day | an error; `from` after `to` is an error on the group |

- With `Prefixed()`, every parameter is `<name>.<parameter>`.
- The key ends every sort, in the sort's direction, unless the sort is
  the key: paging is stable.
- **The sort and the size follow the state form** as hidden inputs in the
  results, joined to it by `form`, so a search or a filter keeps them.
- **A search sends the whole state form**, filters typed and not yet
  applied included: the box and the filters are one form.
- **Parameters the table doesn't own are kept.** Links carry them, and
  the state form holds them as hidden inputs, as rendered.
- **Every address `Bind` builds is local**: the request's path with its
  leading slashes collapsed to one, never a scheme or a host. With script, a
  state request takes the table's own parameters from the control and
  every other one from the live address (finding 1), and drops empty ones.
- A change to search, filters, sort or size goes back to page 1.

### 3.4 Filters (spiked)

One `forms.Form` built from the filterable columns, bound with
`forms.FromValues` from the query string. Every field is labelled by its
column (a range's group too, finding 12) and rendered by `inputs.Field`.

| Kind | Field | Query |
|---|---|---|
| text | `Text` | `=` |
| enum | `Text` with the column's `Options`: a select that refuses a value not offered | `=` the option's `Value()`, a string (finding 11) |
| bool | `Bool` | `=` |
| integer, number | a group of `from` and `to` | `>=`, `<=` |
| date, time | a group of two `Date`s | `>=`; `<=` (date) or `<` the next day's start (time), UTC |

### 3.5 The view

`View[R]` describes a page and holds no request. Every part is exported
for an app that renders the table itself.

```go
type View[R any] struct {
	ID, StateID, ResultsID, SelectID, CaptionID, StatusID string
	Label    i18n.Ter
	URL      string   // the current address, which a refresh reloads
	Own      []string // the query parameters the table owns
	Rejected bool
	Columns  []Head
	Rows     []Row[R]
	Search   Search
	Filters  []forms.Control
	Chips    []Chip
	Clear    templ.SafeURL // removes the search and every filter; "" when none is set
	Hidden   [][2]string   // the query's other parameters, for the state form
	Sort     [2]string     // the sort's parameter and value; "" for the table's own order
	Size     [2]string     // the size's parameter and value; "" for the first size
	Pager    Pager
	Bulk     []actions.Item
	Empty    Empty
	EmptyLink templ.SafeURL
	Status   string // the sentence said after a change: "21–40 of 312", "No results"
}

type Head struct {
	Name     string
	Label    i18n.Ter
	Kind     column.Kind
	Sortable bool
	Sorted   bool
	Desc     bool
	Link     templ.SafeURL // sorts by it next
	ID       string        // "<table>-sort-<name>"
}

type Row[R any] struct {
	Row        R
	Key, ID    string // ID: "<table>-row-<key>"
	Label      string // the row label cell's text
	LabelID    string // "<table>-row-<key>-label"
	Selectable bool   // the table has bulk actions
	Cells      []Cell
	Actions    []actions.Item
}

type Cell struct {
	Column string
	Kind   column.Kind
	Value  column.Value
	Link   templ.SafeURL
	Amount i18n.Ter
	Src    templ.SafeURL // an Image column's address
	Alt    i18n.Ter      // its alternative text
	Mark   string   // the search text, in a searchable text cell
	Label  bool     // the row's label cell
}

type Search struct{ Name, Text string }

type Chip struct {
	Label  i18n.Ter
	Value  string        // the bound value, formatted: an option's label, a bool's word, a date as Date renders it (finding 20)
	Remove templ.SafeURL
}

type Empty int

const (
	NotEmpty Empty = iota
	NoRows   // none at all
	NoMatch  // none for these filters; EmptyLink clears them
	PastEnd  // a page past the last; EmptyLink goes to the last
)

type Pager struct {
	Label                           i18n.Ter // "Pages of <table>" (finding 9)
	From, To                        int
	Total                           int64
	Page, Pages                     int
	First, Prev, Next, Last         templ.SafeURL // "" at an end
	FirstID, PrevID, NextID, LastID string
	Window                          []PageLink    // the current page and two each side
	Sizes                           []SizeLink    // none with one size
	Unpaged                         bool
	Target                          string // the results' id its links swap; "" for none
}

type PageLink struct {
	N       int
	URL     templ.SafeURL
	Current bool
	ID      string // "<table>-page-<n>"
}

type SizeLink struct {
	N       int
	URL     templ.SafeURL
	Current bool
}
```

The row label is the `RowLabel` column, else the first shown text column
(§3.1).

### 3.6 `tablestest` (spiked)

```go
// Bind binds a query string with a gate that admits everything but what
// actionstest.Refuse names.
func Bind[R any](tb testing.TB, t *tables.Table[R], query string, opts ...actionstest.Option) *tables.State[R]
func Query[R any](st *tables.State[R]) listing.Query
func Errors[R any](st *tables.State[R]) []string // "<path>: <key>"
func Check[R any](tb testing.TB, t *tables.Table[R])
```

Amendment A7 ([rich-inputs-interface.md](rich-inputs-interface.md) §5,
locked 2026-11-22) adds `Column.Remote`: a filter on a link column through
a remote select.

## 4. Components (`ui/components`) (spiked)

### 4.1 The table and its pager

```go
type TableView[R any] struct{ /* unexported */ }

// TableOf is declared once at package level, so a page and an action's
// answer render a row alike.
func TableOf[R any](t *tables.Table[R]) *TableView[R]
func (tv *TableView[R]) Cell(c column.Column[R], f func(R) templ.Component) *TableView[R]
func (tv *TableView[R]) Attrs(a templ.Attributes) *TableView[R] // on <ui-table>
func (tv *TableView[R]) Page(v tables.View[R]) templ.Component
func (tv *TableView[R]) Row(row R) templ.Component // resolved with the render's context

type PaginationSpec struct{ /* unexported */ }

// Pagination renders a table's pager; its links swap the pager's Target
// when it has one. Building a pager without a table isn't offered yet.
func Pagination(p tables.Pager) PaginationSpec
func (s PaginationSpec) Attrs(a templ.Attributes) PaginationSpec
```

The markup, with the keyed ids of §3.5:

```html
<ui-table id="orders" class="ui-table" data-ui-own="q sort page size status placed_at.from placed_at.to" data-ui-selected="%d selected">
  <p class="ui-table-status ui-visually-hidden" id="orders-status" role="status"></p>
  <form id="orders-state" class="ui-table-tools" method="get" action="/orders" hx-…>
    search (label, icon, input), the filters by inputs.Field, hidden inputs, Apply
  </form>
  <form id="orders-select" method="post" hidden></form>   <!-- with bulk actions -->
  <div id="orders-results" class="ui-table-results" data-ui-status="1–10 of 57" data-ui-url="/orders?…">
    <input type="hidden" name="sort" value="-total" form="orders-state">
    <input type="hidden" name="size" value="50" form="orders-state">
    <ul class="ui-table-chips">chips, then <a class="ui-table-clear"></ul>
    <p class="ui-table-empty">…</p>   <!-- or: -->
    <div class="ui-table-scroll" role="region" tabindex="0" aria-labelledby="orders-caption">
      <table class="ui-table-grid">
        <caption id="orders-caption">Orders</caption>
        <thead><tr>
          <th scope="col" class="ui-table-check"><span id="orders-select-label" class="ui-visually-hidden">Select</span>
            <input type="checkbox" class="ui-table-all" aria-label="Select every row on this page" data-ui-select-all hidden></th>
          <th scope="col" data-kind="number" aria-sort="descending"><a class="ui-table-sort" id="orders-sort-total" href hx-…>Total</a></th>
          <th scope="col" class="ui-table-actions"><span class="ui-visually-hidden" id="orders-actions-label">Actions</span></th>
        </tr></thead>
        <tbody><tr id="orders-row-1042">
          <td class="ui-table-check"><input type="checkbox" name="selected" value="1042" form="orders-select" aria-labelledby="orders-select-label orders-row-1042-label"></td>
          <td data-kind="text" id="orders-row-1042-label">…</td>
          <td class="ui-table-actions">ActionMenu, keyed "orders-row-1042-actions"</td>
        </tr></tbody>
      </table>
    </div>
    <div class="ui-table-bulk" role="group" aria-label="With the selected rows">
      <button type="submit" id="orders-bulk-1" form="orders-select" formaction="/orders/ship" hx-post="/orders/ship" hx-include="#orders-select" hx-swap="none" …>Mark shipped</button>
      <button type="submit" id="orders-bulk-2" form="orders-select" formaction="/orders/tag" formmethod="get" hx-get="/orders/tag" hx-include="#orders-select" hx-target="#ui-action-dialog-body" hx-swap="innerHTML">Tag</button>
    </div>
    <nav class="ui-pagination" aria-label="Pages of Orders">…</nav>
  </div>
</ui-table>
```

- **Cells** go through the value renderers by kind: `Text`, `Bool`,
  `Date`, `DateTime`, `Enum`, `Link` (a `Link` column), `Image`, `Amount`,
  `Empty` for a null. An infinite date or time shows its `Text`
  ("infinity"). Integers (`Value.Int`) and numbers (`Value.Text` for a
  numeric's exact digits, else `Value.Num`) go through `Text`, aligned to the
  end with tabular numerals by `data-kind="number"` [T7]. A searchable text
  cell marks the search's hits with `<mark>`, except where lowering any of
  its runes changes that rune's length in bytes. `Cell` replaces one
  column's cell.
- **htmx attributes** sit on the state controls only, never on the table
  or its results (they inherit): the search box (`input changed delay:300ms,
  search`), the sort, page, size, chip and clear links target the
  results; the state form targets the table. Each has `hx-select` of its
  target, `hx-swap="outerHTML"`, `hx-push-url="true"` and
  `hx-sync="#<table>:replace"`.
- **The bulk bar** shows while a row is checked, by CSS
  (`:has(input[name="selected"]:checked)`), and follows the table in the
  tab order. Its buttons belong to the selection form (`form`,
  `formaction`, `hx-include` of the form); an `Open` one submits it with
  `formmethod="get"`, so a page holding the form gets the keys in its
  address, and with htmx the same request loads the action dialog. A form
  that acts on the selection carries it on as hidden inputs named
  `SelectionName()`, and its handler reads `Selected(r)`.
- **The pager**: a named `<nav>`, the range, first, previous, a window of
  numbered pages (each named "Page 3", `aria-current="page"` on the
  current), next and last in words; a disabled end is a `<span
  aria-disabled="true">`, not a link; size links named "25 rows a page",
  the current with `aria-current="true"`.

### 4.2 Action placements

```go
type ActionButtonsSpec struct{ /* unexported */ }

func ActionButtons(items []actions.Item) ActionButtonsSpec
func (b ActionButtonsSpec) Attrs(a templ.Attributes) ActionButtonsSpec

type ActionMenuSpec struct{ /* unexported */ }

func ActionMenu(label i18n.Ter, items []actions.Item) ActionMenuSpec
func (m ActionMenuSpec) ID(id string) ActionMenuSpec        // a keyed id for the trigger [T3]
func (m ActionMenuSpec) LabelledBy(id string) ActionMenuSpec // a row's label cell joins the trigger's name
func (m ActionMenuSpec) Attrs(a templ.Attributes) ActionMenuSpec
```

| Kind | Renders |
|---|---|
| `Navigate` | a link (`Button().Href`, `MenuItem().Href`), with the confirmation when it has a question |
| `Request` | a submit button of the action form: `form="ui-action-form"`, `formaction` the URL (`formmethod="post"` for another method, which needs htmx), the request attribute for its method, `hx-swap="none"`, and the confirmation (`components.Confirm`) when it has a question |
| `Open` | a link to the form's page, with `hx-get`, `hx-target="#ui-action-dialog-body"`, `hx-swap="innerHTML"` |
| an unavailable item (any kind) | no `href`, `form`, `formaction`, question or request attribute: inert without script. A Button `Unavailable(reason)`, or a MenuItem `Unavailable(reason)` [T2] |

- `ActionButtons`: the primary item solid, a danger one outlined in the
  danger tone, the rest outlined neutral.
- `ActionMenu`: a ghost icon trigger (`icons.More`) named by the label and,
  with `LabelledBy`, the row's label: "Actions omar42@example.com". Danger
  items last, after a separator.
- In a table, each row's menu is keyed `<table>-row-<key>-actions`, and
  each bulk button `<table>-bulk-<n>` (from 1), so focus can come back to
  it after a refresh replaced it (finding 5). When it is gone (the row was
  removed or filtered out, or the bulk bar is hidden because the selection
  cleared), focus goes to the table's scroll region, else its search box.
- Each placement renders nothing for no items.

### 4.3 The action form and dialog

```go
func ActionForm() templ.Component   // <form id="ui-action-form" method="post" hidden>
func ActionDialog() templ.Component // <dialog id="ui-action-dialog" class="ui-dialog ui-action-dialog" aria-label="Action">
func InActionDialog(r *http.Request) bool // HX-Target is the dialog's body: render the form alone, with its ActionTitle
func ActionTitle(t i18n.Ter) templ.Component // <h2 class="ui-dialog-title" id="ui-action-dialog-title">, which names the dialog
```

- The shells render both, beside `ConfirmDialog` and `Toasts`, outside
  `#ui-main` (§5) [T9].
- **The dialog's name** is the loaded content's title: an element with id
  `ui-action-dialog-title`, which the script points `aria-labelledby` at.
  Empty, or with content that has none, its `aria-label` names it
  (finding 7). It has the locked Dialog's classes and close button, its own
  markup: the locked Dialog has a fixed title and a trigger.
- The script opens it when content arrives in `#ui-action-dialog-body`,
  focuses the first field, and empties it, dropping the loaded name,
  however it closes.
- A form in it posts with `hx-post` targeting the body. A failure answers
  422 with the form and its errors, which the script swaps, and the dialog
  stays open.

### 4.4 Answers: `Reply`

```go
type ReplySpec struct{ /* unexported */ }

func Reply(w http.ResponseWriter, r *http.Request) *ReplySpec
func (a *ReplySpec) Toast(t i18n.Ter) *ReplySpec
func (a *ReplySpec) Replace(id string, c templ.Component) *ReplySpec
func (a *ReplySpec) Remove(id string) *ReplySpec
func (a *ReplySpec) Refresh(tableID string) *ReplySpec
func (a *ReplySpec) Fail(err error) *ReplySpec // [T4]
func (a *ReplySpec) Else(l ui.Link) error
func (a *ReplySpec) Go(l ui.Link) error
```

| | With htmx | Without |
|---|---|---|
| `Replace(id, c)` | `c` replaces `#id` (`HX-Retarget: #id`, `HX-Reswap: outerHTML`); a `<tr>` travels as the answer's own content | 303 to `Else` |
| `Remove(id)` | `#id` is removed (`HX-Retarget: #id`, `HX-Reswap: delete`) | 303 to `Else` |
| `Refresh(table)` | the table reloads its results at its current URL (`ui-table-refresh`; `HX-Reswap: none`) | 303 to `Else` |
| `Fail(err)` | 422, a danger toast with `errors.T(ctx, err)`, `HX-Reswap: none`: nothing changes, a dialog's form stays as it is | 303 to `Else`; `errors.T(ctx, err)` goes to the request's `Flash` as an error *(A9)* |
| no change | nothing changes (`HX-Reswap: none`) | 303 to `Else` |
| `Go(l)` | the browser goes to `l` (`HX-Redirect`) | 303 to `l` |

*(A9)* Without htmx, every 303 hands the toast to the request's
`components.Flash`; `Go` does with htmx too. See
[confab-1-interface.md](confab-1-interface.md) §6.

- **One change per answer, ended by `Else` or `Go`.** A second change
  (`Replace`, `Remove`, `Refresh`, `Fail`) panics; so do `Fail(nil)` and a
  blank id. `Fail` is a change, so `Reply(w, r).Fail(err).Else(l)`.
  *(A9)* `Go` after a change or a toast no longer panics: the change is
  dropped and the toast (or `Fail`'s message) goes to the flash.
- `Toast` goes with any change through `ToastSwap`; `Fail`'s danger toast
  takes its place. `Fail` answers 422 whatever the error's kind: the script
  swaps a 422, and the toast is what people read.
- **Events after settling.** `ui-action-done` (closes the action dialog)
  and `ui-table-refresh` (`{"id": "<table>"}`) go in
  `HX-Trigger-After-Settle`: before the swap, the dialog would close and
  reopen (finding 4). `Fail` sends no `ui-action-done`.
- **`Else` and `Go` refuse any address that isn't local**: it must start
  with one `/`, with no backslash, white space or control character, and
  parse with no scheme or host. Anything else returns `ErrNotLocal`, a
  private error for logs, and nothing is written.

```go
var ErrNotLocal = errors.New("components: Reply: the address isn't local") // errors' New: private
```

## 5. What locked components gain

These change locked text, so they are part of this amendment.

- **`ButtonSpec.Unavailable(reason i18n.Ter)`** [T2]: `aria-disabled="true"`,
  focusable, no `href`, `type="button"`, no `form` or `formaction`, and the
  reason as its description (`aria-describedby`) in a Tooltip around it.
  Apart from `Disabled`, which stays native. A blank reason panics.
- **`MenuItemSpec.Unavailable(reason i18n.Ter)`** [T2]: `aria-disabled`,
  inert, named by its label alone (`aria-labelledby`), the reason a second
  line and its description (`aria-describedby`) (finding 8). The menu's
  arrow keys, Home and End stop on unavailable items; they still skip
  `Disabled` ones (§7.3 of visual-interface.md).
- **`MenuSpec.ID(id string)`** [T3]: the trigger's id, keyed, and the
  popover's `<id>-menu`, for a menu whose trigger focus must find again.
- **The shells** (`layout.App`, `layout.Auth`) render `ActionForm` and
  `ActionDialog` beside `ConfirmDialog` and `Toasts` [T9].
- **ADR 0005's list** of what emits `hx-*` gains the table's state
  controls, the action placements and `Reply`'s headers.

## 6. CSS

`theme/css/components.css` gains, under the checks of ADR 0004 (the
spike's `spike.css` is their draft):

`ui-table`, `ui-table-tools`, `ui-table-search`, `ui-table-filters`,
`ui-table-results` (dimmed while `aria-busy`), `ui-table-status`,
`ui-table-chips`, `ui-table-chip`, `ui-table-clear`, `ui-table-scroll`
(sticky headers), `ui-table-grid` (`[data-kind="number"]` aligned to the
end, tabular), `ui-table-sort`, `ui-table-check`, `ui-table-all`,
`ui-table-actions`, `ui-table-empty`, `ui-table-bulk`; `ui-pagination`,
`ui-pagination-range`, `ui-pagination-pages`, `ui-pagination-page`,
`ui-pagination-end`, `ui-pagination-sizes`, `ui-pagination-size`;
`ui-action-buttons`, `ui-action-reason`, `ui-action-dialog`,
`ui-action-dialog-body`, `ui-action-dialog-close`; and
`.ui-button[aria-disabled="true"]`. The filter row's controls align on
one line (finding 17).

*(A8)* `.ui-table-scroll` is `position: relative`, so the visually hidden
texts in its cells (absolutely positioned) take it as their containing
block and never widen the page.

## 7. The ui script

| Part | Does |
|---|---|
| a 422 | swaps (htmx 2 swaps no 4xx by default) |
| `<ui-table>` | names its selection's checkboxes `selected` [T6]; shows the header checkbox once defined; checks the page's rows, with its mixed state; says the count of checked rows in the status (`data-ui-selected`); after its results are swapped (heard on the document, finding 3), copies their `data-ui-status` into the status and puts focus back by id when it was lost; marks the results `aria-busy` while a request runs, cleared on an error too |
| state requests | builds a GET from a table's state control out of the control's own parameters (`data-ui-own`) and the live address's others, dropping empty ones (finding 1); only for requests that target the table or its results (finding 2) |
| the action dialog | §4.3; remembers the opener's row or keyed id, and gives focus back after the answer settles and again after a refresh (finding 5) |
| `ui-action-done`, `ui-table-refresh` | close the dialog a task later; reload a table's results |
| menus | arrow keys over unavailable items (§5) |
| `toast.js` | a toast whose dialog closed before it arrived goes to the top modal or the page (the spike's fix, kept) |

The script assumes htmx 2, which ui doesn't load: the app does, with its
config. That config should set `historyCacheSize: 0`, so Back fetches the
page rather than restoring an open dialog or a stale selection (finding 19)
[T11].

## 8. Keys

`ui.components.table.`: `search`, `apply`, `clear`, `remove`, `select`,
`select_all`, `selected` ("%d selected"), `bulk`, `empty`, `no_match`,
`past_end`, `go_last`, `from`, `to`, `status` ("%d–%d of %d"), `none`.
`ui.components.pagination.`: `label` ("Pages of %s"), `first`,
`previous`, `next`, `last`, `page` ("Page %d"), `size` ("Rows a page"),
`size_n` ("%d rows a page"), `range` ("%d–%d of %d").
`ui.components.actions.`: `label` ("Actions"), `dialog` ("Action").
Declared errors: `tables.ErrRange` (`errors.tables.range`, "the start is
after the end", invalid), a range filter's (§3.4; named in the release's
API review); `Fail` shows the app's own, and `ErrNotLocal` is private.

## 9. Accessibility

As design §8, with the spike's corrections: two pagers are two names; a
row's checkbox and menu trigger are named with the row's label; the
action dialog always has a name; an unavailable item is named by its label
and described by its reason; focus comes back after a refresh through
keyed ids. `CheckNames`, `CheckIDs` and `CheckSelectors` run on every
story; keyboard and focus after a swap are checked in the pane until the
browser test job exists.

## 10. Questions, answered

epistemer took every default on 2026-11-22, so the text above is locked as
written.

| # | Question | Default | Alternative |
|---|---|---|---|
| T1 | The names (S16) | `actions.New`, `Action`, `Bound`, `Item`, `Resolve`, `Gate`; `tables.New`, `Col`, `Table`, `State`, `View`, `Pager`; `components.TableOf`, `Reply` | your names |
| T2 | S21's state for an action that can be read but not done | `Unavailable(reason i18n.Ter)` on `ButtonSpec` and `MenuItemSpec`, `Item.Reason` non-nil; `actionstest.Unavailable` | `ReadOnly`, `Inert`, or `Disabled(reason)` |
| T3 | A menu's keyed id | `MenuSpec.ID(id)` and `ActionMenuSpec.ID(id)`; a table keys its rows' menus | generated ids, and focus returns to the table instead |
| T4 | A refusal in an action's handler | `Reply(w, r).Fail(err)`: 422 and a danger toast | the app answers it by hand |
| T5 | A rejected state answering a results request | `State.Answer(w, r) int`, which retargets and returns the status | the app sets `HX-Retarget` itself |
| T6 | The selection's parameter | `selected`, which shadows no property of a form (finding 10; review 0015 recommends it); prefixed with `<name>.` on a prefixed table | `id`, as the design had it, with every script reading the form's id by attribute |
| T7 | Numbers in cells | `Text`, aligned by `data-kind`; no new renderer | an `elements.Number` renderer |
| T8 | `Page`'s total | `int64`, as `List<Type>` returns it | `int` |
| T9 | Where the action form and dialog are rendered | by the shells, beside the confirmation and the toasts | by the page that has actions |
| T10 | The live-address merge | in the ui script, from `data-ui-own` | the server writes every link and hidden input without the other tables' parameters, and accepts the loss under htmx |
| T11 | htmx's config (`historyCacheSize: 0`) | the app's, documented: ui loads no htmx | `layout.Document` renders a `<meta name="htmx-config">`, which an app's own then can't replace |
