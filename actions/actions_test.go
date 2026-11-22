package actions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/actions/actionstest"
)

type route struct{ url, method string }

func (r route) URL() string    { return r.url }
func (r route) Method() string { return r.method }

var errNotPaid = errors.Key("errors.shop.not_paid", "only a paid order can be refunded", errors.Conflict)

var (
	view     = actions.New(i18n.Static("View"))
	refund   = actions.New(i18n.Static("Refund")).Danger().Confirm(i18n.Static("Refund it?"))
	reassign = actions.New(i18n.Static("Reassign")).Dialog()
	archive  = actions.New(i18n.Static("Archive"))
)

func bound(paid bool) []actions.Bound {
	var reason error
	if !paid {
		reason = errNotPaid
	}
	return []actions.Bound{
		view.To(route{"/orders/7", "GET"}),
		reassign.To(route{"/orders/7/reassign", "GET"}),
		archive.To(route{"/orders/7/archive", "POST"}),
		refund.To(route{"/orders/7/refund", "POST"}).Unless(reason),
	}
}

func TestResolve(t *testing.T) {
	ctx := actionstest.Context(t, actionstest.Refuse(route{"/orders/7/archive", "POST"}))
	items := actions.Resolve(ctx, bound(false)...)
	actionstest.Assert(t, items, view, reassign, refund)
	want := []actions.Kind{actions.Navigate, actions.Open, actions.Request}
	for i, it := range items {
		if it.Kind != want[i] {
			t.Errorf("item %d: kind %v, want %v", i, it.Kind, want[i])
		}
	}
	if items[2].Method != "POST" || !items[2].Danger || items[2].Question == nil {
		t.Errorf("refund: %+v", items[2])
	}
	if actionstest.Unavailable(items, refund) != errNotPaid || actionstest.Unavailable(items, view) != nil {
		t.Error("unavailable")
	}
	items = actions.Resolve(actionstest.Context(t), bound(true)...)
	actionstest.Assert(t, items, view, reassign, archive, refund)
	if got := actions.Resolve(actionstest.Context(t)); got == nil || len(got) != 0 {
		t.Errorf("no bound: %v", got)
	}
}

func TestAbsent(t *testing.T) {
	ctx := actionstest.Context(t)
	items := actions.Resolve(ctx,
		view.To(nil),
		view.To(ui.URL("javascript:alert(1)")),
		view.To(ui.URL("/x")).If(false),
		archive.To(route{"https://other.example/archive", "POST"}), // a request elsewhere
		reassign.To(ui.URL("//other.example/x")),                   // an Open elsewhere
		view.To(ui.URL("https://docs.example/")),                   // a plain link elsewhere: kept
	)
	if len(items) != 1 || items[0].Kind != actions.Navigate {
		t.Errorf("absent: %+v", items)
	}
}

func TestIdentity(t *testing.T) {
	base := actions.New(i18n.Static("Go"))
	a, b := base.Primary(), base.Icon(nil)
	items := actions.Resolve(actionstest.Context(t), a.To(ui.URL("/a")))
	if !items[0].Is(a) || items[0].Is(b) || items[0].Is(base) {
		t.Error("two actions from one base are one")
	}
	if (actions.Item{}).Is(base) {
		t.Error("a literal item is an action's")
	}
}

func TestPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"a blank label":        func() { actions.New(i18n.Static(" ")) },
		"a blank question":     func() { actions.New(i18n.Static("x")).Confirm(i18n.Static("")) },
		"Dialog then Confirm":  func() { actions.New(i18n.Static("x")).Dialog().Confirm(i18n.Static("q")) },
		"Confirm then Dialog":  func() { actions.New(i18n.Static("x")).Confirm(i18n.Static("q")).Dialog() },
		"Primary and Danger":   func() { actions.New(i18n.Static("x")).Primary().Danger() },
		"Danger and Primary":   func() { actions.New(i18n.Static("x")).Danger().Primary() },
		"no gate":              func() { actions.Resolve(context.Background(), view.To(ui.URL("/x"))) },
		"a nil gate":           func() { actions.Resolve(actions.Gate.With(context.Background(), nil)) },
		"Allowed with no gate": func() { actions.Allowed(context.Background(), ui.URL("/x")) },
		"a Dialog on a POST": func() {
			actions.Resolve(actionstest.Context(t), reassign.To(route{"/x", "POST"}))
		},
	} {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("%s didn't panic", name)
				} else if !strings.HasPrefix(r.(string), "actions: ") {
					t.Errorf("%s: %v", name, r)
				}
			}()
			f()
		}()
	}
}

func TestAllowAll(t *testing.T) {
	ctx := actions.Gate.With(context.Background(), actions.AllowAll)
	if !actions.Allowed(ctx, ui.URL("/x")) || len(actions.Resolve(ctx, bound(true)...)) != 4 {
		t.Error("AllowAll")
	}
}
