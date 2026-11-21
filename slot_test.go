package ui_test

import (
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/kerneltest"
	"github.com/epistemer/ui/uitest"
)

var between = regexp.MustCompile(`>\s+<`)

func squash(s string) string { return between.ReplaceAllString(strings.TrimSpace(s), "><") }

func TestCartModal(t *testing.T) {
	for _, tc := range []struct {
		items []string
		want  string
	}{
		{[]string{"tea", "rice"}, `<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog" class="badge">2</button><dialog id="cart-dialog--t-1"><h2>Your cart</h2><ul><li>tea</li><li>rice</li></ul></dialog>`},
		{nil, `<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog" class="badge">0</button><dialog id="cart-dialog--t-1"><h2>Your cart</h2><p>Your cart is empty.</p></dialog>`},
	} {
		got := squash(uitest.Render(t, kerneltest.CartPage(tc.items)))
		if got != tc.want {
			t.Errorf("items %v:\ngot  %s\nwant %s", tc.items, got, tc.want)
		}
	}
}

// counting builds CartModal's block in Go, counting how often the block,
// the fill and the fallback run.
func TestExecutionCounts(t *testing.T) {
	var blockRuns, fillRuns int
	fillContent := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		fillRuns++
		_, err := io.WriteString(w, "<b>fill</b>")
		return err
	})
	blockOf := func(fillBody bool) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			blockRuns++
			if fillBody {
				return kerneltest.TwiceBody.Render(templ.WithChildren(ctx, fillContent), w)
			}
			return nil
		})
	}
	render := func(fill bool) string {
		blockRuns, fillRuns = 0, 0
		ctx := templ.WithChildren(context.Background(), blockOf(fill))
		var b strings.Builder
		if err := kerneltest.TwicePlaced().Render(ctx, &b); err != nil {
			t.Fatal(err)
		}
		return squash(b.String())
	}
	if got := render(true); got != "<b>fill</b><b>fill</b>" || blockRuns != 1 || fillRuns != 2 {
		t.Errorf("filled: %q, block ran %d, fill ran %d; want once and twice", got, blockRuns, fillRuns)
	}
	if got := render(false); got != "<i>fallback</i><i>fallback</i>" || blockRuns != 1 || fillRuns != 0 {
		t.Errorf("unfilled: %q, block ran %d, fill ran %d", got, blockRuns, fillRuns)
	}
	empty := templ.ComponentFunc(func(context.Context, io.Writer) error { return nil })
	ctx := templ.WithChildren(context.Background(), templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return kerneltest.TwiceBody.Render(templ.WithChildren(ctx, empty), w)
	}))
	var b strings.Builder
	if err := kerneltest.TwicePlaced().Render(ctx, &b); err != nil || strings.TrimSpace(b.String()) != "" {
		t.Errorf("a fill that renders nothing must still replace the fallback: %q %v", b.String(), err)
	}
}

func TestLexicalLookups(t *testing.T) {
	got := squash(uitest.Render(t, kerneltest.Orders()))
	want := `<table><tbody>` +
		`<tr><td>1042</td><td><details><summary>Actions</summary><menu><li>Refund</li></menu></details></td></tr>` +
		`<tr><td>1043</td><td><details><summary>Actions</summary><menu><li>No actions</li></menu></details></td></tr>` +
		`</tbody></table>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestNestedModals(t *testing.T) {
	got := squash(uitest.Render(t, kerneltest.NestedModals()))
	want := `<button aria-controls="cart-dialog--t-1" aria-haspopup="dialog">Cart</button><dialog id="cart-dialog--t-1"><h2>Outer</h2>` +
		`<p>outer body</p><button aria-controls="cart-dialog--t-2" aria-haspopup="dialog">Cart</button><dialog id="cart-dialog--t-2"><h2>Inner</h2><p>inner body</p></dialog>` +
		`</dialog>`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestFilled(t *testing.T) {
	got := squash(uitest.Render(t, kerneltest.Cards()))
	if want := `<article><h3>Tea</h3><footer><button>Add</button></footer></article><article><h3>Coffee</h3></article>`; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestSlotErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    templ.Component
		want error
		says string
	}{
		{"stray content", kerneltest.Stray(), ui.ErrStrayContent, `"<p>loose`},
		{"filled twice", kerneltest.FilledTwice(), ui.ErrFilledTwice, "_templ.go:"},
		{"fills a slot not listed", kerneltest.FillsUndeclared(), ui.ErrNotDeclared, "_templ.go:"},
		{"places a slot not listed", kerneltest.PlacesUndeclared(), ui.ErrNotDeclared, "_templ.go:"},
		{"filled outside a component", kerneltest.FillOutside(), ui.ErrNoComponent, "_templ.go:"},
		{"placed outside a layout", kerneltest.PlaceOutside(), ui.ErrNoComponent, "_templ.go:"},
		{"a slotted component in a block", kerneltest.SlottedInBlock(), ui.ErrStrayContent, "slotted component"},
		{"Provide in a block", kerneltest.ProvideInBlock(), ui.ErrInBlock, "Provide"},
		{"Scope in a block", kerneltest.ScopeInBlock(), ui.ErrInBlock, "ui.Scope"},
		{"Place in a block", kerneltest.PlaceInBlock(), ui.ErrInBlock, "slot declared at"},
		{"stray before a second fill", kerneltest.StrayThenTwice(), ui.ErrStrayContent, `"<b>early`},
		{"stray before Provide", kerneltest.StrayThenProvide(), ui.ErrStrayContent, `"<b>early`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uitest.RenderErr(tc.c)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("message %q does not contain %q", err, tc.says)
			}
		})
	}
}

