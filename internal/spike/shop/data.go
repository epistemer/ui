// Package shop is the toy shop the spike runs: two views' rows in memory,
// their columns as models would generate them, the tables, the actions,
// and the handlers.
package shop

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/listing"
	"github.com/epistemer/models/option"

	"github.com/epistemer/ui/internal/spike/col"
)

// OrderStatus is an enum, as models generates them.
type OrderStatus string

const (
	Pending  OrderStatus = "pending"
	Paid     OrderStatus = "paid"
	Shipped  OrderStatus = "shipped"
	Refunded OrderStatus = "refunded"
)

var statusLabels = map[OrderStatus]i18n.TKer{
	Pending:  i18n.Key("enum.order_status.pending", "pending"),
	Paid:     i18n.Key("enum.order_status.paid", "paid"),
	Shipped:  i18n.Key("enum.order_status.shipped", "shipped"),
	Refunded: i18n.Key("enum.order_status.refunded", "refunded"),
}

func (s OrderStatus) Value() string    { return string(s) }
func (s OrderStatus) Label() i18n.TKer { return statusLabels[s] }
func AllOrderStatusOptions() []option.SelectOption {
	return []option.SelectOption{Pending, Paid, Shipped, Refunded}
}

// OrderListView is a row of order_list_view.
type OrderListView struct {
	ID         int64
	Email      string
	Customer   string
	CustomerID int64
	Status     OrderStatus
	Total      int64 // piastres
	PlacedAt   time.Time
	Assignee   string
	Tags       []string
}

// OrderListViewColumns is what models would generate for the view.
var OrderListViewColumns = struct {
	ID, Email, Customer, Status, Total, PlacedAt, Assignee col.Column[OrderListView]
}{
	ID:       col.Column[OrderListView]{Name: "id", Label: i18n.Key("table.order_list_view.ID", "ID"), Kind: col.Integer, Get: func(o OrderListView) col.Value { return col.IntOf(o.ID) }},
	Email:    col.Column[OrderListView]{Name: "email", Label: i18n.Key("table.order_list_view.Email", "Email"), Kind: col.Text, Get: func(o OrderListView) col.Value { return col.TextOf(o.Email) }},
	Customer: col.Column[OrderListView]{Name: "customer", Label: i18n.Key("table.order_list_view.Customer", "Customer"), Kind: col.Text, Get: func(o OrderListView) col.Value { return col.TextOf(o.Customer) }},
	Status:   col.Column[OrderListView]{Name: "status", Label: i18n.Key("table.order_list_view.Status", "Status"), Kind: col.Enum, Options: AllOrderStatusOptions(), Get: func(o OrderListView) col.Value { return col.EnumOf(o.Status) }},
	Total:    col.Column[OrderListView]{Name: "total", Label: i18n.Key("table.order_list_view.Total", "Total"), Kind: col.Integer, Get: func(o OrderListView) col.Value { return col.IntOf(o.Total) }},
	PlacedAt: col.Column[OrderListView]{Name: "placed_at", Label: i18n.Key("table.order_list_view.PlacedAt", "Placed"), Kind: col.Date, Get: func(o OrderListView) col.Value { return col.TimeOf(o.PlacedAt) }},
	Assignee: col.Column[OrderListView]{Name: "assignee", Label: i18n.Key("table.order_list_view.Assignee", "Assignee"), Kind: col.Text, Get: func(o OrderListView) col.Value {
		if o.Assignee == "" {
			return col.Value{Null: true}
		}
		return col.TextOf(o.Assignee)
	}},
}

// ToyListView is a row of toy_list_view.
type ToyListView struct {
	ID       int64
	Name     string
	Stock    int64
	Electric bool
}

var ToyListViewColumns = struct {
	ID, Name, Stock, Electric col.Column[ToyListView]
}{
	ID:       col.Column[ToyListView]{Name: "id", Label: i18n.Key("table.toy_list_view.ID", "ID"), Kind: col.Integer, Get: func(t ToyListView) col.Value { return col.IntOf(t.ID) }},
	Name:     col.Column[ToyListView]{Name: "name", Label: i18n.Key("table.toy_list_view.Name", "Name"), Kind: col.Text, Get: func(t ToyListView) col.Value { return col.TextOf(t.Name) }},
	Stock:    col.Column[ToyListView]{Name: "stock", Label: i18n.Key("table.toy_list_view.Stock", "Stock"), Kind: col.Integer, Get: func(t ToyListView) col.Value { return col.IntOf(t.Stock) }},
	Electric: col.Column[ToyListView]{Name: "electric", Label: i18n.Key("table.toy_list_view.Electric", "Electric"), Kind: col.Bool, Get: func(t ToyListView) col.Value { return col.BoolOf(t.Electric) }},
}

// Store holds the shop's rows.
type Store struct {
	mu     sync.Mutex
	orders []OrderListView
	toys   []ToyListView
}

var (
	names    = []string{"Layla Adel", "Omar Nabil", "Mona Saeed", "Karim Fawzy", "Nour Hassan", "Youssef Ali", "Salma Tarek", "Hana Mostafa"}
	toyNames = []string{"Bear", "Train", "Kite", "Robot", "Puzzle", "Drum", "Yo-yo", "Blocks", "Car", "Doll", "Rocket", "Boat"}
)

