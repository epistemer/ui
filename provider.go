package ui

import (
	"context"
	"io"

	"github.com/a-h/templ"
)

// Provider passes a typed value down a subtree, so a component can read it
// without every component in between taking it as a parameter: a currency,
// a form's id, the signed-in user.
//
// A Provider is a handle: declare it once in a package-level variable.
// Values live in the context, keyed by the handle (K2, K8), so two
// providers of the same type never see each other's values.
type Provider[T any] struct {
	def T
	_   byte // keeps every Provider distinct in memory, whatever T's size
}

// NewProvider declares a provider whose Value is def wherever nothing was
// provided.
func NewProvider[T any](def T) *Provider[T] {
	return &Provider[T]{def: def}
}

// Provide renders its block with v as the provider's value:
//
//	@Currency.Provide("EGP") {
//		@Price(p)
//	}
func (p *Provider[T]) Provide(v T) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children, ctx := block(ctx)
		if err := inBlock(ctx, "ui.Provider.Provide"); err != nil {
			return err
		}
		return children.Render(p.With(ctx, v), w)
	})
}

// With returns a context in which v is the provider's value, for Go code
// such as middleware and handlers.
func (p *Provider[T]) With(ctx context.Context, v T) context.Context {
	return context.WithValue(ctx, p, provided[T]{v})
}

// provided boxes a value, so a nil interface value that was provided is
// told apart from no value at all.
type provided[T any] struct{ v T }

// Value returns the nearest provided value, else the provider's default.
func (p *Provider[T]) Value(ctx context.Context) T {
	if b, ok := ctx.Value(p).(provided[T]); ok {
		return b.v
	}
	return p.def
}
