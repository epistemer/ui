package shop

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/actions/actionstest"
	"github.com/epistemer/ui/internal/spike/assets"
	"github.com/epistemer/ui/internal/spike/col"
	"github.com/epistemer/ui/internal/spike/tables"
	"github.com/epistemer/ui/internal/spike/tables/tablestest"
	"github.com/epistemer/ui/internal/spike/tableui"
	"github.com/epistemer/ui/theme"
	"github.com/epistemer/ui/uitest"
)

func gated(ctx context.Context) context.Context { return actions.Gate.With(ctx, Gate) }

// css is ui's stylesheet and the spike's, for CheckSelectors.
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
	spike, err := assets.FS.ReadFile("spike.css")
	if err != nil {
		t.Fatal(err)
	}
	return w.Body.String() + string(spike)
}

// external are ids the page's shell renders, outside a table.
var external = uitest.External(tableui.ActionFormID, tableui.ActionDialogBodyID, "ui-toasts")

func check(t *testing.T, name, html, styles string) {
	t.Helper()
	uitest.Golden(t, name, html)
	uitest.CheckIDs(t, html, external)
	uitest.CheckNames(t, html)
	uitest.CheckSelectors(t, html, styles)
}

func bind[R any](t *testing.T, tb *tables.Table[R], query string) *tables.State[R] {
	r := httptest.NewRequest("GET", "/orders?"+query, nil)
	return tb.Bind(r.WithContext(gated(r.Context())))
}