// NewStore fills a store with the same rows every time.
func NewStore() *Store {
	s := &Store{}
	day := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	statuses := []OrderStatus{Paid, Pending, Shipped, Paid, Refunded, Paid, Pending}
	for i := range 57 {
		n := names[i%len(names)]
		first := strings.ToLower(strings.Fields(n)[0])
		s.orders = append(s.orders, OrderListView{
			ID: int64(1001 + i), Email: fmt.Sprintf("%s%d@example.com", first, i+1), Customer: n,
			CustomerID: int64(1 + i%len(names)), Status: statuses[i%len(statuses)],
			Total: int64(2500 + (i*7919)%40000), PlacedAt: day.AddDate(0, 0, i%21),
		})
	}
	for i, n := range toyNames {
		s.toys = append(s.toys, ToyListView{ID: int64(1 + i), Name: n, Stock: int64((i * 37) % 60), Electric: i%3 == 1})
	}
	return s
}

// apply runs a listing query over rows in Go, as models would in SQL.
func apply[R any](rows []R, cols map[string]col.Column[R], q listing.Query) ([]R, int) {
	var out []R
	for _, r := range rows {
		if q.Search != "" {
			hit := false
			for _, n := range q.SearchIn {
				if strings.Contains(strings.ToLower(cols[n].Get(r).Text), strings.ToLower(q.Search)) {
					hit = true
				}
			}
			if !hit {
				continue
			}
		}
		keep := true
		for _, f := range q.Filters {
			if !match(cols[f.Column], r, f) {
				keep = false
			}
		}
		if keep {
			out = append(out, r)
		}
	}
	slices.SortStableFunc(out, func(a, b R) int {
		for _, s := range q.Sort {
			c := compare(cols[s.Column], a, b)
			if s.Desc {
				c = -c
			}
			if c != 0 {
				return c
			}
		}
		return 0
	})
	total := len(out)
	if q.Offset > len(out) {
		out = nil
	} else {
		out = out[q.Offset:]
	}
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, total
}

func compare[R any](c col.Column[R], a, b R) int {
	x, y := c.Get(a), c.Get(b)
	switch c.Kind {
	case col.Integer:
		return cmp.Compare(x.Int, y.Int)
	case col.Date, col.Time:
		return x.Time.Compare(y.Time)
	case col.Bool:
		return cmp.Compare(fmt.Sprint(x.Bool), fmt.Sprint(y.Bool))
	case col.Enum:
		return cmp.Compare(x.Enum.Value(), y.Enum.Value())
	}
	return cmp.Compare(strings.ToLower(x.Text), strings.ToLower(y.Text))
}

func match[R any](c col.Column[R], r R, f listing.Filter) bool {
	v := c.Get(r)
	var d int
	switch c.Kind {
	case col.Integer:
		d = cmp.Compare(v.Int, int64(f.Value.(int)))
	case col.Date, col.Time:
		d = v.Time.Compare(f.Value.(time.Time))
	case col.Bool:
		return f.Op == listing.Eq && v.Bool == f.Value.(bool)
	case col.Enum:
		return f.Op == listing.Eq && v.Enum.Value() == f.Value.(string)
	default:
		return f.Op == listing.Eq && v.Text == f.Value.(string)
	}
	switch f.Op {
	case listing.Eq:
		return d == 0
	case listing.Ge:
		return d >= 0
	case listing.Le:
		return d <= 0
	case listing.Lt:
		return d < 0
	case listing.Gt:
		return d > 0
	}
	return d != 0
}

func columns[R any](cs ...col.Column[R]) map[string]col.Column[R] {
	m := map[string]col.Column[R]{}
	for _, c := range cs {
		m[c.Name] = c
	}
	return m
}

var (
	oc        = OrderListViewColumns
	tc        = ToyListViewColumns
	orderCols = columns(oc.ID, oc.Email, oc.Customer, oc.Status, oc.Total, oc.PlacedAt, oc.Assignee)
	toyCols   = columns(tc.ID, tc.Name, tc.Stock, tc.Electric)
)

// ListOrderListView is the generated List method, over a listing.Source.
func (s *Store) ListOrderListView(src listing.Source) ([]OrderListView, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return apply(s.orders, orderCols, src.ListingQuery())
}

func (s *Store) ListToyListView(src listing.Source) ([]ToyListView, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return apply(s.toys, toyCols, src.ListingQuery())
}

// Order returns one order.
func (s *Store) Order(id int64) (OrderListView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.orders, func(o OrderListView) bool { return o.ID == id })
	if i < 0 {
		return OrderListView{}, false
	}
	return s.orders[i], true
}

// Update changes one order.
func (s *Store) Update(id int64, f func(*OrderListView)) (OrderListView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.orders, func(o OrderListView) bool { return o.ID == id })
	if i < 0 {
		return OrderListView{}, false
	}
	f(&s.orders[i])
	return s.orders[i], true
}

// Delete removes one order.
func (s *Store) Delete(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.orders)
	s.orders = slices.DeleteFunc(s.orders, func(o OrderListView) bool { return o.ID == id })
	return len(s.orders) < n
}
