// Package uitest tests ui components as HTML: render with deterministic
// ids, compare with golden files, and check that every id reference in a
// page resolves.
//
//	func TestCartModal(t *testing.T) {
//		html := uitest.Render(t, shop.CartModal("Your cart"))
//		uitest.CheckIDs(t, html)
//		uitest.Golden(t, "cart_modal", html)
//	}
package uitest

import (
	"bytes"
	"context"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
)

// Option changes how a component is rendered.
type Option func(*options)

type options struct {
	ctx []func(context.Context) context.Context
}

// Context adds a step that prepares the render's context, such as a
// provider's value or a locale. Steps run in order, after ids are set to
// ui.Sequence("t"), so a step that calls ui.WithIDs overrides it.
func Context(f func(context.Context) context.Context) Option {
	if f == nil {
		panic("uitest: Context(nil)")
	}
	return func(o *options) { o.ctx = append(o.ctx, f) }
}

// Render renders c with generated ids from ui.Sequence("t"), in a context
// derived from t.Context(), and returns its HTML. A render error fails t.
func Render(t testing.TB, c templ.Component, opts ...Option) string {
	t.Helper()
	html, err := render(t.Context(), c, opts)
	if err != nil {
		t.Fatalf("uitest: render: %v", err)
	}
	return html
}

// RenderErr is Render for a test that expects an error.
func RenderErr(c templ.Component, opts ...Option) (string, error) {
	return render(context.Background(), c, opts)
}

func render(ctx context.Context, c templ.Component, opts []Option) (string, error) {
	var o options
	for _, opt := range opts {
		if opt != nil { // a zero Option, as in a table test, does nothing
			opt(&o)
		}
	}
	ctx = ui.WithIDs(ctx, ui.Sequence("t"))
	for _, f := range o.ctx {
		ctx = f(ctx)
	}
	var buf bytes.Buffer
	err := c.Render(ctx, &buf)
	return buf.String(), err
}
