# Server-state components and actions: design draft

| | |
|---|---|
| Status | Draft. epistemer decided S1–S6 in the brainstorm of 2026-11-22; S7–S22 stand on their defaults, open to change until the lock (D7.5) |
| Date | 2026-11-22 |
| Roadmap item | D7.1 |
| Review | [0012](reviews/0012-server-state-design.md): 16 findings, 15 of which changed this text |
| Inputs | [ui-inventory.md](ui-inventory.md), [visual-design.md](visual-design.md) §2 and §5, [visual-interface.md](visual-interface.md) (locked), [forms-design.md](forms-design.md), ADRs [0005](adr/0005-client-behaviour.md), [0008](adr/0008-links.md), [0009](adr/0009-actions.md), [0010](adr/0010-tables.md), models' `listing` and `option` packages (models ADRs 0005 and 0006), handlers' `Route` and `Guard`, the brainstorm of 2026-11-22 |

This draft covers the first round of D7: **links, actions, tables,
filters, pagination and bulk selection**. The remote select, uploads and
list fields are the second round (D8). Nothing here is built. A slice spike
(D7.2) tries the parts marked *(to spike)* before any text is locked.

Examples use the toy shop: an `order_list_view` with the columns `id`,
`status`, `email`, `total`, `customer_id`, `customer_name` and `placed_at`.

## 1. What the record of zp asks for

Read from zp on 2026-11-22, as needs. Counts and shapes only.

**The server table** (10 tables, three files and five closures each):

| Kept | Why |
|---|---|
| One column declaration drives the search box, the filter form, the sortable heads and the query | Declared once |
| The state is a GET form, so the URL holds it | A table can be bookmarked and shared, and Back works |
| A row is typed, and its actions see the whole row | No lookups by string at render |
| Filter kinds follow column kinds: text, number and date ranges, enum, bool, a link to another record | The schema already knows them |
| Search hits are marked in the cells | Cheap, and people rely on it |

| Removed by design | What it let through |
|---|---|
| One mutable struct was the declaration, the request's state, the SQL builder and the render model | The model layer imported the table package (models ADR 0006 undid that) |
| A column's name was written four times and joined by strings: the SQL view, the column, the row's value map, the label | A missing key failed at render; an extra one was ignored |
| Cell values were flattened to text and parsed again in the cell | The loss of type forms removed for fields (its flaw C) |
| A filter was a pair of strings whose meaning depended on the column kind | Every reader had to know the kind |
| A filter value that didn't parse was dropped without a word | A wrong check in the binder had no symptom |
| Sort was a submit button's name, not part of the form's state | The next page or filter lost it |
| Rows arrived over SSE from the same URL | No rows without script, two requests per view, a held connection, an error event nothing rendered, history caching switched off; and the count still ran before the first row |
| Each streamed event rendered under a fresh id scope | Ids in rows and the pager didn't match the page's, and a script patched them afterwards |
| Row checkboxes joined one form id that every bulk action shared | One bulk action per table at most |
| A second, client-side table with its own cells, rows and action lists | Two of everything |

**Actions** (about 100 call sites, 13 components, about 10 patterns):

| Kept | Why |
|---|---|
| A shared vocabulary of verbs with their own keys | One translation of "Delete" |
| An actions area that says so when it is empty | No empty menu |
| A state rule written once and called by both the row and the service | The only pairs that never disagreed |

| Removed by design | What it let through |
|---|---|
| No model: an action was a template call with positional arguments | Nothing could list a page's actions, test them or decide about them |
| A component per placement and mechanism, with the request attributes copied six times | The copies drifted |
| No permission reached a template | Every user saw every button |
| Where the row and the service each wrote their own state condition | Some pairs disagreed: a button offered what the service refused |
| Each handler chose its own answer: a redirect, an empty body, a flash | Six ways to answer a delete |
| A dialog per row per action, a whole form included | Pages carried hundreds of hidden dialogs |
| Two ways for a dialog's form to report errors | Most lost the user's input on a redirect |
| Request targets assembled as strings at the call site | Nothing checked that the target existed |

## 2. Principles

These add to [visual-design.md](visual-design.md) §2.

1. **Logic splits from the view, as forms does.** `ui/actions` and
   `ui/tables` declare, bind and describe. They render nothing.
   `ui/components` renders what they describe, and an app can render it
   itself.
2. **Shown and allowed are one fact.** An action goes to a link, and the
   link's route already has a guard. Whether the action shows is that
   guard's answer. No permission is written a second time for the UI.
3. **Absent or explained.** Without the permission, an action is absent.
   When the record's state forbids it, it is disabled and says why, and the
   rule behind it is the function the handler calls too.
4. **A column is declared once, in SQL.** models generates each view's
   columns. A table picks from them. No column name is typed as a string
   in an app.
5. **The URL is the state.** Search, filters, sort and page are query
   parameters, all of them, so none is lost when another changes.
6. **Works without script, better with htmx.** A table is a GET form and
   links. An action is a link or a form post. htmx swaps a region where a
   plain browser would load a page; the handler is the same.
