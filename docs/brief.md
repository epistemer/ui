# ui kernel: design brief

Written 2026-11-17 by epistemer and Claude; brought into the repository on
2026-11-21 with the changes listed in [§9](#9-changes-since-the-first-draft).
The kernel is the root package `ui` of `github.com/epistemer/ui`. It gives
every other ui package three primitives and a test kit. Whatever templ
already does, the kernel leaves to templ. Forms, the module's other first
package, has its own design: [forms-design.md](forms-design.md).

## 1. Scope

**What the kernel provides.**

| Piece | What it answers | Milestone |
|---|---|---|
| Context values, `Provider[T]` | A typed value a whole subtree can read without passing it down as a parameter: a form's id, a table's state, the signed-in user | M1 |
| Element IDs, `ID` | Ids that wire elements together (`for`, `aria-*`, `form`, `hx-target`) and never collide, even across fragments rendered by separate requests | M2 |
| Test kit, `uitest` | Golden HTML, and a check that every id reference in a page resolves | M3 |
| Slots, `Slot` and `SlotOf[T]` | Inline content a caller puts into named places of a component, each with a fallback | M4 |

**What templ already provides.** Used as they are, not wrapped.

| Need | In templ |
|---|---|
| Content rendered once per page, such as a component's script | `templ.NewOnceHandle` |
| CSP nonces | `templ.WithNonce`, `templ.GetNonce` |
| Rendering only part of a page, for htmx | `templ.Fragment`, `templ.RenderFragments` |
| Streaming | `templ.Flush` |
| Class lists and conditional classes | `templ.Classes`, `templ.KV` |
| A local variable inside a template | raw Go statements, `{{ ... }}` |
| Content passed as a value | component parameters |

templ has no named slots. Its documentation passes several components as
parameters instead, which works but forces each piece of content into a
separate templ function. Kernel slots exist for content written inline.

**What other neozp packages own.**

| Concern | Owner |
|---|---|
| Asset URLs with cache busting | ui/layout, or an assets package of its own |
| Text direction from the locale | i18n or ui/layout |
| Formatting values for display | forms codecs, i18n's static labels |
| Cross-field conditions on the client | forms, through `Cond` |
| htmx attribute sets, file previews | ui/elements, ui/inputs |
| Merging forwarded attributes | ui/elements, unless two packages need it (backlog) |
| Routes, URL builders, the HTML error page | handlers |
| CSRF | auth (D13: `http.CrossOriginProtection`, no per-form token) |

**Module and dependencies.**

- Module `github.com/epistemer/ui`, Go 1.27 (decided 2026-11-21: forms, in
  the same module, uses generic methods). The kernel is the root package
  `ui`; the test kit is `ui/uitest`; forms is `ui/forms`. Later packages
  (icons, elements, inputs, components, layout) join the same module.
- The kernel imports templ and the standard library only. It needs neither
  i18n nor errors: its errors are programmer mistakes that end as a 500, so
  plain sentinel errors fit.
- uitest adds `golang.org/x/net/html` (ADR 0003).
- templ v0.3.1118, the current release on 2026-11-21.

## 2. Invariants

An item that breaks one is wrong even if its tests pass.

| # | Invariant |
|---|---|
| K1 | No setup per request. There is no root component: a component renders correctly in a page, an htmx fragment, an SSE event or a test. |
| K2 | Everything a render carries lives in its context as immutable values, written with `context.WithValue` and never changed. A context captured during a render stays valid. |
| K3 | Every kernel component reads its block with `templ.GetChildren` and calls `templ.ClearChildren` before it renders anything, as templ's documentation requires. Otherwise a component further down that is called without a block receives the kernel component's block. |
| K4 | A caller's block runs once per render of the component it is passed to. |
| K5 | Nothing on the kernel's render path buffers output. |
| K6 | Render errors are returned, never dropped. |
| K7 | A mistake found at declaration, such as an invalid ID name, panics. A mistake found during a render returns an error from Render; a function that returns a value and no error, such as `Get` or `Value`, panics instead. |
| K8 | Lookups go by handle, never by a name string. Two packages may declare slots or IDs with the same name and never meet. |

## 3. What templ does today

Checked in templ's source on 2026-11-17 (v0.3.1118, and main at 0.3.1123):

- Children travel in the context as an immutable value: `WithChildren`
  returns a derived context.
- A generated component reads its children and clears them on entry. A
  generated child block does not clear them.
- So a Go component that renders its block must clear the children itself,
  or a component inside the block that is called without a block of its own
  renders the outer block again, recursing. templ's documentation shows
  exactly that `ClearChildren` call. This is K3.

M0.3's contract tests run first after every templ upgrade. K3 and the slot
collect pass (§6) depend on these behaviours, and the tests fail before
anything subtler does.

## 4. Context values (M1)

A provider passes a typed value down a subtree. It is the smallest
primitive, and the pattern the other two reuse: a value in the context,
keyed by a handle.

```go
type Provider[T any] struct{ /* unexported */ }

func NewProvider[T any](def T) *Provider[T]

func (p *Provider[T]) Provide(v T) templ.Component                   // in a template: @Currency.Provide("EGP") { ... }
func (p *Provider[T]) With(ctx context.Context, v T) context.Context // in Go: middleware and handlers
func (p *Provider[T]) Value(ctx context.Context) T                   // the nearest value, else the default
```

- Values are stored with `context.WithValue`, keyed by the provider (K2).
  There is no per-render state to set up or to clean.
- `NewProvider` infers T from its default. A default means something here,
  because `Value` must answer even where nothing was provided.
- `Value` replaces the name `Consume`, so providers and slots read values the
  same way.

## 5. Element IDs (M2)

An ID handle gives two kinds of id: generated ids, unique per render, for
wiring elements inside one render; and keyed ids, the same on every render,
for elements another request targets. Neither needs a root component.

```go
type ID struct{ /* unexported; copies refer to the same handle */ }

func NewID(name string) ID // name matches [a-z][a-z0-9]*(-[a-z0-9]+)*, else NewID panics

func Scope(ids ...ID) templ.Component                          // @ui.Scope(EmailID) { ... }: a fresh value per id for the block
func (id ID) Get(ctx context.Context) string                   // the nearest Scope's value; panics outside one
func (id ID) Part(ctx context.Context, parts ...string) string // Get(ctx) plus escaped parts
func (id ID) Key(parts ...string) string                       // stable, from data; needs no context

type IDSource interface{ Next() string } // each token unique among its own
func WithIDs(ctx context.Context, s IDSource) context.Context
func Sequence(prefix string) IDSource    // prefix-1, prefix-2, ...: byte-stable output
```

| Kind | Call | Example | Same on the next render |
|---|---|---|---|
| Generated | `EmailID.Get(ctx)` | `email--k3x9a2-1f` | No |
| Generated, with a part | `EmailID.Part(ctx, "hint")` | `email--k3x9a2-1f-hint` | No |
| Keyed | `OrderRow.Key("1042")` | `order-row-1042` | Yes |
| Keyed, escaped | `OrderRow.Key("A/7")` | `order-row-A_2f7` | Yes |

- **Which kind.** Generated ids wire elements inside one render: `for` and
  `id`, `aria-*`, `form`. Keyed ids are for elements that a different request
  targets (`hx-target`, `hx-swap-oob`) or that htmx swaps in place.
- **Why process-wide.** htmx matches old and new elements by id when it
  settles a swap, so fragments rendered by separate requests, such as an SSE
  row or a modal loaded later, must never share a generated id. The default
  source is one counter per process that starts from 6 random base-36
  characters, so a clash between two server processes is a 1-in-2-billion
  event and nothing has to be set up per request (K1).
- **No collisions by construction.** Generated ids always contain `--` and
  keyed ids never do. Part escaping turns every byte outside `[A-Za-z0-9]`
  into `_` and two hex digits, and an empty part into `_`, so different parts
  never give the same id. Every id is a valid HTML id and CSS identifier, so
  a `#id` selector needs no escaping.
- **No fallback.** `Get` outside a scope panics with the id's name and the
  fix, `ui.Scope(EmailID)`. A reference that silently gets a fresh id is the
  bug scopes exist to prevent.
- **WithIDs is optional.** Tests use `Sequence`. A handler may use it for
  byte-stable output, giving each separately rendered fragment its own
  prefix.

## 6. Test kit (M3)

uitest lets every ui package test components as HTML: render with
deterministic ids, compare against golden files, and check that every id
reference resolves. It comes before slots so that M4 is tested with it.

```go
package uitest

func Render(t testing.TB, c templ.Component, opts ...Option) string // ids from ui.Sequence("t"); a render error fails t
func RenderErr(c templ.Component, opts ...Option) (string, error)
func Context(f func(context.Context) context.Context) Option      // providers, a locale, ...

func Golden(t testing.TB, name, html string)               // testdata/<name>.golden.html; go test -update rewrites it
func CheckIDs(t testing.TB, html string, opts ...RefOption) // ids unique, every reference resolves
func External(ids ...string) RefOption                     // ids a fragment may reference on the page around it
```

- **Readable goldens.** Before comparing, `Golden` parses the HTML and prints
  one element per line with attributes sorted and text trimmed. A changed
  attribute is one changed line in the diff, and the golden does not depend
  on attribute order.
- **What CheckIDs follows.** `for`, `form`, `list`, `headers`,
  `popovertarget`, `commandfor`, the `aria-*` attributes that take ids
  (`controls`, `labelledby`, `describedby`, `owns`, `activedescendant`,
  `errormessage`, `details`, `flowto`), `href="#..."`, and htmx's
  `hx-target`, `hx-include`, `hx-indicator`, `hx-select` and
  `hx-disabled-elt` when the value is a plain `#id`.
- **Why it matters.** A typed ID handle makes a broken reference unlikely,
  but not impossible: a scope opened in the wrong place still renders.
  CheckIDs catches what the compiler cannot, on every example and every
  golden test.

## 7. Slots (M4)

Slots let a caller write content inline into named places of a component.
The model is strict: a component's block holds only fills and runs once, and
the layout renders straight to the writer. Loose content in a block is an
error; it is never the fallback of an empty slot.

```go
func NewSlot() *Slot               // passes no value: most slots
func NewSlotOf[T any]() *SlotOf[T] // passes a T from the component to whoever fills it

type SlotRef interface{ /* unexported method; *Slot and *SlotOf[T] implement it */ }

// Component author: wrap the layout and list the component's slots.
func Slotted(layout templ.Component, slots ...SlotRef) templ.Component

// In the layout. Filled is on both types.
func (s *Slot) Place() templ.Component         // the caller's fill, else this call's own block
func (s *SlotOf[T]) Place(v T) templ.Component
func (s *Slot) Filled(ctx context.Context) bool

// In a fill, or in the fallback of a valued slot.
func (s *SlotOf[T]) Value(ctx context.Context) T

// Caller: a slot is a templ.Component. Rendering it inside the component's block fills it.
```

A component from the shop examples:

```go
var (
	CartToggle = ui.NewSlotOf[templ.Attributes]()
	CartBody   = ui.NewSlot()
	cartDialog = ui.NewID("cart-dialog")
)

func CartModal(title string) templ.Component {
	return ui.Slotted(cartModal(title), CartToggle, CartBody)
}
```

```templ
templ cartModal(title string) {
	@ui.Scope(cartDialog) {
		@CartToggle.Place(templ.Attributes{"aria-controls": cartDialog.Get(ctx), "aria-haspopup": "dialog"}) {
			<button { CartToggle.Value(ctx)... }>Cart</button>
		}
		<dialog id={ cartDialog.Get(ctx) }>
			<h2>{ title }</h2>
			@CartBody.Place() {
				<p>Your cart is empty.</p>
			}
		</dialog>
	}
}
```

And a caller:

```templ
@shop.CartModal("Your cart") {
	@shop.CartToggle {
		<button { shop.CartToggle.Value(ctx)... } class="badge">{ strconv.Itoa(len(items)) }</button>
	}
	if len(items) > 0 {
		@shop.CartBody {
			for _, it := range items {
				@shop.CartLine(it)
			}
		}
	}
}
```

### 7.1 Rules

1. **One collect pass.** `Slotted` takes the caller's block, clears the
   children (K3) and renders the block once into a sink that keeps nothing.
   Rendering a slot there registers that slot's block and writes nothing.
   Anything else that writes output, other than white space, is an error.
2. **One layout pass.** The layout then renders once, straight to the
   writer. Every placement already knows its fill, so nothing is buffered
   (K5).
3. **A fill replaces the fallback, even when it renders nothing.** `Place`
   renders the caller's fill if there is one, otherwise the placement's own
   block, otherwise nothing. To keep the fallback, the caller leaves the slot
   unfilled; an `if` around the fill does that.
4. **Fill content sees the placement.** It renders in the placement's
   context, so the layout's providers, its ID scopes and the slot's value are
   all visible.
5. **Slot lookups inside fill content are lexical.** A placement or `Filled`
   inside fill content refers to the component its author was writing, not
   to the component being filled. A layout can therefore place one of its own
   slots inside a fill it gives another component, as an order row does when
   its actions go inside a dropdown menu.
6. **Each piece runs as often as it is used.** The block runs once per render
   (K4). A fill runs once per placement, so twice if the layout places it
   twice. A fallback runs only when its slot is unfilled.
7. **Kernel wrappers refuse to run in a block.** `Provide`, `Scope` and
   `Place` written directly in a block, around fills, return `ErrInBlock`:
   their values would never reach the fills, which render later inside the
   layout. The fix is to open them around the component call. A wrapper of
   your own that only renders its block, such as a permission check, is fine
   there.

| Error | When |
|---|---|
| `ErrStrayContent` | The block writes anything but white space outside a fill |
| `ErrFilledTwice` | A block fills the same slot twice, for example inside a loop |
| `ErrNotDeclared` | A block fills, or a layout places, a slot the component did not list in `Slotted` |
| `ErrNoComponent` | A slot is filled outside any component's block, or placed outside a layout |
| `ErrInBlock` | `Provide`, `Scope` or `Place` is written directly in a block |

Each message names the slot by the file and line of its `NewSlot`, captured
once when the variable is declared. `Value` and `Filled` panic outside a
placement and outside a layout respectively (K7). A slot may be listed by
several components. A listed slot that the layout leaves unplaced on one
render is not an error, because placements may be conditional.

### 7.2 Why strict

zp's slots taught three things.

- **Loose content became the fallback of every empty slot.** One component
  put a row's cells beside an actions slot, so a row without actions showed
  its cells a second time inside its actions menu.
- **Each placement re-ran the caller's whole block.** A component with three
  slots ran it three times.
- **A fill that rendered only white space brought the fallback back,** so a
  caller could not remove a default.

**On `NewSlot(v)`.** Of zp's 13 slots, 8 carried no value and 4 carried a
value computed during the render, so a default would be a dummy that only
names the type. `NewSlot()` takes no type at all, and a slot that passes a
value names its type once, `NewSlotOf[T]()`. Placing a valued slot without
its value does not compile.

## 8. Decisions and risks

| # | Question | Decision | What reversing it costs |
|---|---|---|---|
| 1 | How does a caller fill a slot? | Bare, `@shop.CartBody { ... }`: callers outnumber components, with 141 fills against 13 slots in zp | `.Fill()` on both slot types, written by every caller |
| 2 | Does a fill that renders nothing keep the fallback? | No, a fill always wins (rule 3) | Each placement buffers its fill to test it for content, against K5 |
| 3 | Is a listed slot that is never placed an error? | No, placements may be conditional | Components must place every listed slot on every render |
| 4 | What does `Get` do outside a scope? | It panics | A fresh id on each call, which makes broken references silent |
| 5 | Where do generated ids come from by default? | One source per process, with no setup (K1) | Every request installs `WithIDs`, and a forgotten one panics |
| 6 | Is the default slot built after v0.1.0? | Not until a component needs it (backlog) | The late-collect design and its buffering |
| 7 | Are the names right? | `Slotted`, `Place`, `SlotOf`, `Scope`, `Part`, `Key`, and `Provider.Value` in place of `Consume` | Free before the lock; an amendment after it |

Decisions 1–7 were proposed as defaults on 2026-11-17 and accepted by
epistemer with the plan of 2026-11-21.

| Risk | Likelihood / impact | Mitigation |
|---|---|---|
| templ changes how children travel in the context | Low / high | M0.3 runs first after every templ upgrade |
| A wrapper of your own that changes the context, written around fills in a block, silently loses its values | Medium / medium | The kernel's own wrappers refuse with `ErrInBlock`, and the README shows the wrapper moved around the component call |
| Strict blocks feel verbose for a component with one content area | Medium / low | Such a component takes plain children and no slots; the default slot is in the backlog |
| Generated ids differ on every render | Certain / low | `Sequence` for tests and byte-stable output, normalised goldens, and keyed ids where stability matters |
| A misused `Get` or `Value` panics during a request | Low / medium | `net/http` recovers a handler's panic by logging it and aborting the connection: the client gets no 500 page, and a streamed render is cut off. handlers v0.1.0 adds no recovery. So the guard is testing: the message names the fix, a test covers each panic, and uitest renders every example. A recovering wrapper in handlers is a follow-up (API design, epistemer's) |

## 9. Changes since the first draft

Applied on 2026-11-21, after checking the draft against neozp as it stands:

- **Module.** Forms joins this module as `ui/forms` instead of its own
  repository (epistemer, 2026-11-21), so the module starts at Go 1.27 rather
  than moving to it later, and CI runs Go 1.27 only.
- **v1.0.0's gate.** The draft waited for zp's pages to run on the ui
  packages. Moving zp onto neozp is out of neozp's scope (2026-11-16), so
  v1.0.0 is gated on neozp alone (roadmap).
- **Panic risk.** The draft said the HTTP server turns a panic into a 500.
  It doesn't (§8).
- **Ownership** was delegated on 2026-11-21 (ownership.md).
- uitest's dependency ADR is 0003; 0002 is forms'.
