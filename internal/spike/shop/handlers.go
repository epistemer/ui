package shop

import (
	"bytes"
	"context"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/tableui"
	"github.com/epistemer/ui/theme"
)

// App serves the shop.
type App struct {
	Store *Store
	Theme *theme.Theme
}

// Mount registers the shop's routes on mux.
func (a *App) Mount(mux *http.ServeMux) {
	h := func(f func(http.ResponseWriter, *http.Request) error) http.Handler {
		return a.gate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := f(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}))
	}
	mux.Handle("GET /orders", h(a.orders))
	mux.Handle("GET /orders/{id}", h(a.order))
	mux.Handle("POST /orders/{id}/refund", h(a.refund))
	mux.Handle("POST /orders/{id}/delete", h(a.delete))
	mux.Handle("GET /orders/{id}/reassign", h(a.reassign))
	mux.Handle("POST /orders/{id}/reassign", h(a.reassign))
	mux.Handle("POST /orders/{id}/archive", h(func(w http.ResponseWriter, r *http.Request) error {
		http.Error(w, "forbidden", http.StatusForbidden) // the guard's answer
		return nil
	}))
	mux.Handle("POST /orders/ship", h(a.ship))
	mux.Handle("GET /orders/tag", h(a.tag))
	mux.Handle("POST /orders/tag", h(a.tag))
	mux.Handle("GET /customers/{id}", h(a.customer))
	mux.HandleFunc("GET /plain", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "spike-plain", Value: "1", Path: "/"})
		http.Redirect(w, r, "/orders", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /htmx", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "spike-plain", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/orders", http.StatusSeeOther)
	})
}

// gate sets the request's actions gate, as a middleware from the guard
// would.
func (a *App) gate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(actions.Gate.With(r.Context(), Gate)))
	})
}

func (a *App) page(w http.ResponseWriter, r *http.Request, status int, title i18n.Ter, body templ.Component) error {
	_, noPlain := r.Cookie("spike-plain")
	var b bytes.Buffer
	if err := pageTmpl(a.Theme, title, noPlain != nil, body).Render(r.Context(), &b); err != nil {
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

// failure answers a state rule's refusal: a toast with htmx, the page
// again without. Reply has no such ending yet (a finding).
func failure(w http.ResponseWriter, r *http.Request, err error) error {
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/orders", http.StatusSeeOther)
		return nil
	}
	w.Header().Set("HX-Reswap", "none")
	return fragment(w, r, http.StatusUnprocessableEntity, components.ToastSwap(components.Toast(errText(err)).Tone(elements.Danger)))
}

type errTer struct{ err error }

func (e errTer) T(ctx context.Context, _ ...any) string { return errorsT(ctx, e.err) }

func errText(err error) i18n.Ter { return errTer{err} }

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
	st := Orders.Bind(r)
	ts := Toys.Bind(r)
	if !st.Valid() || !ts.Valid() {
		// A request that swaps only the results (the search box) can't
		// show a filter's error, which sits in the state form: the
		// rejection takes the whole table (review 0014, finding 2).
		for _, t := range []struct {
			valid      bool
			id, result string
		}{{st.Valid(), Orders.ID(), Orders.ResultsID()}, {ts.Valid(), Toys.ID(), Toys.ResultsID()}} {
			if !t.valid && r.Header.Get("HX-Target") == t.result {
				w.Header().Set("HX-Retarget", "#"+t.id)
				w.Header().Set("HX-Reselect", "#"+t.id)
			}
		}
		ov, tv := st.Rejected(), ts.Rejected()
		if st.Valid() {
			rows, total := a.Store.ListOrderListView(st)
			ov = st.Page(rows, total)
		}
		if ts.Valid() {
			rows, total := a.Store.ListToyListView(ts)
			tv = ts.Page(rows, total)
		}
		return a.page(w, r, http.StatusUnprocessableEntity, ordersTitle, ordersPage(OrdersView.Page(ov), ToysView.Page(tv)))
	}
	rows, total := a.Store.ListOrderListView(st)
	trows, ttotal := a.Store.ListToyListView(ts)
	return a.page(w, r, http.StatusOK, ordersTitle, ordersPage(OrdersView.Page(st.Page(rows, total)), ToysView.Page(ts.Page(trows, ttotal))))
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
		return failure(w, r, err)
	}
	o, _ = a.Store.Update(o.ID, func(o *OrderListView) { o.Status = Refunded })
	return tableui.Reply(w, r).Toast(refunded).Replace(Orders.RowID(strconv.FormatInt(o.ID, 10)), OrdersView.Row(r.Context(), o)).Else(OrdersRoute())
}

func (a *App) delete(w http.ResponseWriter, r *http.Request) error {
	if !a.Store.Delete(id(r)) {
		http.NotFound(w, r)
		return nil
	}
	return tableui.Reply(w, r).Toast(deleted).Remove(Orders.RowID(r.PathValue("id"))).Else(OrdersRoute())
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
			return tableui.Reply(w, r).Toast(reassigned).Refresh(Orders.ID()).Else(OrdersRoute())
		}
		status = http.StatusUnprocessableEntity
	} else if o.Assignee != "" {
		who.Set(o.Assignee)
	}
	body := reassignBody(o, f)
	if tableui.InActionDialog(r) {
		return fragment(w, r, status, body)
	}
	return a.page(w, r, status, i18n.Static("Reassign"), body)
}

func ids(r *http.Request) []int64 {
	_ = r.ParseForm()
	vals := r.Form["id"] // a GET's query
	if r.Method == http.MethodPost {
		vals = r.PostForm["id"] // the body alone, never the query
	}
	var out []int64
	for _, s := range vals {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (a *App) ship(w http.ResponseWriter, r *http.Request) error {
	chosen := ids(r)
	if len(chosen) == 0 {
		return failure(w, r, errNothing)
	}
	for _, n := range chosen {
		if o, ok := a.Store.Order(n); !ok || o.Status == Refunded {
			return failure(w, r, errCantShip)
		}
	}
	for _, n := range chosen {
		a.Store.Update(n, func(o *OrderListView) { o.Status = Shipped })
	}
	return tableui.Reply(w, r).Toast(shipped).Refresh(Orders.ID()).Else(OrdersRoute())
}

func tagForm() (*forms.Form, *forms.Field[string]) {
	f := forms.New()
	return f, f.Text("tag").Label(tagLabel).Required()
}

func (a *App) tag(w http.ResponseWriter, r *http.Request) error {
	chosen := ids(r)
	f, name := tagForm()
	status := http.StatusOK
	if r.Method == http.MethodPost {
		if f.Submit(r.Context(), r) && len(chosen) > 0 {
			for _, n := range chosen {
				a.Store.Update(n, func(o *OrderListView) { o.Tags = append(o.Tags, name.Value()) })
			}
			return tableui.Reply(w, r).Toast(tagged).Refresh(Orders.ID()).Else(OrdersRoute())
		}
		status = http.StatusUnprocessableEntity
	}
	body := tagBody(chosen, f)
	if tableui.InActionDialog(r) {
		return fragment(w, r, status, body)
	}
	return a.page(w, r, status, i18n.Static("Tag"), body)
}
