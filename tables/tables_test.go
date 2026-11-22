package tables_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"
	"github.com/epistemer/models/listing"
	"github.com/epistemer/models/option"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/tables"
	"github.com/epistemer/ui/tables/tablestest"
)

type status string

func (s status) Value() string    { return string(s) }
func (s status) Label() i18n.TKer { return i18n.Key("enum.test_status."+string(s), string(s)) }

type order struct {
	ID     int64
	Email  string
	Status status
	Total  int64
	Placed time.Time
	Gift   bool
}

func lbl(s string) i18n.TKer { return i18n.Key("table.test_order."+s, s) }

var c = struct{ ID, Email, Status, Total, Placed, Gift column.Column[order] }{
	ID:     column.Column[order]{Name: "id", Label: lbl("ID"), Kind: column.Integer, Get: func(o order) column.Value { return column.Value{Int: o.ID} }},
	Email:  column.Column[order]{Name: "email", Label: lbl("Email"), Kind: column.Text, Get: func(o order) column.Value { return column.Value{Text: o.Email} }},
	Status: column.Column[order]{Name: "status", Label: lbl("Status"), Kind: column.Enum, Options: []option.SelectOption{status("paid"), status("pending")}, Get: func(o order) column.Value { return column.Value{Enum: o.Status} }},
	Total:  column.Column[order]{Name: "total", Label: lbl("Total"), Kind: column.Integer, Get: func(o order) column.Value { return column.Value{Int: o.Total} }},
	Placed: column.Column[order]{Name: "placed_at", Label: lbl("Placed"), Kind: column.Date, Get: func(o order) column.Value { return column.Value{Time: o.Placed} }},
	Gift:   column.Column[order]{Name: "gift", Label: lbl("Gift"), Kind: column.Bool, Get: func(o order) column.Value { return column.Value{Bool: o.Gift} }},
}

var refund = actions.New(i18n.Static("Refund"))

var orders = tables.New("orders", c.ID,
	tables.Col(c.Email).Searchable().Sortable(),
	tables.Col(c.Status).Filter(),
	tables.Col(c.Total).Sortable().Filter().Amount(func(o order) i18n.Ter { return i18n.Static("EGP") }),
	tables.Col(c.Placed).Sortable().Filter(),
	tables.Col(c.Gift).Filter(),
).Label(i18n.Static("Orders")).RowLabel(c.Email).Sort(c.Placed, tables.Desc).Sizes(10, 25).
	Actions(func(ctx context.Context, o order) []actions.Item {
		return actions.Resolve(ctx, refund.To(ui.URL("/orders/"+o.Email)))
	}).
	Bulk(func(ctx context.Context) []actions.Item { return nil })

var toys = tables.New("toys", c.ID, tables.Col(c.Email).Searchable(), tables.Col(c.Total).Filter()).
	Label(i18n.Static("Toys")).Prefixed()

func TestQuery(t *testing.T) {
	st := tablestest.Bind(t, orders, "q=ann&status=paid&placed_at.from=2026-11-03&sort=-total&page=2&size=25&toys.q=x")
	q := tablestest.Query(st)
	if q.Search != "ann" || strings.Join(q.SearchIn, ",") != "email" {
		t.Errorf("search %q in %v", q.Search, q.SearchIn)
	}
	if len(q.Filters) != 2 || q.Filters[0].Column != "status" || q.Filters[0].Value != "paid" || q.Filters[1].Column != "placed_at" {
		t.Errorf("filters %+v", q.Filters)
	}
	if q.Sort[0].Column != "total" || !q.Sort[0].Desc || q.Sort[1].Column != "id" || !q.Sort[1].Desc {
		t.Errorf("sort %v: the key ends it, in its direction", q.Sort)
	}
	if q.Limit != 25 || q.Offset != 25 {
		t.Errorf("limit %d offset %d", q.Limit, q.Offset)
	}
	// Junk in what links write is normalised without a word.
	q = tablestest.Query(tablestest.Bind(t, orders, "sort=status&page=-3&size=7"))
	if q.Sort[0].Column != "placed_at" || q.Offset != 0 || q.Limit != 10 {
		t.Errorf("junk: sort %v offset %d limit %d", q.Sort, q.Offset, q.Limit)
	}
	// A time's range ends on the whole last day; a bool filters.
	q = tablestest.Query(tablestest.Bind(t, orders, "gift=true&total.to=500"))
	if len(q.Filters) != 2 || q.Filters[0].Value != 500 || q.Filters[1].Value != true {
		t.Errorf("total and gift: %+v", q.Filters)
	}
	// The prefixed table reads only its own parameters.
	q = tablestest.Query(tablestest.Bind(t, toys, "q=ann&toys.q=bear&toys.total.from=3"))
	if q.Search != "bear" || len(q.Filters) != 1 || q.Filters[0].Value != 3 {
		t.Errorf("prefixed: %+v", q)
	}
}