func TestProvideAroundTheCall(t *testing.T) {
	if got := squash(uitest.Render(t, kerneltest.ProvidedBody())); !strings.Contains(got, "<b>EGP</b>") {
		t.Errorf("the fill must see a value provided around the component call: %s", got)
	}
}

func TestWrapperInBlockIsFine(t *testing.T) {
	passThrough := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return templ.GetChildren(ctx).Render(templ.ClearChildren(ctx), w)
	})
	fill := templ.ComponentFunc(func(_ context.Context, w io.Writer) error { _, err := io.WriteString(w, "<p>body</p>"); return err })
	block := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		inner := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return kerneltest.CartBody.Render(templ.WithChildren(ctx, fill), w)
		})
		return passThrough.Render(templ.WithChildren(ctx, inner), w)
	})
	html, err := uitest.RenderErr(kerneltest.CartModal("t"), uitest.Context(func(ctx context.Context) context.Context {
		return templ.WithChildren(ctx, block)
	}))
	if err != nil || !strings.Contains(html, "<p>body</p>") {
		t.Errorf("a wrapper that only renders its block: %v, %s", err, html)
	}
}

func TestSlotPanics(t *testing.T) {
	valued := ui.NewSlotOf[string]()
	for name, fn := range map[string]func(){
		"Slotted nil":      func() { ui.Slotted(nil) },
		"Slotted nil slot": func() { ui.Slotted(kerneltest.Price(), nil) },
		"Slotted twice":    func() { ui.Slotted(kerneltest.Price(), kerneltest.CartBody, kerneltest.CartBody) },
		"Filled outside":   func() { kerneltest.CartBody.Filled(context.Background()) },
		"Value outside":    func() { valued.Value(context.Background()) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

func TestValuedSlotNilInterface(t *testing.T) {
	s := ui.NewSlotOf[error]()
	var got error = io.EOF
	probe := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error { got = s.Value(ctx); return nil })
	layout := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return s.Place(nil).Render(templ.WithChildren(ctx, probe), w)
	})
	if _, err := uitest.RenderErr(ui.Slotted(layout, s)); err != nil || got != nil {
		t.Errorf("Place(nil) of an interface type: Value %v, err %v", got, err)
	}
}

func TestStrayWhiteSpace(t *testing.T) {
	run := func(text string) error {
		block := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			_, err := io.WriteString(w, text)
			return err
		})
		_, err := uitest.RenderErr(kerneltest.CartModal("t"), uitest.Context(func(ctx context.Context) context.Context {
			return templ.WithChildren(ctx, block)
		}))
		return err
	}
	if err := run(" \t\n\f\r"); err != nil {
		t.Errorf("HTML white space: %v", err)
	}
	if err := run("\u00a0"); !errors.Is(err, ui.ErrStrayContent) {
		t.Errorf("a non-breaking space is content: %v", err)
	}
	long := strings.Repeat("x", 100)
	if err := run(long); err == nil || strings.Contains(err.Error(), strings.Repeat("x", 41)) || !strings.Contains(err.Error(), strings.Repeat("x", 40)) {
		t.Errorf("the message quotes 40 bytes: %v", err)
	}
}

func TestStrayIgnoredWriteError(t *testing.T) {
	block := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, "<p>ignored</p>") // the error is dropped
		return nil
	})
	_, err := uitest.RenderErr(kerneltest.CartModal("t"), uitest.Context(func(ctx context.Context) context.Context {
		return templ.WithChildren(ctx, block)
	}))
	if !errors.Is(err, ui.ErrStrayContent) {
		t.Errorf("a block that drops the write's error: %v", err)
	}
}

func TestValueShadowing(t *testing.T) {
	if got := squash(uitest.Render(t, kerneltest.NestedValued())); got != "<i>inner</i><i>outer</i>" {
		t.Errorf("got %s: the nearest placement's value wins, the outer one returns after", got)
	}
}

func TestSlottedInFallback(t *testing.T) {
	if got := squash(uitest.Render(t, kerneltest.CardInFallback())); !strings.Contains(got, "<h3>Fallback card</h3>") {
		t.Errorf("a slotted component inside a fallback: %s", got)
	}
}

func TestNilAndZeroSlots(t *testing.T) {
	var typedNil *ui.Slot
	for name, fn := range map[string]func(){
		"typed nil in Slotted": func() { ui.Slotted(kerneltest.Price(), typedNil) },
		"zero Slot Place":      func() { (&ui.Slot{}).Place() },
		"zero SlotOf Value":    func() { (&ui.SlotOf[int]{}).Value(context.Background()) },
		"Filled undeclared": func() {
			layout := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
				kerneltest.CardFooter.Filled(ctx)
				return nil
			})
			ui.Slotted(layout, kerneltest.CartBody).Render(context.Background(), io.Discard)
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if msg, _ := recover().(string); !strings.HasPrefix(msg, "ui:") {
					t.Errorf("panic %q: want a ui: message", msg)
				}
			}()
			fn()
		})
	}
}
