package ui

import (
	"context"

	"github.com/a-h/templ"
)

// block returns the block a kernel component was called with and a context
// with the children cleared. Every kernel component that renders its block
// calls it first (K3): otherwise a component inside the block that renders
// children but was called without a block of its own would render this
// block again.
func block(ctx context.Context) (templ.Component, context.Context) {
	return templ.GetChildren(ctx), templ.ClearChildren(ctx)
}
