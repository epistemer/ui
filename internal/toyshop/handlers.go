package toyshop

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
	"github.com/epistemer/ui/theme"
)

// App serves the shop.
type App struct {
	Store *Store
	Theme *theme.Theme
	// Scripts are what a page loads beside the theme: htmx, with the
	// config tables want (historyCacheSize 0). Nil serves pages without.
	Scripts templ.Component
}

// Mount registers the shop's routes on mux, under Prefix.
func (a *App) Mount(mux *http.ServeMux) {
	h := func(f func(http.ResponseWriter, *http.Request) error) http.Handler {
		return a.gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := f(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}))
	}
	mux.Handle("GET "+Prefix+"/orders", h(a.orders))
	mux.Handle("GET "+Prefix+"/orders/{id}", h(a.order))
	mux.Handle("POST "+Prefix+"/orders/{id}/refund", h(a.refund))
	mux.Handle("POST "+Prefix+"/orders/{id}/delete", h(a.delete))
	mux.Handle("GET "+Prefix+"/orders/{id}/reassign", h(a.reassign))
	mux.Handle("POST "+Prefix+"/orders/{id}/reassign", h(a.reassign))
	mux.Handle("POST "+Prefix+"/orders/{id}/archive", h(func(w http.ResponseWriter, r *http.Request) error {
		http.Error(w, "forbidden", http.StatusForbidden) // the guard's answer
		return nil
	}))
	mux.Handle("POST "+Prefix+"/orders/ship", h(a.ship))
	mux.Handle("GET "+Prefix+"/orders/tag", h(a.tag))
	mux.Handle("POST "+Prefix+"/orders/tag", h(a.tag))
	mux.Handle("GET "+Prefix+"/customers/{id}", h(a.customer))
}

// gate sets the request's actions gate, as a middleware from the guard
// would.
func (a *App) gate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(actions.Gate.With(r.Context(), Gate)))
	})
}

func (a *App) page(w http.ResponseWriter, r *http.Request, status int, title i18n.Ter, body templ.Component) error {
	var b bytes.Buffer
	if err := pageTmpl(a.Theme, a.Scripts, title, body).Render(r.Context(), &b); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(b.Bytes())
	return err
}

func fragment(w http.ResponseWriter, r *http.Request, status int, c templ.Component) error {
	var b bytes.Buffer
	if err := c.Render(r.Context(), &b); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(b.Bytes())
	return err
}

var (
	ordersTitle   = i18n.Key("page.orders.title", "Orders")
	refunded      = i18n.Key("page.orders.refunded", "The order was refunded.")
	deleted       = i18n.Key("page.orders.deleted", "The order was deleted.")
	reassigned    = i18n.Key("page.orders.reassigned", "The order was reassigned.")
	shipped       = i18n.Key("page.orders.shipped", "The orders were marked shipped.")
	tagged        = i18n.Key("page.orders.tagged", "The orders were tagged.")
	assigneeLabel = i18n.Key("page.orders.assignee", "Assignee")
	tagLabel      = i18n.Key("page.orders.tag_name", "Tag")
)

func (a *App) orders(w http.ResponseWriter, r *http.Request) error {
	st, ts := Orders.Bind(r), Toys.Bind(r)
	ov, tv := st.Rejected(), ts.Rejected()
	if st.Valid() {
		rows, total := a.Store.ListOrderListView(st)
		ov = st.Page(rows, total)
	}
	if ts.Valid() {
		rows, total := a.Store.ListToyListView(ts)
		tv = ts.Page(rows, total)
	}
	status := max(st.Answer(w, r), ts.Answer(w, r))
	return a.page(w, r, status, ordersTitle, ordersPage(OrdersView.Page(ov), ToysView.Page(tv)))
}

func id(r *http.Request) int64 {
	n, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return n
}

func (a *App) order(w http.ResponseWriter, r *http.Request) error {
	o, ok := a.Store.Order(id(r))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	return a.page(w, r, http.StatusOK, i18n.Static(o.Email), orderPage(o, orderActions(r.Context(), o)))
}