func TestRejected(t *testing.T) {
	st := tablestest.Bind(t, orders, "placed_at.from=2026-11-15&placed_at.to=2026-11-02&status=lost")
	if st.Valid() {
		t.Fatal("bound")
	}
	if got := strings.Join(tablestest.Errors(st), ";"); got != "status: errors.forms.not_an_option;placed_at: errors.tables.range" {
		t.Errorf("errors %q", got)
	}
	if q := tablestest.Query(st); len(q.Filters) != 0 {
		t.Error("a rejected state filters")
	}
	v := st.Rejected()
	if !v.Rejected || len(v.Rows) != 0 {
		t.Error("rejected view")
	}
	// Answer: 422, and a results request is retargeted to the table.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/orders", nil)
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "orders-results")
	if st.Answer(w, r) != 422 || w.Header().Get("HX-Retarget") != "#orders" || w.Header().Get("HX-Reselect") != "#orders" {
		t.Errorf("answer %v", w.Header())
	}
	if ok := tablestest.Bind(t, orders, ""); ok.Answer(httptest.NewRecorder(), r) != 200 {
		t.Error("a valid state's answer")
	}
}

func TestView(t *testing.T) {
	st := tablestest.Bind(t, orders, "q=ann&status=paid&gift=true&placed_at.from=2026-11-03&sort=total&size=25&page=2&toys.q=x&other=1")
	rows := []order{{ID: 7, Email: "ann@example.com", Status: "paid", Total: 900}, {ID: 8, Email: "a b@x", Total: 10}}
	v := st.Page(rows, 30)
	if v.Sort != [2]string{"sort", "total"} || v.Size != [2]string{"size", "25"} {
		t.Errorf("sort %v size %v", v.Sort, v.Size)
	}
	if got := hidden(v.Hidden); got != "other=1&toys.q=x" {
		t.Errorf("hidden %q", got)
	}
	chips := []string{}
	for _, ch := range v.Chips {
		chips = append(chips, ch.Value)
	}
	if strings.Join(chips, "|") != "paid|≥ 2026-11-03|Yes" {
		t.Errorf("chips %q", chips)
	}
	if v.Pager.From != 26 || v.Pager.To != 30 || v.Pager.Pages != 2 || v.Pager.Next != "" || v.Pager.Target != "orders-results" {
		t.Errorf("pager %+v", v.Pager)
	}
	if v.Pager.Label.T(context.Background()) != "Pages of Orders" {
		t.Errorf("pager label %q", v.Pager.Label.T(context.Background()))
	}
	if v.Rows[1].ID != "orders-row-8" || v.Rows[0].Label != "ann@example.com" || !v.Rows[0].Selectable || len(v.Rows[0].Actions) != 1 {
		t.Errorf("rows %+v", v.Rows[0])
	}
	if v.Status != "26–30 of 30" {
		t.Errorf("status %q", v.Status)
	}
	// The sort link toggles; a change goes back to page 1 and keeps the rest.
	for _, h := range v.Columns {
		if h.Name == "total" && (!h.Sorted || h.Desc || !strings.Contains(string(h.Link), "sort=-total") || strings.Contains(string(h.Link), "page=")) {
			t.Errorf("total's head %+v", h)
		}
	}
	if e := tablestest.Bind(t, orders, "q=nobody").Page(nil, 0); e.Empty != tables.NoMatch || e.EmptyLink != "/" {
		t.Errorf("no match: %v %q", e.Empty, e.EmptyLink)
	}
	if e := tablestest.Bind(t, orders, "").Page(nil, 0); e.Empty != tables.NoRows {
		t.Error("no rows")
	}
	if e := tablestest.Bind(t, orders, "page=9").Page(nil, 12); e.Empty != tables.PastEnd || e.EmptyLink != "/?page=2" {
		t.Errorf("past end: %v %q", e.Empty, e.EmptyLink)
	}
}

