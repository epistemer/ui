// Package actionstest tests code that resolves actions, with no HTML
// (server-state-interface.md §2.1).
//
//	ctx := actionstest.Context(t, actionstest.Refuse(adminroutes.OrderRefund(o.ID)))
//	actionstest.Assert(t, orderActions(ctx, o), View, Reassign) // Refund is absent
package actionstest

import (
	"context"
	"testing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/internal/links"
)

// Option sets what the test gate refuses.
type Option func(*options)

type options struct{ refused map[string]bool }

// Refuse makes the gate refuse links with these addresses.
func Refuse(ls ...ui.Link) Option {
	return func(o *options) {
		for _, l := range ls {
			o.refused[string(links.Href(l))] = true
		}
	}
}

// Context is a context, from t, whose gate admits every link but those
// Refuse names.
func Context(t testing.TB, opts ...Option) context.Context {
	t.Helper()
	o := &options{refused: map[string]bool{}}
	for _, f := range opts {
		f(o)
	}
	return actions.Gate.With(t.Context(), func(l ui.Link) bool { return !o.refused[string(links.Href(l))] })
}

// Assert fails the test unless items are these actions, in order.
func Assert(t testing.TB, items []actions.Item, want ...actions.Action) {
	t.Helper()
	if len(items) != len(want) {
		t.Errorf("actionstest: %d items, want %d", len(items), len(want))
		return
	}
	for i, w := range want {
		if !items[i].Is(w) {
			t.Errorf("actionstest: item %d (%s) isn't the action wanted", i, items[i].Label.T(context.Background()))
		}
	}
}

// Unavailable is the reason the item bound from a is unavailable: nil
// when it's available or absent.
func Unavailable(items []actions.Item, a actions.Action) error {
	for _, it := range items {
		if it.Is(a) {
			return it.Reason
		}
	}
	return nil
}
