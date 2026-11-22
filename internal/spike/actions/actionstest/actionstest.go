// Package actionstest is the spike of ui/actions/actionstest (§5.7).
package actionstest

import (
	"context"
	"testing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/links"
	"github.com/epistemer/ui/internal/spike/actions"
)

// Option sets what the test gate refuses.
type Option func(map[string]bool)

// Refuse makes the gate refuse links with these addresses.
func Refuse(ls ...ui.Link) Option {
	return func(m map[string]bool) {
		for _, l := range ls {
			m[string(links.Href(l))] = true
		}
	}
}

// Context is a request context whose gate admits everything but what
// Refuse names.
func Context(t *testing.T, opts ...Option) context.Context {
	t.Helper()
	refused := map[string]bool{}
	for _, o := range opts {
		o(refused)
	}
	return actions.Gate.With(t.Context(), func(l ui.Link) bool { return !refused[string(links.Href(l))] })
}

// Assert checks the items are these actions, in order.
func Assert(t *testing.T, items []actions.Item, want ...actions.Action) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("actionstest: %d items, want %d", len(items), len(want))
	}
	for i, w := range want {
		if !items[i].Is(w) {
			t.Errorf("actionstest: item %d isn't the action wanted", i)
		}
	}
}

// Disabled is the reason the item from a is disabled, or nil.
func Disabled(items []actions.Item, a actions.Action) error {
	for _, it := range items {
		if it.Is(a) {
			return it.Reason
		}
	}
	return nil
}
