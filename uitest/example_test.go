package uitest_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/uitest"
)

// ExampleRender demonstrates rendering a component to HTML using uitest.RenderErr.
// Generated IDs are automatically drawn from ui.Sequence("t").
func ExampleRender() {
	btnID := ui.NewID("cart-btn")

	component := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintf(w, "<button id=%q>Cart</button>", btnID.Get(ctx))
			return nil
		})
		return ui.Scope(btnID).Render(templ.WithChildren(ctx, child), w)
	})

	html, err := uitest.RenderErr(component)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(html)

	// Output:
	// <button id="cart-btn--t-1">Cart</button>
}

// ExampleCheckIDs demonstrates verifying that every id in rendered HTML is unique
// and that all id references (such as for, aria-*, and hx-target) resolve.
func ExampleCheckIDs() {
	// In a test function TestXxx(t *testing.T):
	var t *testing.T

	rawHTML := `<label for="email--t-1">Email</label><input id="email--t-1"/>`
	uitest.CheckIDs(t, rawHTML)
}
