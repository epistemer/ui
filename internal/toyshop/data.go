// Package shop is the toy shop the spike runs: two views' rows in memory,
// their columns as models would generate them, the tables, the actions,
// and the handlers.
package toyshop

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"
	"github.com/epistemer/models/listing"
	"github.com/epistemer/models/option"
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
	ID, Email, Customer, CustomerID, Status, Total, PlacedAt, Assignee column.Column[OrderListView]
}{
	ID:         column.Column[OrderListView]{Name: "id", Label: i18n.Key("table.order_list_view.ID", "ID"), Kind: column.Integer, Get: func(o OrderListView) column.Value { return intOf(o.ID) }},
	Email:      column.Column[OrderListView]{Name: "email", Label: i18n.Key("table.order_list_view.Email", "Email"), Kind: column.Text, Get: func(o OrderListView) column.Value { return textOf(o.Email) }},
	Customer:   column.Column[OrderListView]{Name: "customer", Label: i18n.Key("table.order_list_view.Customer", "Customer"), Kind: column.Text, Get: func(o OrderListView) column.Value { return textOf(o.Customer) }},
	CustomerID: column.Column[OrderListView]{Name: "customer_id", Label: i18n.Key("table.order_list_view.CustomerID", "Customer"), Kind: column.Integer, Get: func(o OrderListView) column.Value { return intOf(o.CustomerID) }},
	Status:     column.Column[OrderListView]{Name: "status", Label: i18n.Key("table.order_list_view.Status", "Status"), Kind: column.Enum, Options: AllOrderStatusOptions(), Get: func(o OrderListView) column.Value { return enumOf(o.Status) }},
	Total:      column.Column[OrderListView]{Name: "total", Label: i18n.Key("table.order_list_view.Total", "Total"), Kind: column.Integer, Get: func(o OrderListView) column.Value { return intOf(o.Total) }},
	PlacedAt:   column.Column[OrderListView]{Name: "placed_at", Label: i18n.Key("table.order_list_view.PlacedAt", "Placed"), Kind: column.Date, Get: func(o OrderListView) column.Value { return timeOf(o.PlacedAt) }},
	Assignee: column.Column[OrderListView]{Name: "assignee", Label: i18n.Key("table.order_list_view.Assignee", "Assignee"), Kind: column.Text, Get: func(o OrderListView) column.Value {
		if o.Assignee == "" {
			return column.Value{Null: true}
		}
		return textOf(o.Assignee)
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
	ID, Name, Stock, Electric column.Column[ToyListView]
}{
	ID:       column.Column[ToyListView]{Name: "id", Label: i18n.Key("table.toy_list_view.ID", "ID"), Kind: column.Integer, Get: func(t ToyListView) column.Value { return intOf(t.ID) }},
	Name:     column.Column[ToyListView]{Name: "name", Label: i18n.Key("table.toy_list_view.Name", "Name"), Kind: column.Text, Get: func(t ToyListView) column.Value { return textOf(t.Name) }},
	Stock:    column.Column[ToyListView]{Name: "stock", Label: i18n.Key("table.toy_list_view.Stock", "Stock"), Kind: column.Integer, Get: func(t ToyListView) column.Value { return intOf(t.Stock) }},
	Electric: column.Column[ToyListView]{Name: "electric", Label: i18n.Key("table.toy_list_view.Electric", "Electric"), Kind: column.Bool, Get: func(t ToyListView) column.Value { return boolOf(t.Electric) }},
}

// Plain values, as generated getters build them.
func textOf(s string) column.Value    { return column.Value{Text: s} }
func intOf(n int64) column.Value      { return column.Value{Int: n} }
func boolOf(b bool) column.Value      { return column.Value{Bool: b} }
func timeOf(t time.Time) column.Value { return column.Value{Time: t, Null: t.IsZero()} }
func enumOf(o option.SelectOption) column.Value {
	return column.Value{Enum: o, Null: o == nil}
}

// Customer is a customer record in the shop.
type Customer struct {
	ID    int64
	Name  string
	Email string
}

// Store holds the shop's rows.
type Store struct {
	mu        sync.Mutex
	orders    []OrderListView
	toys      []ToyListView
	customers []Customer
}

var (
	names = []string{"Layla Adel", "Omar Nabil", "Mona Saeed", "Karim Fawzy", "Nour Hassan", "Youssef Ali", "Salma Tarek", "Hana Mostafa"}
	// extraNames fill out the customer directory past SuggestionLimit so
	// search answers can exercise the "more" sentinel without shifting the
	// seeded orders above.
	extraNames = []string{
		"Bassem Aziz", "Carla Ruiz", "Dina Farouk", "Elias Berg",
		"Fatma Nour", "Gus Holm", "Ann Lee", "Ilse Vos",
		"Jamal Odeh", "Kira Novak", "Laila Haddad", "Nils Lund",
		"Petra Kos", "Rana Adel",
	}
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
	for i, n := range append(slices.Clone(names), extraNames...) {
		first := strings.ToLower(strings.Fields(n)[0])
		s.customers = append(s.customers, Customer{
			ID:    int64(1 + i),
			Name:  n,
			Email: fmt.Sprintf("%s@example.com", first),
		})
	}
	for i, n := range toyNames {
		s.toys = append(s.toys, ToyListView{ID: int64(1 + i), Name: n, Stock: int64((i * 37) % 60), Electric: i%3 == 1})
	}
	return s
}

var defaultStore = NewStore()

// apply runs a listing query over rows in Go, as models would in SQL.
func apply[R any](rows []R, cols map[string]column.Column[R], q listing.Query) ([]R, int64) {
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
	return out, int64(total)
}

func compare[R any](c column.Column[R], a, b R) int {
	x, y := c.Get(a), c.Get(b)
	switch c.Kind {
	case column.Integer:
		return cmp.Compare(x.Int, y.Int)
	case column.Date, column.Time:
		return x.Time.Compare(y.Time)
	case column.Bool:
		return cmp.Compare(fmt.Sprint(x.Bool), fmt.Sprint(y.Bool))
	case column.Enum:
		return cmp.Compare(x.Enum.Value(), y.Enum.Value())
	}
	return cmp.Compare(strings.ToLower(x.Text), strings.ToLower(y.Text))
}

func match[R any](c column.Column[R], r R, f listing.Filter) bool {
	v := c.Get(r)
	var d int
	switch c.Kind {
	case column.Integer:
		var want int64
		switch x := f.Value.(type) {
		case int:
			want = int64(x)
		case int64:
			want = x
		}
		d = cmp.Compare(v.Int, want)
	case column.Date, column.Time:
		d = v.Time.Compare(f.Value.(time.Time))
	case column.Bool:
		return f.Op == listing.Eq && v.Bool == f.Value.(bool)
	case column.Enum:
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

func columns[R any](cs ...column.Column[R]) map[string]column.Column[R] {
	m := map[string]column.Column[R]{}
	for _, c := range cs {
		m[c.Name] = c
	}
	return m
}

var (
	oc        = OrderListViewColumns
	tc        = ToyListViewColumns
	orderCols = columns(oc.ID, oc.Email, oc.Customer, oc.CustomerID, oc.Status, oc.Total, oc.PlacedAt, oc.Assignee)
	toyCols   = columns(tc.ID, tc.Name, tc.Stock, tc.Electric)
)

// ListOrderListView is the generated List method, over a listing.Source.
func (s *Store) ListOrderListView(src listing.Source) ([]OrderListView, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return apply(s.orders, orderCols, src.ListingQuery())
}

func (s *Store) ListToyListView(src listing.Source) ([]ToyListView, int64) {
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

// CustomerName returns a customer's display name by id.
func (s *Store) CustomerName(id int64) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.customers {
		if c.ID == id {
			return c.Name, true
		}
	}
	return "", false
}

// SearchCustomers finds customers matching q in name or email, sorted by
// name then id, returning at most limit rows.
func (s *Store) SearchCustomers(q string, limit int) []Customer {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Customer
	for _, c := range s.customers {
		if q == "" || strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(strings.ToLower(c.Email), q) {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b Customer) int {
		if d := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); d != 0 {
			return d
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ProductName returns a toy product's name by id.
func (s *Store) ProductName(id int64) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.toys {
		if t.ID == id {
			return t.Name, true
		}
	}
	return "", false
}

// SearchProducts finds products matching q in name, sorted by name then
// id, returning at most limit rows.
func (s *Store) SearchProducts(q string, limit int) []ToyListView {
	s.mu.Lock()
	defer s.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	var out []ToyListView
	for _, t := range s.toys {
		if q == "" || strings.Contains(strings.ToLower(t.Name), q) {
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b ToyListView) int {
		if d := cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); d != 0 {
			return d
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

type customerLabeler struct{ store *Store }

func (l customerLabeler) LabelOf(_ context.Context, id string) (string, bool, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", false, nil
	}
	name, ok := l.store.CustomerName(n)
	return name, ok, nil
}

// Customers returns a comparable listing.Labeler over the store's customers.
func (s *Store) Customers() listing.Labeler { return customerLabeler{store: s} }

type productLabeler struct{ store *Store }

func (l productLabeler) LabelOf(_ context.Context, id string) (string, bool, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", false, nil
	}
	name, ok := l.store.ProductName(n)
	return name, ok, nil
}

// Products returns a comparable listing.Labeler over the store's products.
func (s *Store) Products() listing.Labeler { return productLabeler{store: s} }