7. **One handler, one response.** No second request fetches the rows.
8. **User input never panics; a programmer's mistake does, at
   declaration** (forms' I10). A filter value that doesn't parse is an
   error the user sees.
9. **Request targets are keyed ids**, so `uitest.CheckIDs` can check them.

## 3. Packages

| Package | Layer | Holds | Imports |
|---|---|---|---|
| `ui` (kernel) | kernel | gains `Link` and `URL` (§4) | unchanged |
| `ui/actions` | logic | actions, the gate, items, results' vocabulary (§5) | the kernel, i18n, errors, templ (for the types of an icon and of attributes; it renders nothing) |
| `ui/actions/actionstest` | test | gates and assertions for tests | actions, `testing` |
| `ui/tables` | logic | tables, their state, filters, pages (§6) | the kernel, forms, actions, i18n, errors, models' `listing` and `option`, `net/http`, `net/url` |
| `ui/tables/tablestest` | test | binding from a query string, assertions | tables, `testing` |
| `ui/components` | components | gains `TableOf`, `Pagination`, `ActionButtons`, `ActionMenu`, `ActionForm`, `ActionDialog`, `Reply` (§7) | gains actions, tables, inputs, `net/http` |

Neither logic package imports handlers or auth. They read links through
the kernel's interface and ask a gate the app sets (§5.3).

## 4. Links (S5; ADR 0008)

handlers' generated URL builders will return a link value, the route
together with its built URL, instead of a string. So ui takes a link
wherever it takes an href today.

```go
package ui

// Link is anything with a URL: a route link, or URL's adapter.
type Link = interface{ URL() string }

// URL makes a Link of a literal or external address.
func URL(s string) Link
```

- **An alias, not a named interface**, so the gate function handlers
  returns, declared over `interface{ URL() string }`, is assignable where
  ui wants a function over `Link`, with neither package importing the
  other (§5.3). A function literal an app passes to ui states `ui.Link`
  as its result and returns a route link from it.
- **Six locked signatures change** (amendment A5, §11): `ButtonSpec.Href`,
  `elements.Link`, `MenuItemSpec.Href`, `components.Tab`,
  `layout.NavItem`, `layout.Crumb.Href`. Whether the two that take an
  image's `src` follow is S8.
- **Every link is sanitised at render**, through `templ.URL`: `http`,
  `https`, `mailto`, `tel` and relative addresses pass. So a `Link` built
  over user input is no more dangerous than a string was.
- **No address** is a nil link, one whose URL is empty, or one templ
  refuses. It means what an empty `templ.SafeURL` means today: no `href`
  on a button, `Empty` from the value renderer, the current page in a
  crumb.
- **What else a link may offer.** ui asserts one optional method,
  `Method() string`; an action to a link without it is a GET. A type that
  wraps a link must pass `Method` on. The route itself never reaches ui:
  the gate reads it, not ui.
- **A literal is not a route.** `ui.URL("/orders/7/refund")` has no route
  behind it, so nothing can be asked about it: it is always shown, and it
  is a GET. An app's own routes are named through their builders.

What an app writes changes from
`Href(templ.URL(adminroutes.Order(o.ID)))` to
`Href(adminroutes.Order(o.ID))`, and from `Href("/docs")` to
`Href(ui.URL("/docs"))`.

## 5. Actions (S1; ADR 0009)

### 5.1 Declare, bind, resolve, describe

```go
// Declared once: what it looks like and what it asks.
var (
	View   = actions.New(L10nView).Icon(icons.Eye())
	Refund = actions.New(L10nRefund).Icon(icons.Undo()).Danger().
		Confirm(L10nRefundQuestion)
	Reassign = actions.New(L10nReassign).Dialog()
)

// Bound to a subject and resolved against the request.
func orderActions(ctx context.Context, o models.OrderListView) []actions.Item {
	return actions.Resolve(ctx,
		View.To(adminroutes.Order(o.ID)),
		Refund.To(adminroutes.OrderRefund(o.ID)).Unless(shop.CanRefund(o)),
		Reassign.To(adminroutes.OrderReassign(o.ID)),
	)
}
```

```templ
@components.ActionMenu(L10nOrderActions, orderActions(ctx, o))
```

| Step | Type | What it is |
|---|---|---|
| Declare | `actions.Action` | An immutable value: label, icon, danger or primary, a confirmation question, whether it opens the action dialog, extra attributes. Builders return copies, as the visual specs do |
| Bind | `actions.Bound` | The action with a link, and optionally `Unless(err)` or `If(ok)` |
| Resolve | `actions.Resolve(ctx, bound...)` | Asks the request's gate about each link and drops what it refuses or what `If(false)` excluded |
| Describe | `actions.Item` | What a renderer reads, as `forms.Control` is for a field: `Label`, `Icon`, `Kind`, `Method`, `URL`, `Question`, `Danger`, `Primary`, `Disabled`, `Reason`, `Attrs` |

```go
func New(label i18n.Ter) Action
func (a Action) Icon(c templ.Component) Action
func (a Action) Danger() Action
func (a Action) Primary() Action
func (a Action) Confirm(question i18n.Ter) Action
func (a Action) Dialog() Action
func (a Action) Attrs(attrs templ.Attributes) Action
func (a Action) To(l ui.Link) Bound

func (b Bound) Unless(err error) Bound // disabled, with err as the reason, when err != nil
func (b Bound) If(ok bool) Bound       // absent when false

func Resolve(ctx context.Context, bound ...Bound) []Item

type Kind int
const (
	Navigate Kind = iota // a GET link
	Request              // any other method
	Open                 // a GET whose answer shows in the action dialog
)
```

- A blank label, or `Dialog` with `Confirm`, panics where the action is
  declared.
- An action bound to no address (§4) is absent, as with `If(false)`.
  Binding happens per row in a request, so it never panics.
- `Resolve` keeps the order given. An empty result is an empty slice, and
  every placement renders nothing for it.

### 5.2 The two kinds of "no"

| Why not | What the user gets | Declared as |
|---|---|---|
| The request's guard refuses the link's route | Nothing: the action is absent | Nothing: it follows from the link |
| The record's state forbids it | The action, disabled, with the reason | `Unless(shop.CanRefund(o))` |
| The page chooses not to offer it | Nothing | `If(o.Status != models.OrderStatusDraft)` |

`shop.CanRefund` is an ordinary function returning an errors declaration
or nil. The refund handler calls it before it acts, so the button and the
handler can't disagree, and the declaration's text (`errors.T`) is the
reason a disabled item shows. This is forms' `rule.New` idea for records.

A disabled item is never the only security: the handler runs the rule,
and the route's guard runs the permission. The UI only stops offering
what would be refused.

### 5.3 The gate

```go
// Gate is set once per request and read by Resolve.
var Gate *ui.Provider[func(ui.Link) bool]

func AllowAll(ui.Link) bool // admits every link: for an app with no permissions
func Allowed(ctx context.Context, l ui.Link) bool
```

- The app sets it in a middleware. With handlers, from the guard that
  protects the routes:

  ```go
  admit, err := guard.Admit(r) // one memoised check for the whole request
  if err != nil {
  	fail(w, r, err) // the session or the permissions couldn't be read
  	return
  }
  ctx := actions.Gate.With(r.Context(), handlers.Gate(admit))
  ```

  `handlers.Gate` is part of the handlers amendment (§10). For a route
  link it answers `admit` for the link's route. A link that is no route
  link has nothing to ask about, and is admitted (§4).
- **No gate set is a programmer's mistake**: `Resolve` panics, naming
  `actions.Gate` (S9). An app without permissions sets `actions.AllowAll`.
- The gate is the route's guard, so it also carries what the route
  declares beyond a permission: a public route is admitted, a session-only
  route needs a session.
- **The gate decides what is shown, never what is allowed.** The route's
  guard decides that, on every request, whatever a page showed.
- A freshness requirement (`auth.Within`) isn't known to `Admit`; the
  action shows, and the route answers with auth's stale error, which the
  app turns into a step-up. Out of this design.
- Cost: `Admit` memoises per permission, so a page of 50 rows with four
  actions each is 200 map lookups after the first few store reads.
- **The same gate serves anything with a link.** `actions.Allowed` lets a
  page drop a nav item or a tab the user can't open. Doing that inside
  `NavItem` and `Link` themselves is S10; the default is no, so the
  locked elements stay free of hidden behaviour.

### 5.4 How an action runs

Taken from the item's kind, never chosen by the caller:

| Kind | Without script | With htmx |
|---|---|---|
| `Navigate` | `<a href>` | the same; a boosted page boosts it |
| `Request` | a submit button of the page's action form, with `formaction` the URL | the button also carries the request attribute for its method and swaps nothing by default; the answer says what changes (§5.6) |
| `Open` | `<a href>` to a page holding the form | the answer loads into the one action dialog, which the ui script opens when content arrives in it |

- **The action form** is one empty `<form id="ui-action-form"
  method="post" hidden>` the shells render once, beside the confirmation
  dialog. Every `Request` control names it with the `form` attribute, so
  a button in a menu, a card or another form posts to its own
  `formaction` and nothing nests.
- `PUT`, `PATCH` and `DELETE` have no form in HTML, so an action with one
  of them needs htmx. A page that must work without script uses `POST`
  routes.
- **Cross-site requests** are refused by auth's `Load`, which every route
  runs through, from the browser's own headers. So no form here carries a
  token.
- **Confirmation** is the locked `components.Confirm` attribute, set from
  the action's question. Nothing new.
- **The action dialog** is one `<dialog id="ui-action-dialog">` the
  shells render too. A form action's body is fetched when it opens, so no
  row carries a form.
  - The form inside it targets the dialog's body. It answers `422` with
    itself and its errors, the user's input kept, and the dialog stays
    open.
  - The ui script opens the dialog when content arrives in its body, and
    closes it only when an answer says so (§5.6). The first load and a
    failed submission both leave it open.
  - The ui script lets a `422` swap. htmx 2 swaps no `4xx` by default.
- **One helper** tells a handler whether the request came from the
  dialog: `components.InActionDialog(r)`. Both the `Open` route and the
  form's `422` branch use it to render the form alone, or a whole page
  for a browser without script.

*(to spike: the dialog's opening and closing, the 422, and htmx's
handling of a submit button that names its form by attribute)*

### 5.5 Placements

In `ui/components`, all reading `[]actions.Item`:

| Placement | Renders | Used for |
|---|---|---|
| `ActionButtons(items)` | a row of Buttons: the primary one filled, a danger one in the danger tone, the rest outlined | a page header, a card's footer, a detail page |
| `ActionMenu(label, items)` | a Menu with an icon trigger and one MenuItem each; danger items last, after a separator | a table row |
| the bulk bar (§6.7) | submit buttons of the selection's form | a table |

- Each renders nothing at all for no items.
- **A disabled item can do nothing and can be read.** It renders with no
  `href`, no `type="submit"`, no `form` or `formaction`, no question and
  no request attribute, so it is inert without script too. It stays
  focusable, carries `aria-disabled="true"`, and its reason is its
  description: a Tooltip on a button, a second line in a menu item.
- That asks two things of the locked elements (S21): a Button that is
  `aria-disabled` and focusable instead of natively `disabled`, and a
  menu whose arrow keys stop on disabled items, as the APG recommends,
  where §7.3 now skips them.
- An app with another placement reads the items itself; nothing in a
  placement is private.

### 5.6 What an action answers

A handler answers through one builder instead of choosing a mechanism:

```go
return components.Reply(w, r).
	Toast(L10nRefunded).
	Replace(Orders.RowID(o.ID), OrdersView.Row(ctx, o)).
	Else(adminroutes.Orders()) // where a request without htmx goes
```

One change, an optional toast, and an ending:

| Change | With htmx | Without |
|---|---|---|
| `Replace(id, c)` | `c` replaces the element with that keyed id | redirect to `Else` |
| `Remove(id)` | the element with that id is removed | redirect to `Else` |
| `Refresh(table)` | the table reloads its results at its current URL | redirect to `Else` |
| none | nothing changes on the page | redirect to `Else` |

| Ending | Does |
|---|---|
| `Else(link) error` | sends the answer; a request without htmx is redirected to `link` with `303` |
| `Go(link) error` | sends the browser to `link` either way, instead of a change |

- **`Toast(text)`** goes with any change through the locked `ToastSwap`.
  With `Go`, and without htmx, there is no page left to show it on: ui
  keeps no flash storage (S11), so the app's own carries the message to
  the next page, as §7.5 of the lock says.
- **One change per answer.** The answer retargets itself at the id, so a
  table row travels as the response's own content, which htmx 2 parses as
  a row. Out-of-band rows would each need a `<template>` wrapper. A
  handler that changed more than one row refreshes the table.
- **Any successful answer closes the action dialog**, by an event the
  script listens for.
- **`Else` and `Go` take a local address**, one that starts with a single
  `/`. Anything else is an error, so a `next` parameter can't send a user
  to another site.
- An action in a menu, a dialog or the bulk bar needs no target of its
  own, and `CheckIDs` can check every id an answer names.

*(to spike: retargeting a row, removal, the refresh event, the close
event)*

### 5.7 actionstest

```go
ctx := actionstest.Context(t, actionstest.Refuse(adminroutes.OrderRefund(o.ID)))
items := orderActions(ctx, o)
actionstest.Assert(t, items, View, Reassign) // Refund is absent
```

A test states which links the gate refuses and reads items, with no HTML.
`actionstest.Disabled(items, Refund)` returns the reason. A context from
`actionstest.Context` with no option admits everything.

## 6. Tables (S2–S4; ADR 0010)

### 6.1 Columns come from models

models generates, for every relation whose name ends in `_view`, a set of
columns beside its listing method (the models amendment, §10):

```go
c := models.OrderListViewColumns // c.ID, c.Status, c.Email, c.Total, …
```

A column is what the schema knows, and nothing about a page:

| What | From |
|---|---|
| its SQL name | the view |
| its label | the label models already generates for the column |
| its kind: text, integer, number, bool, date, time, enum | the column's SQL type |
| an enum's options | the enum type's `All<Type>Options` |
| a getter from a row to a plain value, null included | generated, unwrapping the driver's null types, so ui never imports the driver |

A column can be built by hand, for a test or for rows that don't come
from sqlc.

### 6.2 Declaring a table

```go
var Orders = tables.New("orders", c.ID,
	tables.Col(c.Status).Filter(),
	tables.Col(c.Email).Searchable(),
	tables.Col(c.CustomerName).Searchable().Link(func(o models.OrderListView) ui.Link {
		return adminroutes.Customer(o.CustomerID)
	}),
	tables.Col(c.Total).Sortable().Amount(shop.Money),
	tables.Col(c.PlacedAt).Sortable().Filter(),
).
	Label(L10nOrders).
	RowLabel(c.Email).
	Sort(c.PlacedAt, tables.Desc).
	Actions(orderActions). // func(ctx, row) []actions.Item, as in §5.1
	Bulk(bulkOrderActions) // func(ctx) []actions.Item
```

- **The key comes first.** `tables.New(name, key, columns...)`: the key
  column identifies a row. It gives rows their ids, the selection its
  values, and every sort its last term in the sort's own direction, so
  paging is stable (models asks for a sort that ends on a unique column).
  It is shown only if it is also listed.
- `tables.New[R]` takes its row type from the columns, so nobody writes
  it. The table is declared once, in one expression at package level, and
  only read afterwards; a request gets a state (§6.3), not a copy.
- **Everything a page says about a column is said in one place**, on
  `tables.Col` (S22):

  | Option | Meaning |
  |---|---|
  | `Searchable()` | the search box looks in it |
  | `Sortable()` | its head sorts |
  | `Filter()` | it gets a filter, by its kind (§6.5) |
  | `Hidden()` | kept for a filter or a sort, not shown |
  | `Link(func(R) ui.Link)` | its text links to another record |
  | `Amount(func(R) i18n.Ter)` | it is money: the function formats it, since ui imports no decimal library and a currency is the app's |
  | `Image(func(R) i18n.Ter)` | its text is an image's address, with that alternative text |

- `RowLabel` names the column whose text names a row to assistive
  technology (§8). `Sort` is the order when the URL names none.
  `Sizes(20, 50, 100)` sets the page sizes and `Unpaged()` shows every
  row. `Prefixed()` puts the table's name before its parameters (§6.4).
- **Mistakes panic where the table is declared** when one call can see
  them: a name that isn't `[a-z][a-z0-9-]*`, no columns, a column twice,
  an option on a kind it doesn't fit, a filter on a column whose name
  forms can't take as a field's or that is a reserved parameter (§6.4).
  What needs the whole declaration (a row label or a default sort that
  isn't a column of the table) panics on first use, and
  `tablestest.Check` finds it in a test.

### 6.3 A request

```go
func (h *H) Orders(w http.ResponseWriter, r *http.Request) error {
	st := Orders.Bind(r)
	if !st.Valid() {
		return h.page(w, r, http.StatusUnprocessableEntity, ordersPage(st.Rejected()))
	}
	rows, total, err := h.q.ListOrderListView(r.Context(), st)
	if err != nil {
		return err
	}
	return h.page(w, r, http.StatusOK, ordersPage(st.Page(rows, total)))
}
```

| Step | What happens |
|---|---|
| `Bind(r)` | reads the query string into a `*tables.State[R]`: the search text, the filters (a form, §6.5), the sort, the page and its size |
| `Valid()` | false when a filter's value doesn't parse |
| `Rejected()` | the view of a state that isn't valid: the filter form with its errors, and neither rows, an empty state nor a pager. As a form that didn't validate does nothing (S12). The status is `422`, which the script lets htmx swap |
| the query | the state is a `listing.Source`; models builds the SQL and checks every column name |
| `Page(rows, total)` | a `tables.View[R]`: what a renderer reads |

`tables.View[R]` describes, and holds no request:

| Part | Holds |
|---|---|
| the table | its ids (view, results, state form, selection form, caption), its label, the current URL |
| columns | label, kind, whether sorted and which way, the link that sorts by it next |
| rows | the row, its key and id, its label, its cells as values, its action items |
| search | the control's name and current text |
| filters | the filter form's controls, and one chip per active filter with the link that removes it |
| pager | the range shown, the total, the links to the first, previous, next and last pages and a window of numbered ones, the sizes on offer |
| bulk | the selection form's id and the bulk action items |
| empty | no rows at all; none for these filters (with the link that clears them); or a page past the last one (with the link to the last) |
| status | the one sentence an assistive technology hears after a change |

A handler that answers an action needs one row again, outside any page:
`Orders.Row(ctx, row)` describes it and `Orders.RowID(key)` is its id.

### 6.4 The URL

| Parameter | Meaning | Junk |
|---|---|---|
| `q` | the search text | trimmed; cut at 200 runes |
| `sort` | a column's name, with `-` before it for descending | an unknown or unsortable column falls back to the table's own order |
| `page` | the page, from 1 | not a number, or below 1: page 1 |
| `size` | rows a page | not on the table's list: the first |
| a column's name | an equality filter | an error on the filter (§6.5) |
| `<column>.from`, `<column>.to` | a range filter's ends, both inclusive | the same |

- Sort, page and size are written by links ui renders, so junk in them is
  normalised without a word. Filters are typed by people, so junk in them
  is reported.
- A filter can't sit on a column named `q`, `sort`, `page` or `size`
  (S13).
- **Parameters that aren't the table's are kept.** Every link the view
  offers is the current URL with one thing changed, and the state form
  carries the rest of the query as hidden inputs, because a GET form
  replaces the whole query. So a second table, or a page's own
  parameter, survives a search.
- **Two tables on a page** each declare `Prefixed()`, and their
  parameters become `orders.q`, `orders.sort`, `orders.status`. Ids come
  from the name, so a page shows a table once.
- A change to search, filters, sort or size goes back to page 1.

### 6.5 Filters are a form

A table's filters are one `forms.Form`, declared from the filterable
columns and bound from the query string with `forms.FromValues`:

| Column kind | Field | Query |
|---|---|---|
| text | `Text` | `=` |
| enum | `Enum`, with the column's options | `=` |
| bool | `Bool`, three states | `=` |
| integer, number | a group of two, `from` and `to` | `>=`, `<=` |
| date | a group of two `Date` fields | `>=`, `<=` |
| time | a group of two `Date` fields | `>=` the first day's start, `<` the day after the last |

- A field's label is the column's. Rendering is `inputs.Field` on each
  control; nothing about filters is a new input.
- A range whose `from` is after its `to` is an error on the group.
- A day's start is UTC until neozp carries a time zone per request (the
  backlog's item for forms' `Date`).
- Values reach models typed (an enum as its Go type, a date as a
  `time.Time`), which is what `listing.Filter` asks so that a wrong value
  is a 422 from ui and never a 500 from the database.
- Not in the first cut, because models' `listing` has no operator for
  them: several values of one enum (`IN`), "is empty" (`IS NULL`), "text
  contains". Each is a models amendment first.
- A filter on a column that links to another record needs the remote
  select and models' `Labeler`. It comes with D8; until then such a
  column takes no filter.

### 6.6 Rendering

```go
// Declared once, so a page and an action's answer render a row alike.
var OrdersView = components.TableOf(Orders).Cell(c.Status, statusCell)
```

```templ
@OrdersView.Page(view)
```

```html
<ui-table id="orders" class="ui-table">
  <p class="ui-table-status" role="status"></p>
  <form id="orders-state" method="get" class="ui-table-tools">
    search, filters, size; hidden inputs for the query's other parameters
  </form>
  <div id="orders-results" class="ui-table-results" data-ui-status="21–40 of 312" data-ui-url="/orders?…">
    <input type="hidden" name="sort" value="-placed_at" form="orders-state">
    chips for the active filters
    <div class="ui-table-scroll" role="region" tabindex="0" aria-labelledby="orders-caption">
      <table>
        <caption id="orders-caption">Orders</caption>
        <thead>…<th scope="col" aria-sort="descending"><a id="orders-sort-placed_at" href="?sort=placed_at">Placed</a></th>…</thead>
        <tbody>…<tr id="orders-row-1042">…</tr>…</tbody>
      </table>
    </div>
    the bulk bar
    <nav class="ui-pagination" aria-label="Pages">…</nav>
  </div>
</ui-table>
```

- **Two parts.** The state form holds what a person types in; the
  results hold what changes when they do. A search swaps only the
  results, so the box being typed in is never replaced (§6.8).
- **The sort's hidden input sits in the results** and joins the state
  form by the `form` attribute, so it follows a swap.
- **Cells** render through the locked value renderers (`Bool`, `Date`,
  `DateTime`, `Enum`, `Link`, `Image`, `Amount`, `Empty`, and `Text`),
  chosen by the column's kind and its options. A null is `Empty`.
  Integers and numbers align to the end with tabular numerals.
  `Cell(column, func(R) templ.Component)` on the table's view replaces
  one column's cell, which is how an enum gets toned badges.
- **Search hits** are marked with `<mark>` in searchable text cells. The
  table writes those cells itself, with `Text`'s class, since `Text`
  takes a `Ter` and escapes it.
- **Sorting** is a link in the `<th>`, since it goes to another URL.
  `aria-sort` sits on the sorted column's `<th>` alone; the direction
  icon is decorative.
- **The pager** is `components.Pagination(view.Pager())`, a component of
  its own: a `<nav>` with a name, links, and `aria-current="page"`.
- **The scroll region** has `role="region"`, a name from the caption and
  `tabindex="0"`, so a keyboard can scroll a wide table. Headers stick. A
  stacked layout for narrow containers is S14.
- **Announcements.** The status element is outside the results and
  starts empty. After a swap, `<ui-table>` copies the results'
  `data-ui-status` into it. The table itself is never a live region.
- **Ids** are keyed from the table's name, so they are the same in the
  page and in any later answer: `orders`, `orders-state`,
  `orders-results`, `orders-select`, `orders-caption`,
  `orders-row-<key>`, and one per sort and page link, so focus can be put
  back on the control that was used.
- **Empty states**: no rows at all; none for these filters, with a link
  that clears them; a page past the last. ui's texts for these, the pager
  and the status are keys under `ui.components.table.` and
  `ui.components.pagination.`, as the lock's §11 names its keys.

### 6.7 Selection and bulk actions

- A table with bulk actions gets a checkbox column. Each checkbox is
  `name="id"`, its value the row's key, and belongs to the selection
  form, `<form id="orders-select" method="post">`, through the `form`
  attribute, so the selection form and the state form never nest.
- **A bulk action is a submit button of that form** with its own
  `formaction`. Any number of them work, with no script: the browser
  posts the checked ids to the button's URL. Its question, if it has one,
  is the locked confirmation.
- **A bulk action that needs more input** is an `Open` action. Its button
  submits the selection form with `formmethod="get"`, so a browser
  without script reaches a page holding the form, the ids in its address.
  With htmx the same request, with the selection included, loads the
  action dialog. Either way the form posts the ids on with its fields.
- **The bar follows the table in the document**, so it comes after the
  checkboxes in the tab order, and shows while something is checked, by
  CSS: `:has(input[form="orders-select"]:checked)`.
- `<ui-table>`, a small custom element in the ui script, does what HTML
  can't: the header checkbox that checks the page's rows (and its mixed
  state), the count of checked rows, said in the status element, and the
  status and focus after a swap. The header checkbox is hidden until the
  element is defined. Without the script the row checkboxes and the
  buttons still work.
- The selection is the page's. Changing page, sort or filter clears it.
  "All 312 matching" is not in the first cut (S15).
- Bulk actions resolve through the gate like any other. State rules are
  per record, so the handler checks each id and answers all or nothing:
  with htmx a toast and the selection kept; without, the app's own flash
  on the page it redirects to. The UI doesn't pre-check a selection.

### 6.8 Transport (S4)

- **Without script** the state form submits, and sort and page links
  navigate. That is the whole table.
- **With htmx**, the controls that change the table's state carry the
  request attributes, and nothing else does. htmx attributes inherit, so
  they are never set on the table or its results: a link in a cell and a
  row's actions must stay what they are.

  | Control | Swaps |
  |---|---|
  | the search box, on input after a pause | the results |
  | a sort link, a page link, a filter chip, a "clear" link | the results |
  | the state form, submitted | the whole table, so a filter's error shows beside its field |

  Each takes its target from the answer by id, moves the address bar, and
  replaces a request still in flight for the same table.
- **Focus.** After a swap of the results, `<ui-table>` puts focus back on
  the control with the id that had it. The search box is never swapped
  while it is typed in.
- **The handler doesn't change.** It may render the table alone when the
  request targets it (`tables.Targeted(r, Orders)`), which saves the
  shell; a whole page is also a right answer.
- While a request runs the results are `aria-busy` and dimmed.
- **No SSE.** Pages hold 20 to 100 rows; one response carries them. The
  count is models' business, which already skips it when the page tells
  the total. A table that updates as events happen is another need, for
  when a page has it.
- ui emits `hx-*` here because the table is a component that makes
  requests (ADR 0005). It targets htmx 2.

*(to spike: every row of the table above, Back, and focus)*

### 6.9 tablestest

```go
st := tablestest.Bind(t, Orders, "q=ann&status=paid&sort=-placed_at&page=2")
tablestest.Query(st)           // the listing.Query, to compare
tablestest.Errors(st)          // ["placed_at.from: errors.forms.invalid"]
tablestest.Check(t, Orders)    // every column labelled, the row label and the default sort are its columns
```

`tablestest.Bind` sets a gate that admits everything, so a table with
actions binds in a test. Rendering is tested as every component is:
stories, goldens, `CheckIDs`, `CheckNames`, `CheckSelectors`.

## 7. What `ui/components` gains

| Component | Reads | Notes |
|---|---|---|
| `TableOf(table)` | a `*tables.Table[R]` | a view declared once: `.Cell(column, func(R) templ.Component)`, `.Attrs`; `.Page(view)` and `.Row(ctx, row)` render |
| `Pagination(pager)` | `tables.Pager` | usable without a table |
| `ActionButtons(items)`, `ActionMenu(label, items)` | `[]actions.Item` | §5.5 |
| `ActionForm()`, `ActionDialog()` | — | once per page, in the shells, beside `ConfirmDialog` |
| `InActionDialog(r)` | the request | §5.4 |
| `Reply(w, r)` | — | §5.6 |

CSS: `table.css`, `pagination.css`, `actions.css` in `theme/css/`, under
the checks of ADR 0004. Script: `<ui-table>`, the action dialog's opening
and closing, and letting a 422 swap, inside the 12 kB budget.

## 8. Accessibility

| Component | Contract |
|---|---|
| Table | a `<caption>`; `<th scope="col">`; `aria-sort` on the sorted column only; a named, focusable scroll region; one status element, no live region on the table |
| Rows | a row's checkbox and its menu's trigger are each named by their own label and the row's label cell together, so no two rows' controls share a name. The row label is the `RowLabel` column, or the first shown text column |
| Selection | the header checkbox has a name of its own and is hidden until the script can run it; the count of checked rows is said in the status element; the bulk bar follows the table in the tab order |
| Pagination | a named `<nav>`; `aria-current="page"`; previous and next named in words, not arrows; a disabled end is not a link |
| Filters | forms' fields through `inputs.Field`: labels, errors, groups as fieldsets |
| Actions | a disabled action is focusable, inert, and says why (§5.5); an icon-only trigger has a name; a danger action is distinguished by more than colour (its place and its confirmation) |
| Action dialog | the locked Dialog's contract; focus moves into the loaded form and back to the trigger on close |

`CheckNames` covers the names. Keyboard and focus after a swap wait for
the browser test job, and the spike checks them by hand until then.

## 9. Decisions

Decided by epistemer on 2026-11-22:

| # | Question | Decision |
|---|---|---|
| S1 | Where an action's permission comes from | From its route: the action is absent when the request's guard doesn't admit its link |
| S2 | How a table gets from a row to its cells | Column sets generated by models per `_view` relation |
| S3 | A generated column's shape | name, label, a kind from the SQL type, enum options, and a getter returning a plain value with nulls unwrapped |
| S4 | Transport | one handler, one response; a GET form and links; htmx swaps the region; no SSE for rows |
| S5 | Route links | handlers' builders return a link; every ui element or helper that takes an href takes `interface{ URL() string }` |
| S6 | Scope and the tag | this round is tables, pagination, filters, bulk and actions; v0.2.0 is tagged after this design is reviewed |

Standing on their defaults:

| # | Question | Default | Alternative |
|---|---|---|---|
| S7 | Does ui import handlers? | No. ui owns `Link` and the gate's type; handlers' link satisfies one and its `Gate` helper the other | ui imports handlers and reads `Route` and `Guard` itself: one less helper, and handlers and genx enter ui's module graph |
| S8 | Do `AvatarSpec.Src` and `elements.Image` take a `Link` too? | Yes: a file served by a route gets its address from the same builders | keep `templ.SafeURL` for `src`, and convert at each call |
| S9 | `Resolve` with no gate set | panics; an app without permissions sets `actions.AllowAll` | admit everything: one less line, and a forgotten middleware shows every action to everyone |
| S10 | Do `NavItem`, `Tab` and `elements.Link` hide or unlink themselves when the gate refuses? | No: a page asks `actions.Allowed` | yes: less to write, and locked elements gain behaviour that depends on the request |
| S11 | A toast after a redirect | ui keeps no flash storage; the app's own renders a `Toast` on the next page, as locked | ui stores flashes: it would need a session or a cookie of its own |
| S12 | A filter value that doesn't parse | the errors and no rows, status 422 | the rows without that filter, and the error beside it |
| S13 | Filter parameter names | the column's name; a filter on a column named like a reserved parameter panics at declaration | every filter under a prefix (`f.status`): no collisions, longer URLs |
| S14 | Narrow containers | a scroll region with sticky headers | a stacked layout by container query, as a variant |
| S15 | Selecting every row that matches | not in the first cut | a second mode that posts the query instead of ids, for handlers that can act on a query |
| S16 | Names | `ui.Link`, `ui.URL`; `ui/actions` with `Action`, `Bound`, `Item`, `Resolve`, `Gate`; `ui/tables` with `Table`, `Col`, `State`, `View`, `Pager`; `components.TableOf`, `Reply` | epistemer's choice at the lock |
| S17 | Where the placements and `Reply` live | `ui/components` | a package of their own (`ui/components/tableview`) |
| S18 | Row actions' placement in a table | one menu per row | the first action as a button beside the menu |
| S19 | A second level in an action menu | none: it becomes an `Open` action or a page | submenus in `Menu` |
| S20 | A client-side table | none: a short list is `Unpaged()` | a second component |
| S21 | A disabled action that can be reached and read | amend the lock: a Button state that is `aria-disabled` and focusable, and a menu whose arrow keys stop on disabled items (§7.3 skips them today) | keep both as locked; a disabled action's reason is then out of a keyboard's reach |
| S22 | Where a column's page options are set | on `tables.Col(column)`, all of them; models' column holds only what the schema knows | `Searchable`, `Sortable` and `Filter` on models' column (`c.Email.Searchable()`), as the brainstorm's sketch had it: shorter for a plain column, but the options then live in two packages and models carries what a page decides |

## 10. What other packages are asked for

Each is that package's own amendment, drafted after the spike and brought
to epistemer.

**handlers** (breaking, so its v0.2.0):
- generated builders return `handlers.Link` instead of `string`;
- `Link` has `URL() string`, `Method() string` and `Route() Route`, and a
  way to add a query;
- `handlers.Gate(admit func(Route) bool) func(interface{ URL() string }) bool`,
  which asks `admit` about a route link and admits any other link.

**models** (its v0.4.0):
- a column type and a plain value type (S3), holding only what the schema
  knows;
- a generated column set per `_view` relation;
- later, each its own amendment: `IN`, `IS NULL` and "contains" in
  `listing`.

**ui's own accepted and locked text:**
- amendment A5 (§11), now;
- ADR 0005's list of what emits `hx-*` (the server table, the remote
  select, the command palette, `ToastSwap`) gains action controls and
  `Reply`; its "streamed rows" goes (ADR 0010);
