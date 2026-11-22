# ui

neozp's UI module for Go and [templ](https://templ.guide).

The kernel provides typed context values (`Provider`), element IDs that wire
elements together without colliding across separately rendered fragments (`ID`),
and named slots for inline content (`Slot`, `SlotOf`). It imports templ and the
standard library only.

- **`ui/forms`** — declare, bind, validate and describe HTML forms: one
  typed `Field[T]` per field, typed rules (`ui/forms/rule`), cross-field
  conditions shared with the client, groups and lists. Imports no templ.

## Writing a slotted component

Slots let a caller write inline content into named places of a component.
Declare slots as package-level variables using `ui.NewSlot()` for standard slots
or `ui.NewSlotOf[T]()` for valued slots that pass data to their fills.

Wrap the layout with `ui.Slotted(layout, slots...)`, listing every slot the
component accepts. In the layout, place each slot with `Place`:

```templ
package components

import "github.com/epistemer/ui"

var (
	CartToggle = ui.NewSlotOf[templ.Attributes]()
	CartBody   = ui.NewSlot()
	cartDialog = ui.NewID("cart-dialog")
)

func CartModal(title string) templ.Component {
	return ui.Slotted(cartModal(title), CartToggle, CartBody)
}

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

Key patterns:
- **Fallbacks:** Pass a block to `@slot.Place() { ... }`. When unfilled, the
  placement renders its fallback.
- **Valued slots:** `@valuedSlot.Place(val)` provides data to callers. Call
  `valuedSlot.Value(ctx)` inside the placement's fallback or inside a fill.
- **Conditional wrappers (`Filled`):** Call `slot.Filled(ctx)` to inspect whether
  the caller filled the slot (e.g. `if Footer.Filled(ctx) { <footer>@Footer.Place()</footer> }`)
  so empty wrapper tags are omitted.

### Strict rules

1. **A block holds only fills:** The caller block runs once during a collect pass
   that sinks output. Any loose content outside a slot fill returns `ui.ErrStrayContent`.
2. **No wrappers in blocks:** a provider's `Provide`, `ui.Scope` and `Place` written directly
   inside a slotted component's caller block return `ui.ErrInBlock` because their
   context values would never reach the fills (which render later during the layout
   pass). Open them around the component call instead.

### Testing slotted components

Package `uitest` provides test helpers that render components with deterministic IDs
from `ui.Sequence("t")`:

```go
func TestCartModal(t *testing.T) {
	html := uitest.Render(t, components.CartModal("Your cart"))
	uitest.CheckIDs(t, html)
	uitest.Golden(t, "cart_modal", html)
}
```

- `uitest.Render(t, component)` renders the component to an HTML string.
- `uitest.CheckIDs(t, html)` verifies that IDs are unique and non-empty, and that
  every reference (`for`, `aria-*`, `hx-target`, etc.) resolves.
- `uitest.Golden(t, name, html)` normalises HTML (one element per line, sorted
  attributes) and asserts against `testdata/<name>.golden.html` (update with `-update`).

When rendered unfilled, `CartModal("Your cart")` produces:

```html
<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog">Cart</button><dialog id="cart-dialog--t-1"><h2>Your cart</h2><p>Your cart is empty.</p></dialog>
```

## Filling one

A caller fills slots by rendering them directly inside the component's block:

```templ
@components.CartModal("Your Cart") {
	@components.CartToggle {
		<button { components.CartToggle.Value(ctx)... } class="badge">View Cart (2)</button>
	}
	if len(items) > 0 {
		@components.CartBody {
			<ul>
				<li>Green Tea</li>
				<li>Coffee Beans</li>
			</ul>
		}
	}
}
```

Renders:

```html
<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog" class="badge">View Cart (2)</button><dialog id="cart-dialog--t-1"><h2>Your Cart</h2><ul><li>Green Tea</li><li>Coffee Beans</li></ul></dialog>
```

### Conditional fills and fallbacks

To conditionally fill a slot, wrap the fill in `if`:
- When the condition is true, the fill replaces the fallback.
- When false, the slot is not filled, so the fallback renders.

A fill replaces the fallback **even when it renders nothing**. For example:

```templ
@components.CartModal("Your cart") {
	@components.CartBody {
	}
}
```

Renders:

```html
<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog">Cart</button><dialog id="cart-dialog--t-1"><h2>Your cart</h2></dialog>
```

Here `<p>Your cart is empty.</p>` is suppressed. This rule allows callers to
intentionally clear default content, and ensures the kernel never buffers rendered
output to check whether a fill was empty. To keep the fallback, do not fill the
slot (use `if`).

## Declaring a form and handling it

Embed `*forms.Form` in a struct of typed field handles. A constructor declares
fields, rules, conditions, groups, and lists:

```go
type ToyForm struct {
	*forms.Form
	Name     *forms.Field[string]
	Electric *forms.Field[bool]
	Voltage  *forms.Field[int]
	Safety   *SafetyForm
	Tags     *forms.List[*TagForm]
}