func hidden(h [][2]string) string {
	q := url.Values{}
	for _, kv := range h {
		q.Add(kv[0], kv[1])
	}
	return q.Encode()
}

func TestNormalised(t *testing.T) {
	// Junk in sort, size and page never travels on in links; empties drop.
	v := tablestest.Bind(t, orders, "sort=bogus&size=7&page=junk&q=&other=&x=1").Page(nil, 0)
	if v.URL != "/?x=1" || v.Sort[1] != "" || v.Size[1] != "" || hidden(v.Hidden) != "x=1" {
		t.Errorf("URL %q sort %v size %v hidden %v", v.URL, v.Sort, v.Size, v.Hidden)
	}
	// A huge page caps rather than overflowing the offset.
	q := tablestest.Query(tablestest.Bind(t, orders, "page=9223372036854775807&size=25"))
	if q.Offset < 0 {
		t.Errorf("offset %d", q.Offset)
	}
	// An empty text key is "", not "0".
	byText := tables.New("t", c.Email, tables.Col(c.Total)).Label(i18n.Static("T"))
	if r := byText.Row(context.Background(), order{}); r.Key != "" || r.ID != "t-row-" {
		t.Errorf("empty text key: %q %q", r.Key, r.ID)
	}
	// A rejected state shows no chips.
	if v := tablestest.Bind(t, orders, "gift=maybe&status=paid").Rejected(); len(v.Chips) != 0 {
		t.Errorf("rejected chips: %v", v.Chips)
	}
}

func TestIDsAndSelection(t *testing.T) {
	if got := orders.RowID("a b@x.y"); got != "orders-row-a_20b_40x_2ey" {
		t.Errorf("RowID %q", got)
	}
	if orders.SelectionName() != "selected" || toys.SelectionName() != "toys.selected" {
		t.Error("SelectionName")
	}
	post := httptest.NewRequest("POST", "/orders/ship?selected=9", strings.NewReader("selected=1&selected=2"))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := orders.Selected(post); strings.Join(got, ",") != "1,2" {
		t.Errorf("a POST's selection, the query left out: %v", got)
	}
	get := httptest.NewRequest("GET", "/orders/tag?selected=3&selected=4", nil)
	if got := orders.Selected(get); strings.Join(got, ",") != "3,4" {
		t.Errorf("a GET's selection: %v", got)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "orders-results")
	if !tables.Targeted(r, orders) || tables.Targeted(r, toys) {
		t.Error("Targeted")
	}
	// An address Bind builds stays on the site.
	st := orders.Bind(httptest.NewRequest("GET", "//evil.example/orders?q=a", nil).WithContext(actions.Gate.With(context.Background(), actions.AllowAll)))
	if v := st.Page(nil, 0); strings.HasPrefix(v.URL, "//") || strings.HasPrefix(string(v.Clear), "//") {
		t.Errorf("not local: %q %q", v.URL, v.Clear)
	}
}