- the contract of actions and tables, and what S21 asks of Button and
  Menu, as a later amendment after the spike (D7.5).

## 11. Amendment A5, proposed: links

To be locked by epistemer, and built before v0.2.0 is tagged.

**interface.md, the kernel** gains:

```go
type Link = interface{ URL() string }
func URL(s string) Link
```

`URL("")` is a link with no address. The kernel's imports don't change.

**visual-interface.md** changes these signatures, and nothing they render:

| Section | Now | Becomes |
|---|---|---|
| §5.1 | `func (b ButtonSpec) Href(u templ.SafeURL) ButtonSpec` | `func (b ButtonSpec) Href(l ui.Link) ButtonSpec` |
| §5.4 | `func Link(href templ.SafeURL, text i18n.Ter) templ.Component` | `func Link(to ui.Link, text i18n.Ter) templ.Component` |
| §7.3 | `func (m MenuItemSpec) Href(u templ.SafeURL) MenuItemSpec` | `func (m MenuItemSpec) Href(l ui.Link) MenuItemSpec` |
| §7.4 | `func Tab(href templ.SafeURL) TabSpec` | `func Tab(to ui.Link) TabSpec` |
| §8.2 | `func NavItem(href templ.SafeURL) NavItemSpec` | `func NavItem(to ui.Link) NavItemSpec` |
| §8.4 | `Crumb.Href templ.SafeURL` | `Crumb.Href ui.Link` |
| §5.3, if S8 holds | `func (a AvatarSpec) Src(u templ.SafeURL) AvatarSpec` | `func (a AvatarSpec) Src(l ui.Link) AvatarSpec` |
| §5.4, if S8 holds | `func Image(src templ.SafeURL, alt i18n.Ter) templ.Component` | `func Image(src ui.Link, alt i18n.Ter) templ.Component` |

