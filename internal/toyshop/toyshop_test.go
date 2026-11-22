package toyshop

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/actions/actionstest"
	"github.com/epistemer/ui/inputs"
	"github.com/epistemer/ui/tables"
	"github.com/epistemer/ui/tables/tablestest"
	"github.com/epistemer/ui/theme"
	"github.com/epistemer/ui/uitest"
)

func gated(ctx context.Context) context.Context {
	return inputs.WithLabelCache(actions.Gate.With(ctx, Gate))
}

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

	// Remote customer filter binds, filters rows, and renders the chip with the customer's name.
	w := do(h, "GET", "/shop/orders?customer_id=1", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("filtered orders: %d", w.Code)
	}
	filtered := w.Body.String()
	if !strings.Contains(filtered, "Customer: Layla Adel") || strings.Contains(filtered, "Omar Nabil") {
		t.Errorf("customer filter chip/rows mismatch:\n%s", filtered)
	}
	uitest.CheckIDs(t, filtered)
	uitest.CheckNames(t, filtered)
	uitest.CheckSelectors(t, filtered, css(t))
}

func postMultipart(h http.Handler, target string, fields [][2]string, files []struct {
	field, name string
	data        []byte
}) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, kv := range fields {
		_ = mw.WriteField(kv[0], kv[1])
	}
	for _, f := range files {
		fw, _ := mw.CreateFormFile(f.field, f.name)
		_, _ = fw.Write(f.data)
	}
	_ = mw.Close()
	r := httptest.NewRequest(http.MethodPost, target, &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestOrderNewPage(t *testing.T) {
	h, _ := server(t)
	styles := css(t)

	// GET renders cleanly with stored files, remote selects, and one blank line.
	w := do(h, http.MethodGet, "/shop/orders/new", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /shop/orders/new: %d", w.Code)
	}
	html := w.Body.String()
	uitest.CheckIDs(t, html)
	uitest.CheckNames(t, html)
	uitest.CheckSelectors(t, html, styles)
	for _, want := range []string{
		`data-ui-search="/shop/customers/search"`,
		`data-ui-search="/shop/products/search"`,
		`name="receipt.keep" value="receipt-pdf" checked`,
		`name="receipt.keep" value="box-png" checked`,
		`src="/shop/files/box-png"`,
		`name="lines[0].product_id"`,
		`name="lines[0].qty"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("GET /shop/orders/new missing %q", want)
		}
	}

	// Stored file route serves both files and 404s unknown keys.
	if fw := do(h, http.MethodGet, "/shop/files/receipt-pdf", "", nil); fw.Code != http.StatusOK || fw.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("receipt-pdf: %d %q", fw.Code, fw.Header().Get("Content-Type"))
	}
	if fw := do(h, http.MethodGet, "/shop/files/box-png", "", nil); fw.Code != http.StatusOK || fw.Header().Get("Content-Type") != "image/png" {
		t.Errorf("box-png: %d %q", fw.Code, fw.Header().Get("Content-Type"))
	}
	if fw := do(h, http.MethodGet, "/shop/files/missing", "", nil); fw.Code != http.StatusNotFound {
		t.Errorf("missing file: %d", fw.Code)
	}

	// Failed POST (missing customer) returns 422, preserves keep states, and shows the lost note.
	failed := postMultipart(h, "/shop/orders/new", [][2]string{
		{"receipt.keep", "receipt-pdf"},
		{"lines[0].product_id", "1"},
		{"lines[0].qty", "2"},
	}, []struct {
		field, name string
		data        []byte
	}{
		{"receipt", "extra.pdf", []byte("%PDF-1.4\n%%EOF\n")},
	})
	if failed.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid POST: got %d, want 422", failed.Code)
	}
	fhtml := failed.Body.String()
	uitest.CheckIDs(t, fhtml)
	uitest.CheckNames(t, fhtml)
	uitest.CheckSelectors(t, fhtml, styles)
	if !strings.Contains(fhtml, `value="receipt-pdf" checked`) {
		t.Error("kept file receipt-pdf wasn't preserved as checked")
	}
	if strings.Contains(fhtml, `value="box-png" checked`) {
		t.Error("unchecked file box-png became checked again")
	}
	if !strings.Contains(fhtml, "ui-upload-lost") || !strings.Contains(fhtml, "Choose the files again") {
		t.Error("lost note missing on failed upload form")
	}

	// Valid POST renders a success alert summarizing customer, line count, kept keys and added count.
	ok := postMultipart(h, "/shop/orders/new", [][2]string{
		{"customer_id", "1"},
		{"receipt.keep", "box-png"},
		{"lines[0].product_id", "1"},
		{"lines[0].qty", "2"},
		{"lines[1].product_id", "4"},
		{"lines[1].qty", "3"},
	}, []struct {
		field, name string
		data        []byte
	}{
		{"receipt", "invoice.pdf", []byte("%PDF-1.4\n%%EOF\n")},
	})
	if ok.Code != http.StatusOK {
		t.Fatalf("valid POST: got %d, want 200\n%s", ok.Code, ok.Body.String())
	}
	ohtml := ok.Body.String()
	uitest.CheckIDs(t, ohtml)
	uitest.CheckNames(t, ohtml)
	uitest.CheckSelectors(t, ohtml, styles)
	for _, want := range []string{
		`data-tone="success"`,
		"Layla Adel",
		"2 lines",
		"box-png",
		"added 1",
	} {
		if !strings.Contains(ohtml, want) {
			t.Errorf("valid POST missing %q in:\n%s", want, ohtml)
		}
	}

	// Body over the 1.5 MiB limit answers 413.
	big := postMultipart(h, "/shop/orders/new", [][2]string{
		{"customer_id", "1"},
		{"lines[0].product_id", "1"},
		{"lines[0].qty", "1"},
	}, []struct {
		field, name string
		data        []byte
	}{
		{"receipt", "huge.pdf", bytes.Repeat([]byte("a"), maxOrderBody+1024)},
	})
	if big.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized POST: got %d, want 413", big.Code)
	}
}

func TestSearchHandlers(t *testing.T) {
	h, _ := server(t)

	// Customer search filters by q (case-insensitive).
	w := do(h, http.MethodGet, "/shop/customers/search?q=LAYLA", "", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "data-ui-suggestions") || !strings.Contains(w.Body.String(), `data-value="1"`) || !strings.Contains(w.Body.String(), "Layla Adel") || strings.Contains(w.Body.String(), "data-ui-more") {
		t.Errorf("customer search q=LAYLA: %d\n%s", w.Code, w.Body.String())
	}

	// Empty q fetches SuggestionLimit+1 so data-ui-more is rendered.
	w = do(h, http.MethodGet, "/shop/customers/search", "", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "data-ui-suggestions") || !strings.Contains(w.Body.String(), "data-ui-more") {
		t.Errorf("customer search empty q: %d\n%s", w.Code, w.Body.String())
	}

	// Product search filters by q.
	w = do(h, http.MethodGet, "/shop/products/search?q=rob", "", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "data-ui-suggestions") || !strings.Contains(w.Body.String(), `data-value="4"`) || !strings.Contains(w.Body.String(), "Robot") || strings.Contains(w.Body.String(), "data-ui-more") {
		t.Errorf("product search q=rob: %d\n%s", w.Code, w.Body.String())
	}
}
