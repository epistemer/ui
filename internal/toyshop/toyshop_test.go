package toyshop

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/actions/actionstest"
	"github.com/epistemer/ui/tables"
	"github.com/epistemer/ui/tables/tablestest"
	"github.com/epistemer/ui/theme"
	"github.com/epistemer/ui/uitest"
)

func gated(ctx context.Context) context.Context { return actions.Gate.With(ctx, Gate) }

// css is ui's stylesheet, for CheckSelectors.
func css(t *testing.T) string {
	t.Helper()
	th, err := theme.New("/ui/", theme.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	head := uitest.Render(t, th.Head())
	href := regexp.MustCompile(`href="(/ui/[^"]+/ui\.css)"`).FindStringSubmatch(head)
	w := httptest.NewRecorder()
	th.ServeHTTP(w, httptest.NewRequest("GET", href[1], nil))
	return w.Body.String()
}

// external are ids the shell renders, outside a table.
var external = uitest.External("ui-action-form", "ui-action-dialog-body", "ui-toasts")

func check(t *testing.T, name, html, styles string) {
	t.Helper()
	uitest.Golden(t, name, html)
	uitest.CheckIDs(t, html, external)
	uitest.CheckNames(t, html)
	uitest.CheckSelectors(t, html, styles)
}

func bind[R any](t *testing.T, tb *tables.Table[R], query string) *tables.State[R] {
	r := httptest.NewRequest("GET", Prefix+"/orders?"+query, nil)
	return tb.Bind(r.WithContext(gated(r.Context())))
}

func TestOrdersTable(t *testing.T) {
	styles := css(t)
	s := NewStore()
	for _, c := range []struct{ name, query string }{
		{"orders-default", ""},
		{"orders-filtered", "q=layla&status=paid&sort=-total&toys.q=r"},
		{"orders-page-2", "page=2&size=25"},
		{"orders-no-match", "q=nobody"},
		{"orders-past-end", "page=9"},
	} {
		t.Run(c.name, func(t *testing.T) {
			st := bind(t, Orders, c.query)
			if !st.Valid() {
				t.Fatalf("%s didn't bind", c.query)
			}
			rows, total := s.ListOrderListView(st)
			html := uitest.Render(t, OrdersView.Page(st.Page(rows, total)), uitest.Context(gated))
			check(t, c.name, html, styles)
		})
	}
	t.Run("orders-rejected", func(t *testing.T) {
		st := bind(t, Orders, "placed_at.from=2026-11-15&placed_at.to=2026-11-02&status=lost")
		if st.Valid() {
			t.Fatal("a reversed range and an unknown status bound")
		}
		html := uitest.Render(t, OrdersView.Page(st.Rejected()), uitest.Context(gated))
		if strings.Contains(html, "<tbody") || strings.Contains(html, "ui-pagination") {
			t.Error("a rejected state shows rows or a pager")
		}
		check(t, "orders-rejected", html, styles)
	})
	t.Run("toys-prefixed", func(t *testing.T) {
		st := bind(t, Toys, "toys.q=r&toys.stock.from=10&toys.electric=true&q=layla")
		rows, total := s.ListToyListView(st)
		html := uitest.Render(t, ToysView.Page(st.Page(rows, total)))
		if !strings.Contains(html, `name="q" value="layla"`) {
			t.Error("the orders' search isn't kept in the toys' state form")
		}
		check(t, "toys-prefixed", html, styles)
	})
}

func TestActions(t *testing.T) {
	s := NewStore()
	paid, _ := s.Order(1001)
	pending, _ := s.Order(1002)
	items := orderActions(actionstest.Context(t, actionstest.Refuse(OrderArchive(1001))), paid)
	actionstest.Assert(t, items, View, Reassign, Refund, Delete)
	items = orderActions(actionstest.Context(t), pending)
	actionstest.Assert(t, items, View, Reassign, Archive, Refund, Delete)
	if actionstest.Unavailable(items, Refund) != ErrNotPaid {
		t.Error("a pending order's refund isn't unavailable by ErrNotPaid")
	}
	tablestest.Check(t, Orders)
	tablestest.Check(t, Toys)
}

func TestOrderCard(t *testing.T) {
	s := NewStore()
	o, _ := s.Order(1002) // pending: refund unavailable
	html := uitest.Render(t, orderPage(o, orderActions(gated(context.Background()), o)), uitest.Context(gated))
	if !strings.Contains(html, `aria-disabled="true"`) || strings.Contains(html, `formaction="/shop/orders/1002/refund"`) {
		t.Error("an unavailable request can still post")
	}
	check(t, "order-card", html, css(t))
}

// server is the shop with stable ids.
func server(t *testing.T) (http.Handler, *Store) {
	th, err := theme.New("/ui/", theme.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app := &App{Store: NewStore(), Theme: th}
	app.Mount(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(ui.WithIDs(r.Context(), ui.Sequence("t"))))
	}), app.Store
}