and adds to §2:

- An address is a `ui.Link`. Components render it through `templ.URL`,
  which refuses schemes other than `http`, `https`, `mailto` and `tel`.
- A nil link, one whose `URL()` is empty, or one `templ.URL` refuses, is
  no address: what an empty `templ.SafeURL` meant.

Goldens don't change: the same addresses render the same markup.

## 12. The slice spike (D7.2)

In `internal/spike`, in the coding session, with the proposed column,
value and link shapes declared locally so no other repository changes
first. The toy shop's orders:

- a table with search, an enum filter and a date range, sort, paging and
  two sizes, working with script off;
- the same with htmx: the results swap on search without touching the
  box, the whole-table swap on submit with a filter's error at 422, the
  address bar, Back, focus put back by id, the status text, and a link in
  a cell that still navigates;
- a second table on the page, to see that neither loses its state;
- row actions: a link, a request with a confirmation gated by a state
  rule (disabled, reachable, inert without script), an `Open` action
  whose form fails once with a 422 and then succeeds, one action the gate
  refuses;
- the action form: a request item in a menu and in a card posting with
  script off;
- two bulk actions, one of them an `Open` action;
- `Reply`: replace a row, remove a row, refresh the table, a toast, the
  dialog closing;
- goldens, `CheckIDs`, `CheckNames`, `CheckSelectors`, and the two test
  kits.

Its findings amend this draft, as the D4 spike amended visual-design.md.

## 13. Out of this round

- The remote select (`<ui-combobox>`), uploads and list fields
  (`<ui-repeat>`): D8. A filter on a link column waits for the first.
- Live tables.
- Column choosers, saved filters, export.
- A detail page from generated columns: models generates columns for
  views only, and a detail page reads a table's row. Worth a look once
  tables are built.
- Per-record permissions: auth has none yet. `Unless` and `If` cover
  state; a rule that reads the session is the app's.