func TestCheck(t *testing.T) {
	tablestest.Check(t, orders)
	noLabel := tables.New("a", c.ID, tables.Col(c.Email))
	if noLabel.Check() == nil {
		t.Error("a table without a label passes")
	}
	noRowLabel := tables.New("b", c.ID, tables.Col(c.Total)).Label(i18n.Static("B")).
		Actions(func(context.Context, order) []actions.Item { return nil })
	if noRowLabel.Check() == nil {
		t.Error("a table with actions and no row label passes")
	}
	defer func() {
		if recover() == nil {
			t.Error("binding a table Check refuses didn't panic")
		}
	}()
	noRowLabel.Bind(httptest.NewRequest("GET", "/", nil))
}

func TestDeclarationPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"a bad name":      func() { tables.New("Orders", c.ID, tables.Col(c.Email)) },
		"no columns":      func() { tables.New("x", c.ID) },
		"a date key":      func() { tables.New("x", c.Placed, tables.Col(c.Email)) },
		"a column twice":  func() { tables.New("x", c.ID, tables.Col(c.Email), tables.Col(c.Email)) },
		"a q filter":      func() { tables.New("x", c.ID, tables.Col(column.Column[order]{Name: "q", Kind: column.Text}).Filter()) },
		"search a number": func() { tables.New("x", c.ID, tables.Col(c.Total).Searchable()) },
		"amount on text":  func() { tables.New("x", c.ID, tables.Col(c.Email).Amount(func(order) i18n.Ter { return nil })) },
		"two renderers": func() {
			tables.New("x", c.ID, tables.Col(c.Total).Amount(func(order) i18n.Ter { return nil }).Link(func(order) ui.Link { return nil }))
		},
		"an unknown label": func() { tables.New("x", c.ID, tables.Col(c.Email)).RowLabel(c.Total) },
		"an unsortable sort": func() {
			tables.New("x", c.ID, tables.Col(c.Email)).Sort(c.Email, tables.Asc)
		},
		"no sizes": func() { tables.New("x", c.ID, tables.Col(c.Email)).Sizes() },
		"a builder after use": func() {
			tb := tables.New("x", c.ID, tables.Col(c.Email)).Label(i18n.Static("X"))
			tb.Bind(httptest.NewRequest("GET", "/", nil))
			tb.Sizes(5)
		},
	} {
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.HasPrefix(r.(string), "tables: ") {
					t.Errorf("%s: %v", name, r)
				}
			}()
			f()
		}()
	}
}

type stubLabeler struct {
	names map[string]string
	err   error
	calls int
}

func (l *stubLabeler) LabelOf(_ context.Context, id string) (string, bool, error) {
	l.calls++
	if l.err != nil {
		return "", false, l.err
	}
	name, ok := l.names[id]
	return name, ok, nil
}

func TestRemotePanicsAndCheck(t *testing.T) {
	lab := &stubLabeler{names: map[string]string{"1": "Ann"}}
	search := ui.URL("/customers/search")

	for name, f := range map[string]func(){
		"nil search link": func() { tables.Col(c.ID).Filter().Remote(nil, lab) },
		"nil labeler":     func() { tables.Col(c.ID).Filter().Remote(search, nil) },
	} {
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.HasPrefix(r.(string), "tables: ") {
					t.Errorf("%s: %v", name, r)
				}
			}()
			f()
		}()
	}

	numCol := column.Column[order]{Name: "rate", Label: lbl("Rate"), Kind: column.Number, Get: func(order) column.Value { return column.Value{} }}
	for name, col := range map[string]tables.Column[order]{
		"without filter": tables.Col(c.Total).Remote(search, lab),
		"enum column":    tables.Col(c.Status).Filter().Remote(search, lab),
		"date column":    tables.Col(c.Placed).Filter().Remote(search, lab),
		"bool column":    tables.Col(c.Gift).Filter().Remote(search, lab),
		"number column":  tables.Col(numCol).Filter().Remote(search, lab),
	} {
		tb := tables.New("r", c.ID, tables.Col(c.Email), col).Label(i18n.Static("R"))
		if tb.Check() == nil {
			t.Errorf("%s: Check passed", name)
		}
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: Bind didn't panic", name)
				}
			}()
			tb.Bind(httptest.NewRequest("GET", "/", nil))
		}()
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: Row didn't panic", name)
				}
			}()
			tb.Row(context.Background(), order{})
		}()
	}

	// Remote before Filter on Text and Integer is legal.
	okTable := tables.New("ok", c.ID,
		tables.Col(c.Email).Remote(search, lab).Filter(),
		tables.Col(c.Total).Remote(search, lab).Filter(),
	).Label(i18n.Static("OK"))
	tablestest.Check(t, okTable)
}