func NewToyForm() *ToyForm {
	f := &ToyForm{Form: forms.New()}
	f.Name = f.Text("name").Required().Is(rule.MinLen(3))
	f.Electric = f.Checkbox("electric")
	f.Voltage = f.Int("voltage").When(f.Electric.Filled()).Required()
	f.Safety = f.Group("safety", func(g *forms.Form) *SafetyForm {
		return &SafetyForm{Form: g, Age: g.Int("min_age").Required()}
	})
	f.Tags = f.List("tags", func(g *forms.Form) *TagForm {
		return &TagForm{Form: g, Name: g.Text("name").Required()}
	})
	return f
}
```

### Handling submissions and edits

In an HTTP handler, `Submit` parses the body, binds values, and validates. On
failure or service errors (`Fail`), re-render with `422 Unprocessable Entity`:

```go
func (h *Handler) CreateToy(w http.ResponseWriter, r *http.Request) {
	f := NewToyForm()
	if !f.Submit(r.Context(), r) {
		h.render(w, r, http.StatusUnprocessableEntity, f)
		return
	}
	toy, err := h.toys.Create(r.Context(), f.Name.Value(), f.Voltage.Ptr())
	if err != nil {
		f.Fail(err)
		h.render(w, r, http.StatusUnprocessableEntity, f)
		return
	}
	http.Redirect(w, r, "/toys/"+toy.ID, http.StatusSeeOther)
}
```

For an edit page, populate initial values using `Set` (or `SetPtr` for nullable
values):

```go
func (h *Handler) EditToy(w http.ResponseWriter, r *http.Request, toy Toy) {
	f := NewToyForm()
	f.Name.Set(toy.Name)
	f.Electric.Set(toy.Voltage != nil) // Voltage's condition reads it
	f.Voltage.SetPtr(toy.Voltage)
	h.render(w, r, http.StatusOK, f)
}
```

### Errors and testing

Errors are translated with `errors.T(ctx, err)`:

```go
if err := f.Name.Err(); err != nil {
	msg := errors.T(r.Context(), err) // "must be at least 3 characters"
}
```

Package `formstest` tests forms without HTTP:

```go
func TestToyForm(t *testing.T) {
	f := NewToyForm()
	ok := formstest.Submit(t, context.Background(), f.Form, "name=Bear&electric=on&voltage=5&safety.min_age=3")
	if !ok {
		t.Fatalf("unexpected errors: %v", formstest.Errors(f.Form))
	}
}
```

## Building a page

The visual packages render with ui's own CSS, in cascade layers an app's
styles override, and a small script of ES modules. Mount the theme, then
build pages from the shells, components and elements:

```go
th, err := theme.New("/ui/", settings) // settings: theme.Config, e.g. themeschema's
mux.Handle("/ui/", th)
```

```templ
templ ordersPage(r *http.Request, th *theme.Theme) {
	@layout.Document(i18n.Static("Orders")).Head(th.Head()).ColorScheme(layout.ColorSchemeOf(r)) {
		@layout.App() {
			@layout.AppBrand {
				Bricks & Co
			}
			@layout.AppNav {
				@layout.NavItem("/orders").Current(true) {
					Orders
				}
			}
			@layout.AppBar {
				@layout.ColorSchemeMenu(layout.ColorSchemeOf(r))
			}
			@layout.AppMain {
				@layout.PageHeader(i18n.Static("Orders")) {
					@elements.Button().Href("/orders/new") {
						New order
					}
				}
			}
		}
	}
}
```

The brand colours, fonts and radius change at run time through cfgx
(`theme/themeschema`). `go run ./cmd/catalog` shows every component in its
states, in LTR and RTL and in light and dark.

Status: v0.2.0. The designs are [docs/brief.md](docs/brief.md),
[docs/forms-design.md](docs/forms-design.md) and
[docs/visual-design.md](docs/visual-design.md); the contracts are
[docs/interface.md](docs/interface.md) and
[docs/visual-interface.md](docs/visual-interface.md); progress is in
[docs/roadmap.md](docs/roadmap.md).

Requires Go 1.27.

## License

MIT
