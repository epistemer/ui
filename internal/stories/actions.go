package stories

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
	"github.com/epistemer/ui/icons"
)

// demoRoute stands in for a handlers link: an address and a method.
type demoRoute struct{ url, method string }

func (r demoRoute) URL() string    { return r.url }
func (r demoRoute) Method() string { return r.method }

var errNotPaid = errors.Key("errors.shop.not_paid", "only a paid order can be refunded", errors.Conflict)

var (
	actView     = actions.New(i18n.Static("View")).Icon(icons.Eye())
	actEdit     = actions.New(i18n.Static("Edit")).Primary()
	actReassign = actions.New(i18n.Static("Reassign")).Dialog()
	actRefund   = actions.New(i18n.Static("Refund")).Danger().Confirm(i18n.Static("Refund this order?"))
	actDelete   = actions.New(i18n.Static("Delete")).Danger().Confirm(i18n.Static("Delete this order?"))
)

// orderItems are an order's actions, resolved with a gate that admits
// everything; paid decides whether Refund is available.
func orderItems(ctx context.Context, paid bool) []actions.Item {
	ctx = actions.Gate.With(ctx, actions.AllowAll)
	var reason error
	if !paid {
		reason = errNotPaid
	}
	return actions.Resolve(ctx,
		actView.To(ui.URL("/orders/1042")),
		actReassign.To(demoRoute{"/demo/reassign", "GET"}),
		actRefund.To(demoRoute{"/demo/refund", "POST"}).Unless(reason),
		actDelete.To(demoRoute{"/demo/fail", "POST"}),
	)
}

// withItems renders c with the order's items, resolved in the render's
// context.
func withItems(paid bool, c func([]actions.Item) templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c(orderItems(ctx, paid)).Render(ctx, w)
	})
}

func editItems(ctx context.Context) []actions.Item {
	ctx = actions.Gate.With(ctx, actions.AllowAll)
	return actions.Resolve(ctx, actEdit.To(ui.URL("/orders/1042/edit")), actView.To(ui.URL("/orders/1042")))
}

var assigneeLabel = i18n.Static("Assignee")

func reassignForm() (*forms.Form, *forms.Field[string]) {
	f := forms.New()
	return f, f.Text("assignee").Label(assigneeLabel).Required().Is(rule.MinLen(2))
}

// Demo registers what the action stories ask for on mux: the reassign
// form (an Open action, failing once with a 422), a refund (a request
// answered with a toast), and a refusal (Reply.Fail).
func Demo(mux *http.ServeMux) {
	mux.HandleFunc("GET /demo/reassign", func(w http.ResponseWriter, r *http.Request) {
		f, _ := reassignForm()
		reassignPage(f, components.InActionDialog(r)).Render(r.Context(), w)
	})
	mux.HandleFunc("POST /demo/reassign", func(w http.ResponseWriter, r *http.Request) {
		f, who := reassignForm()
		if !f.Submit(r.Context(), r) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			reassignPage(f, components.InActionDialog(r)).Render(r.Context(), w)
			return
		}
		components.Reply(w, r).Toast(i18n.Static("Reassigned to " + who.Value() + ".")).Else(ui.URL("/"))
	})
	mux.HandleFunc("POST /demo/refund", func(w http.ResponseWriter, r *http.Request) {
		components.Reply(w, r).Toast(i18n.Static("The order was refunded.")).Else(ui.URL("/"))
	})
	mux.HandleFunc("POST /demo/fail", func(w http.ResponseWriter, r *http.Request) {
		components.Reply(w, r).Fail(errNotPaid).Else(ui.URL("/"))
	})
}