func TestRemoteFilterAndChips(t *testing.T) {
	custLab := &stubLabeler{names: map[string]string{"7": "Ann Lee"}}
	refLab := &stubLabeler{names: map[string]string{"u-42": "Batch 42"}}
	custSearch := ui.URL("/customers/search")
	refSearch := ui.URL("/refs/search")

	custCol := column.Column[order]{Name: "customer_id", Label: lbl("Customer"), Kind: column.Integer, Get: func(o order) column.Value { return column.Value{Int: o.ID} }}
	refCol := column.Column[order]{Name: "ref", Label: lbl("Ref"), Kind: column.Text, Get: func(o order) column.Value { return column.Value{Text: o.Email} }}

	tb := tables.New("orders", c.ID,
		tables.Col(c.Email),
		tables.Col(custCol).Hidden().Filter().Remote(custSearch, custLab),
		tables.Col(refCol).Hidden().Filter().Remote(refSearch, refLab),
		tables.Col(c.Status).Filter(),
	).Label(i18n.Static("Orders"))

	// Integer and Text remote filters bind as Eq with typed values (int64, string).
	st := tablestest.Bind(t, tb, "customer_id=7&ref=u-42&other=1")
	if !st.Valid() {
		t.Fatalf("expected valid state, got %v", tablestest.Errors(st))
	}
	q := tablestest.Query(st)
	if len(q.Filters) != 2 {
		t.Fatalf("filters %+v", q.Filters)
	}
	if q.Filters[0].Column != "customer_id" || q.Filters[0].Op != listing.Eq {
		t.Errorf("int filter: %+v", q.Filters[0])
	}
	if got, ok := q.Filters[0].Value.(int64); !ok || got != 7 {
		t.Errorf("int filter value: %T(%v), want int64(7)", q.Filters[0].Value, q.Filters[0].Value)
	}
	if q.Filters[1].Column != "ref" || q.Filters[1].Op != listing.Eq {
		t.Errorf("text filter: %+v", q.Filters[1])
	}
	if got, ok := q.Filters[1].Value.(string); !ok || got != "u-42" {
		t.Errorf("text filter value: %T(%v), want string(\"u-42\")", q.Filters[1].Value, q.Filters[1].Value)
	}

	// Parameter ownership, filter controls, and chips (found).
	v := st.Page(nil, 0)
	if v.Err != nil {
		t.Fatalf("unexpected error: %v", v.Err)
	}
	if !slices.Contains(v.Own, "customer_id") || slices.Contains(v.Own, "customer_id.from") || !slices.Contains(v.Own, "ref") {
		t.Errorf("own %v", v.Own)
	}
	if hidden(v.Hidden) != "other=1" {
		t.Errorf("hidden %q", hidden(v.Hidden))
	}
	if len(v.Filters) < 2 || v.Filters[0].Kind() != forms.KindRemote || v.Filters[1].Kind() != forms.KindRemote {
		t.Fatalf("filter controls: %v", v.Filters)
	}
	if s, l, id, ok := v.Filters[0].Remote(); !ok || s.URL() != custSearch.URL() || l != custLab || id != "7" {
		t.Errorf("customer_id Remote() = (%v, %v, %q, %v)", s, l, id, ok)
	}
	if len(v.Chips) != 2 || v.Chips[0].Value != "Ann Lee" || v.Chips[1].Value != "Batch 42" {
		t.Errorf("chips %+v", v.Chips)
	}
	if strings.Contains(string(v.Chips[0].Remove), "customer_id=") || !strings.Contains(string(v.Chips[0].Remove), "ref=u-42") {
		t.Errorf("remove link %q", v.Chips[0].Remove)
	}
	if strings.Contains(string(v.Clear), "customer_id=") || strings.Contains(string(v.Clear), "ref=") {
		t.Errorf("clear link %q", v.Clear)
	}

	// LabelOf is called once per link filter per request, even across repeated Page/Rejected calls.
	_ = st.Rejected()
	_ = st.Page(nil, 0)
	if custLab.calls != 1 || refLab.calls != 1 {
		t.Errorf("LabelOf calls: cust=%d ref=%d, want 1 each", custLab.calls, refLab.calls)
	}

	// Miss shows the id.
	custLab.calls, refLab.calls = 0, 0
	miss := tablestest.Bind(t, tb, "customer_id=999&ref=unknown").Page(nil, 0)
	if miss.Err != nil || len(miss.Chips) != 2 || miss.Chips[0].Value != "999" || miss.Chips[1].Value != "unknown" {
		t.Errorf("miss: err=%v chips=%+v", miss.Err, miss.Chips)
	}
	if custLab.calls != 1 || refLab.calls != 1 {
		t.Errorf("miss calls: cust=%d ref=%d", custLab.calls, refLab.calls)
	}

	// Unset filters do not call LabelOf.
	custLab.calls, refLab.calls = 0, 0
	_ = tablestest.Bind(t, tb, "").Page(nil, 0)
	if custLab.calls != 0 || refLab.calls != 0 {
		t.Errorf("unset calls: cust=%d ref=%d", custLab.calls, refLab.calls)
	}

	// Invalid state never calls LabelOf.
	custLab.calls, refLab.calls = 0, 0
	bad := tablestest.Bind(t, tb, "customer_id=abc")
	if bad.Valid() {
		t.Fatal("non-integer bound")
	}
	_ = bad.Rejected()
	_ = bad.Page(nil, 0)
	if custLab.calls != 0 {
		t.Errorf("invalid customer_id called LabelOf %d times", custLab.calls)
	}
	bad2 := tablestest.Bind(t, tb, "customer_id=7&status=bogus")
	if bad2.Valid() {
		t.Fatal("bogus status bound")
	}
	_ = bad2.Rejected()
	if custLab.calls != 0 {
		t.Errorf("invalid sibling filter called LabelOf %d times", custLab.calls)
	}

	// Error fails the view.
	boom := errors.New("db down")
	custLab.err = boom
	custLab.calls = 0
	errView := tablestest.Bind(t, tb, "customer_id=7").Page(nil, 0)
	if !errors.Is(errView.Err, boom) || len(errView.Chips) != 0 {
		t.Errorf("error view: err=%v chips=%+v", errView.Err, errView.Chips)
	}
	custLab.err = nil

	// Prefixed table owns "<name>.<col>" and binds typed int64.
	prefixed := tables.New("p", c.ID,
		tables.Col(c.Email),
		tables.Col(custCol).Hidden().Filter().Remote(custSearch, custLab),
	).Label(i18n.Static("P")).Prefixed()
	pst := tablestest.Bind(t, prefixed, "p.customer_id=7&customer_id=9")
	pq := tablestest.Query(pst)
	if len(pq.Filters) != 1 || pq.Filters[0].Value != int64(7) {
		t.Errorf("prefixed query: %+v", pq.Filters)
	}
	pv := pst.Page(nil, 0)
	if !slices.Contains(pv.Own, "p.customer_id") || hidden(pv.Hidden) != "customer_id=9" || len(pv.Chips) != 1 || pv.Chips[0].Value != "Ann Lee" {
		t.Errorf("prefixed view: own=%v hidden=%q chips=%+v", pv.Own, hidden(pv.Hidden), pv.Chips)
	}
}