func do(h http.Handler, method, target, body string, hx map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range hx {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

var hx = map[string]string{"HX-Request": "true"}

func TestAnswers(t *testing.T) {
	h, s := server(t)
	if w := do(h, "POST", "/shop/orders/1001/refund", "", nil); w.Code != 303 || w.Header().Get("Location") != "/shop/orders" {
		t.Errorf("refund without htmx: %d %q", w.Code, w.Header().Get("Location"))
	}
	s.Update(1008, func(o *OrderListView) { o.Status = Paid })
	w := do(h, "POST", "/shop/orders/1008/refund", "", hx)
	body := w.Body.String()
	if w.Code != 200 || w.Header().Get("HX-Retarget") != "#orders-row-1008" || !strings.HasPrefix(body, `<tr id="orders-row-1008"`) || !strings.Contains(body, `hx-swap-oob="beforeend:#ui-toasts"`) {
		t.Errorf("refund with htmx: %d %v\n%s", w.Code, w.Header(), body)
	}
	if w := do(h, "POST", "/shop/orders/1008/refund", "", hx); w.Code != 422 || !strings.Contains(w.Body.String(), `data-tone="danger"`) {
		t.Errorf("a second refund: %d", w.Code)
	}
	if w := do(h, "POST", "/shop/orders/1003/delete", "", hx); w.Header().Get("HX-Reswap") != "delete" || w.Header().Get("HX-Retarget") != "#orders-row-1003" {
		t.Errorf("delete: %v", w.Header())
	}
	if w := do(h, "POST", "/shop/orders/ship", "selected=1004&selected=1006", hx); !strings.Contains(w.Header().Get("HX-Trigger-After-Settle"), `"ui-table-refresh":{"id":"orders"}`) {
		t.Errorf("ship: %d %v", w.Code, w.Header())
	}
	dlg := map[string]string{"HX-Request": "true", "HX-Target": "ui-action-dialog-body"}
	if w := do(h, "GET", "/shop/orders/1002/reassign", "", dlg); w.Code != 200 || strings.Contains(w.Body.String(), "<html") || !strings.Contains(w.Body.String(), `id="ui-action-dialog-title"`) {
		t.Errorf("the dialog's form: %d", w.Code)
	}
	if w := do(h, "POST", "/shop/orders/1002/reassign", "assignee=", dlg); w.Code != 422 || strings.Contains(w.Body.String(), "<html") {
		t.Errorf("a failed form in the dialog: %d", w.Code)
	}
	if w := do(h, "POST", "/shop/orders/1002/reassign", "assignee=Mona", dlg); w.Code != 200 || !strings.Contains(w.Header().Get("HX-Trigger-After-Settle"), "ui-table-refresh") {
		t.Errorf("reassigned: %d %v", w.Code, w.Header())
	}
	if w := do(h, "POST", "/shop/orders/1002/reassign", "assignee=", nil); w.Code != 422 || !strings.Contains(w.Body.String(), "<html") {
		t.Errorf("a failed form without htmx: %d", w.Code)
	}
	if w := do(h, "GET", "/shop/orders/tag?selected=1001&selected=1002", "", nil); !strings.Contains(w.Body.String(), `name="selected" value="1002"`) {
		t.Error("the tag page lost the selection")
	}
	if w := do(h, "GET", "/shop/orders?placed_at.from=x", "", nil); w.Code != 422 {
		t.Errorf("a junk date: %d", w.Code)
	}
	w = do(h, "GET", "/shop/orders?q=a&placed_at.from=x", "", map[string]string{"HX-Request": "true", "HX-Target": "orders-results"})
	if w.Code != 422 || w.Header().Get("HX-Retarget") != "#orders" || w.Header().Get("HX-Reselect") != "#orders" {
		t.Errorf("a rejected search: %d %v", w.Code, w.Header())
	}
}

func TestOrdersPage(t *testing.T) {
	h, _ := server(t)
	html := do(h, "GET", "/shop/orders", "", nil).Body.String()
	uitest.CheckIDs(t, html)
	uitest.CheckNames(t, html)
	uitest.CheckSelectors(t, html, css(t))
}