func TestOrdersTable(t *testing.T) {
	styles := css(t)
	s := NewStore()
	for _, c := range []struct{ name, query string }{
		{"orders-default", ""},
		{"orders-filtered", "q=layla&status=paid&sort=-total&page=1&toys.q=r"},
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
		got := tablestest.Errors(st)
		want := []string{"status: errors.forms.not_an_option", "placed_at: errors.tables.range"}
		if strings.Join(got, ";") != strings.Join(want, ";") {
			t.Errorf("errors %q, want %q", got, want)
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

func TestQuery(t *testing.T) {
	st := tablestest.Bind(t, Orders, "q=ann&status=paid&placed_at.from=2026-11-03&sort=-total&page=2&size=25")
	q := tablestest.Query(st)
	if q.Search != "ann" || strings.Join(q.SearchIn, ",") != "email,customer" {
		t.Errorf("search %q in %v", q.Search, q.SearchIn)
	}
	if len(q.Filters) != 2 || q.Filters[0].Column != "status" || q.Filters[1].Column != "placed_at" {
		t.Errorf("filters %v", q.Filters)
	}
	if len(q.Sort) != 2 || q.Sort[0].Column != "total" || !q.Sort[0].Desc || q.Sort[1].Column != "id" || !q.Sort[1].Desc {
		t.Errorf("sort %v: the key must end it, in its direction", q.Sort)
	}
	if q.Limit != 25 || q.Offset != 25 {
		t.Errorf("limit %d offset %d", q.Limit, q.Offset)
	}
	// Junk in what links write is normalised without a word.
	q = tablestest.Query(tablestest.Bind(t, Orders, "sort=status&page=-3&size=7"))
	if q.Sort[0].Column != "placed_at" || q.Offset != 0 || q.Limit != 10 {
		t.Errorf("junk: sort %v offset %d limit %d", q.Sort, q.Offset, q.Limit)
	}
	tablestest.Check(t, Orders)
	tablestest.Check(t, Toys)
}

func TestActions(t *testing.T) {
	s := NewStore()
	paid, _ := s.Order(1001)
	pending, _ := s.Order(1002)
	items := orderActions(actionstest.Context(t, actionstest.Refuse(OrderArchive(1001))), paid)
	actionstest.Assert(t, items, View, Reassign, Refund, Delete)
	if actionstest.Disabled(items, Refund) != nil {
		t.Error("a paid order's refund is disabled")
	}
	items = orderActions(actionstest.Context(t), pending)
	actionstest.Assert(t, items, View, Reassign, Archive, Refund, Delete)
	if actionstest.Disabled(items, Refund) != ErrNotPaid {
		t.Error("a pending order's refund isn't disabled by ErrNotPaid")
	}
	for _, it := range items {
		want := map[string]actions.Kind{"View": actions.Navigate, "Reassign": actions.Open, "Archive": actions.Request, "Refund": actions.Request, "Delete": actions.Request}[it.Label.T(context.Background())]
		if it.Kind != want {
			t.Errorf("%s: kind %v, want %v", it.Label.T(context.Background()), it.Kind, want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("Resolve without a gate didn't panic")
		}
	}()
	actions.Resolve(context.Background(), View.To(OrderRoute(1)))
}

func TestPlacements(t *testing.T) {
	styles := css(t)
	s := NewStore()
	o, _ := s.Order(1002) // pending: refund disabled
	ctx := gated(context.Background())
	html := uitest.Render(t, orderPage(o, orderActions(ctx, o)), uitest.Context(gated))
	if !strings.Contains(html, `aria-disabled="true"`) || strings.Contains(html, `formaction="/orders/1002/refund"`) {
		t.Error("a disabled request can still post")
	}
	check(t, "order-card", html, styles)
	if html := uitest.Render(t, tableui.ActionButtons(nil)); strings.TrimSpace(html) != "" {
		t.Errorf("no items rendered %q", html)
	}
}

// server is the shop with stable ids, as a handler would get them.
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

func TestReply(t *testing.T) {
	h, s := server(t)
	// Without script: a redirect to Else.
	if w := do(h, "POST", "/orders/1001/refund", "", nil); w.Code != 303 || w.Header().Get("Location") != "/orders" {
		t.Errorf("refund without htmx: %d %q", w.Code, w.Header().Get("Location"))
	}
	// With htmx: the row, retargeted, and a toast out of band.
	s.Update(1008, func(o *OrderListView) { o.Status = Paid })
	w := do(h, "POST", "/orders/1008/refund", "", hx)
	body := w.Body.String()
	if w.Code != 200 || w.Header().Get("HX-Retarget") != "#orders-row-1008" || w.Header().Get("HX-Reswap") != "outerHTML" ||
		!strings.HasPrefix(body, `<tr id="orders-row-1008"`) || !strings.Contains(body, `hx-swap-oob="beforeend:#ui-toasts"`) {
		t.Errorf("refund with htmx: %d %v\n%s", w.Code, w.Header(), body)
	}
	if !strings.Contains(w.Header().Get("HX-Trigger-After-Settle"), "ui-action-done") {
		t.Error("no close event after settling")
	}
	// A state rule refuses: 422 and a danger toast.
	if w := do(h, "POST", "/orders/1008/refund", "", hx); w.Code != 422 || !strings.Contains(w.Body.String(), `data-tone="danger"`) {
		t.Errorf("a second refund: %d", w.Code)
	}
	if w := do(h, "POST", "/orders/1003/delete", "", hx); w.Header().Get("HX-Reswap") != "delete" || w.Header().Get("HX-Retarget") != "#orders-row-1003" {
		t.Errorf("delete: %v", w.Header())
	}
	if w := do(h, "POST", "/orders/ship", "id=1004&id=1006", hx); !strings.Contains(w.Header().Get("HX-Trigger-After-Settle"), `"ui-table-refresh":{"id":"orders"}`) {
		t.Errorf("ship: %v", w.Header())
	}
	// The action dialog: the form alone, a 422 kept open, then success.
	dlg := map[string]string{"HX-Request": "true", "HX-Target": tableui.ActionDialogBodyID}
	if w := do(h, "GET", "/orders/1002/reassign", "", dlg); w.Code != 200 || strings.Contains(w.Body.String(), "<html") {
		t.Errorf("the dialog's form came as a page: %d", w.Code)
	}
	if w := do(h, "POST", "/orders/1002/reassign", "assignee=", dlg); w.Code != 422 || strings.Contains(w.Body.String(), "<html") {
		t.Errorf("a failed form in the dialog: %d", w.Code)
	}
	if w := do(h, "POST", "/orders/1002/reassign", "assignee=Mona", dlg); w.Code != 200 || !strings.Contains(w.Header().Get("HX-Trigger-After-Settle"), "ui-table-refresh") {
		t.Errorf("reassigned: %d %v", w.Code, w.Header())
	}
	// Without script, the same form is a page, and a failure is a 422 page.
	if w := do(h, "POST", "/orders/1002/reassign", "assignee=", nil); w.Code != 422 || !strings.Contains(w.Body.String(), "<html") {
		t.Errorf("a failed form without htmx: %d", w.Code)
	}
	// Bulk Open without script: a GET page holding the ids.
	if w := do(h, "GET", "/orders/tag?id=1001&id=1002", "", nil); !strings.Contains(w.Body.String(), `name="id" value="1002"`) {
		t.Error("the tag page lost the selection")
	}
	// A filter that doesn't parse: 422 and the page.
	if w := do(h, "GET", "/orders?placed_at.from=x", "", nil); w.Code != 422 {
		t.Errorf("a junk date: %d", w.Code)
	}
}

func TestElseIsLocal(t *testing.T) {
	for _, u := range []string{"https://evil.example/", "//evil.example/", "/\\evil.example", "/\t/evil.example", "/\n/evil.example", "/ /evil.example"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		if err := tableui.Reply(w, r).Else(ui.URL(u)); err != tableui.ErrNotLocal {
			t.Errorf("Else(%q): %v", u, err)
		}
	}
}

// The page renders whole, with every check.
func TestOrdersPage(t *testing.T) {
	h, _ := server(t)
	w := do(h, "GET", "/orders", "", nil)
	body, _ := io.ReadAll(w.Body)
	html := string(body)
	uitest.CheckIDs(t, html)
	uitest.CheckNames(t, html)
}

// A search while a filter's value is rejected takes the whole table, so
// the error shows (review 0014).
func TestRejectedSearchTakesTheTable(t *testing.T) {
	h, _ := server(t)
	w := do(h, "GET", "/orders?q=a&placed_at.from=x", "", map[string]string{"HX-Request": "true", "HX-Target": "orders-results"})
	if w.Code != 422 || w.Header().Get("HX-Retarget") != "#orders" || w.Header().Get("HX-Reselect") != "#orders" {
		t.Errorf("%d %v", w.Code, w.Header())
	}
}

func TestDeclarationPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"no sizes":    func() { tables.New("x", oc.ID, tables.Col(oc.Email)).Sizes() },
		"a zero size": func() { tables.New("x", oc.ID, tables.Col(oc.Email)).Sizes(10, 0) },
		"a date key":  func() { tables.New("x", oc.PlacedAt, tables.Col(oc.Email)) },
		"a q filter": func() {
			tables.New("x", oc.ID, tables.Col(col.Column[OrderListView]{Name: "q", Kind: col.Text}).Filter())
		},
		"amount text":  func() { tables.New("x", oc.ID, tables.Col(oc.Email).Amount(money)) },
		"bad name":     func() { tables.New("X", oc.ID, tables.Col(oc.Email)) },
		"listed twice": func() { tables.New("x", oc.ID, tables.Col(oc.Email), tables.Col(oc.Email)) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s didn't panic", name)
				}
			}()
			f()
		}()
	}
}

func TestKeySortOnce(t *testing.T) {
	tb := tables.New("k", oc.ID, tables.Col(oc.ID).Sortable(), tables.Col(oc.Email)).Label(i18n.Static("K"))
	q := tablestest.Query(tablestest.Bind(t, tb, "sort=-id"))
	if len(q.Sort) != 1 || q.Sort[0].Column != "id" || !q.Sort[0].Desc {
		t.Errorf("sort %v", q.Sort)
	}
}