func (a *App) customer(w http.ResponseWriter, r *http.Request) error {
	return a.page(w, r, http.StatusOK, i18n.Static("Customer "+r.PathValue("id")), customerPage(r.PathValue("id")))
}

func (a *App) refund(w http.ResponseWriter, r *http.Request) error {
	o, ok := a.Store.Order(id(r))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	if err := CanRefund(o); err != nil {
		return components.Reply(w, r).Fail(err).Else(OrdersRoute())
	}
	o, _ = a.Store.Update(o.ID, func(o *OrderListView) { o.Status = Refunded })
	return components.Reply(w, r).Toast(refunded).Replace(Orders.RowID(strconv.FormatInt(o.ID, 10)), OrdersView.Row(o)).Else(OrdersRoute())
}

func (a *App) delete(w http.ResponseWriter, r *http.Request) error {
	if !a.Store.Delete(id(r)) {
		http.NotFound(w, r)
		return nil
	}
	return components.Reply(w, r).Toast(deleted).Remove(Orders.RowID(r.PathValue("id"))).Else(OrdersRoute())
}

func reassignForm() (*forms.Form, *forms.Field[string]) {
	f := forms.New()
	return f, f.Text("assignee").Label(assigneeLabel).Required().Is(rule.MinLen(2))
}

func (a *App) reassign(w http.ResponseWriter, r *http.Request) error {
	o, ok := a.Store.Order(id(r))
	if !ok {
		http.NotFound(w, r)
		return nil
	}
	f, who := reassignForm()
	status := http.StatusOK
	if r.Method == http.MethodPost {
		if f.Submit(r.Context(), r) {
			a.Store.Update(o.ID, func(o *OrderListView) { o.Assignee = who.Value() })
			return components.Reply(w, r).Toast(reassigned).Refresh(Orders.ID()).Else(OrdersRoute())
		}
		status = http.StatusUnprocessableEntity
	} else if o.Assignee != "" {
		who.Set(o.Assignee)
	}
	body := reassignBody(o, f)
	if components.InActionDialog(r) {
		return fragment(w, r, status, body)
	}
	return a.page(w, r, status, i18n.Static("Reassign"), body)
}

// keys reads the selection's keys as order ids.
func keys(r *http.Request) []int64 {
	var out []int64
	for _, s := range Orders.Selected(r) {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (a *App) ship(w http.ResponseWriter, r *http.Request) error {
	chosen := keys(r)
	if len(chosen) == 0 {
		return components.Reply(w, r).Fail(errNothing).Else(OrdersRoute())
	}
	for _, n := range chosen {
		if o, ok := a.Store.Order(n); !ok || o.Status == Refunded {
			// All or nothing: one refused order refuses the request.
			return components.Reply(w, r).Fail(errCantShip).Else(OrdersRoute())
		}
	}
	for _, n := range chosen {
		a.Store.Update(n, func(o *OrderListView) { o.Status = Shipped })
	}
	return components.Reply(w, r).Toast(shipped).Refresh(Orders.ID()).Else(OrdersRoute())
}

func tagForm() (*forms.Form, *forms.Field[string]) {
	f := forms.New()
	return f, f.Text("tag").Label(tagLabel).Required()
}

func (a *App) tag(w http.ResponseWriter, r *http.Request) error {
	chosen := keys(r)
	f, name := tagForm()
	status := http.StatusOK
	if r.Method == http.MethodPost {
		if f.Submit(r.Context(), r) && len(chosen) > 0 {
			for _, n := range chosen {
				a.Store.Update(n, func(o *OrderListView) { o.Tags = append(o.Tags, name.Value()) })
			}
			return components.Reply(w, r).Toast(tagged).Refresh(Orders.ID()).Else(OrdersRoute())
		}
		status = http.StatusUnprocessableEntity
	}
	body := tagBody(chosen, f)
	if components.InActionDialog(r) {
		return fragment(w, r, status, body)
	}
	return a.page(w, r, status, i18n.Static("Tag"), body)
}
