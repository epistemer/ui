// Package tables declares a table once from generated columns, binds its
// state from the URL, and describes a page of rows
// (server-state-interface.md §3, ADR 0010). It renders nothing:
// components.TableOf renders the view, and an app can render it itself.
//
//	var Orders = tables.New("orders", c.ID,
//		tables.Col(c.Status).Filter(),
//		tables.Col(c.Email).Searchable().Sortable(),
//		tables.Col(c.PlacedAt).Sortable().Filter(),
//	).Label(L10nOrders).RowLabel(c.Email).Sort(c.PlacedAt, tables.Desc)
//
//	st := Orders.Bind(r)
//	if !st.Valid() { … st.Rejected() … }
//	rows, total, err := q.ListOrderListView(ctx, st) // a State is a listing.Source
//	view := st.Page(rows, total)
package tables

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"
	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
)

// ErrRange is a range filter whose start is after its end.
var ErrRange = errors.Key("errors.tables.range", "the start is after the end", errors.Invalid)

// Dir is a sort's direction.
type Dir bool

const (
	Asc  Dir = false
	Desc Dir = true
)

// Column is a generated column with what the page says about it (S22).
type Column[R any] struct {
	c          column.Column[R]
	searchable bool
	sortable   bool
	filter     bool
	hidden     bool
	link       func(R) ui.Link
	amount     func(R) i18n.Ter
	src        func(R) ui.Link
	alt        func(R) i18n.Ter
	search     ui.Link
	labeler    listing.Labeler
}

// Col starts a table column from a generated one.
func Col[R any](c column.Column[R]) Column[R] { return Column[R]{c: c} }

// Searchable makes the search box look in it. Text only.
func (c Column[R]) Searchable() Column[R] { c.searchable = true; return c }

// Sortable makes its head sort.
func (c Column[R]) Sortable() Column[R] { c.sortable = true; return c }

// Filter gives it a filter, by its kind (§3.4).
func (c Column[R]) Filter() Column[R] { c.filter = true; return c }

// Remote makes the column's filter a remote select with listing.Eq; the
// chip names the value through l (R13). Legal on a Text or Integer column
// with Filter(); a nil search link or Labeler panics at once.
func (c Column[R]) Remote(search ui.Link, l listing.Labeler) Column[R] {
	if search == nil {
		panic(fmt.Sprintf("tables: Remote on %q with a nil search link", c.c.Name))
	}
	if l == nil {
		panic(fmt.Sprintf("tables: Remote on %q with a nil Labeler", c.c.Name))
	}
	c.search, c.labeler = search, l
	return c
}

// Hidden keeps it for a filter or a sort without showing it.
func (c Column[R]) Hidden() Column[R] { c.hidden = true; return c }

// Link makes its text a link to another record.
func (c Column[R]) Link(f func(R) ui.Link) Column[R] { c.link = f; return c }

// Amount shows it as money, formatted by f: ui imports no decimal library,
// and a currency is the app's. Integers (minor units) or numbers.
func (c Column[R]) Amount(f func(R) i18n.Ter) Column[R] { c.amount = f; return c }

// Image shows an image, its address from src (a route, S8) and its
// alternative text from alt.
func (c Column[R]) Image(src func(R) ui.Link, alt func(R) i18n.Ter) Column[R] {
	c.src, c.alt = src, alt
	return c
}

// Table is declared once, at package level, and only read afterwards.
type Table[R any] struct {
	name     string
	key      column.Column[R]
	cols     []Column[R]
	label    i18n.Ter
	rowLabel string
	sort     string
	desc     bool
	sizes    []int
	unpaged  bool
	prefixed bool
	actions  func(context.Context, R) []actions.Item
	bulk     func(context.Context) []actions.Item
	sealed   atomic.Bool
}

var (
	tableName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reserved  = []string{"q", "sort", "page", "size", "selected"}
)

// New declares a table named name whose rows are keyed by key (text or
// an integer). Declaration mistakes panic (§3.1).
func New[R any](name string, key column.Column[R], cols ...Column[R]) *Table[R] {
	if !tableName.MatchString(name) {
		panic(fmt.Sprintf("tables: New(%q): a name is [a-z][a-z0-9-]*", name))
	}
	if len(cols) == 0 {
		panic(fmt.Sprintf("tables: %s: New with no columns", name))
	}
	if key.Kind != column.Text && key.Kind != column.Integer {
		panic(fmt.Sprintf("tables: %s: the key %q is %v, neither text nor an integer", name, key.Name, key.Kind))
	}
	seen := map[string]bool{}
	for _, c := range cols {
		n := c.c.Name
		renderers := 0
		for _, set := range []bool{c.link != nil, c.amount != nil, c.src != nil} {
			if set {
				renderers++
			}
		}
		switch {
		case seen[n]:
			panic(fmt.Sprintf("tables: %s: column %q listed twice", name, n))
		case c.filter && slices.Contains(reserved, n):
			panic(fmt.Sprintf("tables: %s: a filter on %q, a reserved parameter (S13)", name, n))
		case c.filter && !fieldName.MatchString(n):
			panic(fmt.Sprintf("tables: %s: a filter on %q, which forms can't take as a field's name", name, n))
		case c.searchable && c.c.Kind != column.Text:
			panic(fmt.Sprintf("tables: %s: Searchable on %q, which isn't text", name, n))
		case c.amount != nil && c.c.Kind != column.Integer && c.c.Kind != column.Number:
			panic(fmt.Sprintf("tables: %s: Amount on %q, which isn't an integer or a number", name, n))
		case renderers > 1:
			panic(fmt.Sprintf("tables: %s: column %q has more than one of Link, Amount and Image", name, n))
		case c.src != nil && c.alt == nil:
			panic(fmt.Sprintf("tables: %s: Image on %q with no alternative text", name, n))
		}
		seen[n] = true
	}
	return &Table[R]{name: name, key: key, cols: slices.Clone(cols), sizes: []int{20, 50, 100}}
}

func (t *Table[R]) builder(what string) {
	if t.sealed.Load() {
		panic(fmt.Sprintf("tables: %s: %s after the table's first use: declare a table once, at package level", t.name, what))
	}
}

// Label is the caption; required.
func (t *Table[R]) Label(l i18n.Ter) *Table[R] { t.builder("Label"); t.label = l; return t }

// RowLabel names the column whose text names a row's controls.
func (t *Table[R]) RowLabel(c column.Column[R]) *Table[R] {
	t.builder("RowLabel")
	if _, ok := t.column(c.Name); !ok {
		panic(fmt.Sprintf("tables: %s: RowLabel(%q) isn't a column of the table", t.name, c.Name))
	}
	t.rowLabel = c.Name
	return t
}

// Sort is the order when the URL names none; its column must be Sortable.
func (t *Table[R]) Sort(c column.Column[R], d Dir) *Table[R] {
	t.builder("Sort")
	tc, ok := t.column(c.Name)
	if !ok || !tc.sortable {
		panic(fmt.Sprintf("tables: %s: Sort(%q): not a sortable column of the table", t.name, c.Name))
	}
	t.sort, t.desc = c.Name, bool(d)
	return t
}

// Sizes sets the page sizes on offer, the first the default.
func (t *Table[R]) Sizes(n ...int) *Table[R] {
	t.builder("Sizes")
	if len(n) == 0 || slices.ContainsFunc(n, func(x int) bool { return x < 1 }) {
		panic(fmt.Sprintf("tables: %s: Sizes(%v): at least one size, each above 0", t.name, n))
	}
	t.sizes = slices.Clone(n)
	return t
}

// Unpaged shows every row.
func (t *Table[R]) Unpaged() *Table[R] { t.builder("Unpaged"); t.unpaged = true; return t }

// Prefixed puts every parameter under "<name>.", for a page of two tables.
func (t *Table[R]) Prefixed() *Table[R] { t.builder("Prefixed"); t.prefixed = true; return t }

// Actions gives each row its actions.
func (t *Table[R]) Actions(f func(context.Context, R) []actions.Item) *Table[R] {
	t.builder("Actions")
	t.actions = f
	return t
}

// Bulk gives the table actions on its selected rows.
func (t *Table[R]) Bulk(f func(context.Context) []actions.Item) *Table[R] {
	t.builder("Bulk")
	t.bulk = f
	return t
}

func (t *Table[R]) column(name string) (Column[R], bool) {
	i := slices.IndexFunc(t.cols, func(c Column[R]) bool { return c.c.Name == name })
	if i < 0 {
		return Column[R]{}, false
	}
	return t.cols[i], true
}

// labelColumn is the row label's column: RowLabel, else the first shown
// text column; "" when there is none.
func (t *Table[R]) labelColumn() string {
	if t.rowLabel != "" {
		c, _ := t.column(t.rowLabel)
		if c.hidden || c.c.Kind != column.Text {
			return ""
		}
		return t.rowLabel
	}
	for _, c := range t.cols {
		if !c.hidden && c.c.Kind == column.Text {
			return c.c.Name
		}
	}
	return ""
}

// Check reports what needs the whole declaration: no Label, a column
// without a label, Remote on a column without Filter() or of a kind other
// than Text or Integer, or a table with actions or bulk actions and no row
// label, which names a row's controls.
func (t *Table[R]) Check() error {
	if t.label == nil {
		return fmt.Errorf("tables: %s has no Label: its caption names it", t.name)
	}
	for _, c := range t.cols {
		if c.c.Label == nil {
			return fmt.Errorf("tables: %s: column %q has no label", t.name, c.c.Name)
		}
		if c.labeler != nil {
			if !c.filter {
				return fmt.Errorf("tables: %s: Remote on %q without Filter()", t.name, c.c.Name)
			}
			if c.c.Kind != column.Text && c.c.Kind != column.Integer {
				return fmt.Errorf("tables: %s: Remote on %q, which isn't text or an integer", t.name, c.c.Name)
			}
		}
	}
	if (t.actions != nil || t.bulk != nil) && t.labelColumn() == "" {
		return fmt.Errorf("tables: %s has actions and no row label: RowLabel a shown text column, or show one", t.name)
	}
	return nil
}

// use seals the table and panics on what Check reports.
func (t *Table[R]) use() {
	if t.sealed.Swap(true) {
		return
	}
	if err := t.Check(); err != nil {
		t.sealed.Store(false)
		panic(err.Error())
	}
}

func (t *Table[R]) param(p string) string {
	if t.prefixed {
		return t.name + "." + p
	}
	return p
}

// ID is the table's id, "<name>".
func (t *Table[R]) ID() string { return t.name }

// RowID is a row's id, "<name>-row-<key>", the key escaped so the id is
// one token and a valid selector (§3.1).
func (t *Table[R]) RowID(key string) string { return t.name + "-row-" + escape(key) }

func (t *Table[R]) resultsID() string { return t.name + "-results" }

// escape keeps [A-Za-z0-9_-] and writes every other byte as _xx.
func escape(s string) string {
	var b strings.Builder
	for i := range len(s) {
		c := s[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "_%02x", c)
		}
	}
	return b.String()
}

// SelectionName is the parameter the selection's checkboxes post under:
// "selected", or "<name>.selected" on a prefixed table [T6]. A form that
// acts on the selection carries the keys on under it.
func (t *Table[R]) SelectionName() string { return t.param("selected") }

// Selected reads the checked rows' keys: from the body for a POST, from
// the query for a GET (a bulk Open).
func (t *Table[R]) Selected(r *http.Request) []string {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return slices.Clone(r.URL.Query()[t.SelectionName()])
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
		return nil
	}
	return slices.Clone(r.PostForm[t.SelectionName()])
}

// Targeted reports whether an htmx request targets the table or its
// results, so a page of several tables can query only that one.
func Targeted[R any](r *http.Request, t *Table[R]) bool {
	if r.Header.Get("HX-Request") != "true" {
		return false
	}
	h := r.Header.Get("HX-Target")
	return h == t.ID() || h == t.resultsID()
}

// filter is one filterable column's fields.
type filter[R any] struct {
	c          Column[R]
	text       *forms.Field[string]
	i64        *forms.Field[int64]
	b          *forms.Field[bool]
	ifrom, ito *forms.Field[int]
	ffrom, fto *forms.Field[float64]
	dfrom, dto *forms.Field[time.Time]
}

func (f filter[R]) params(t *Table[R]) []string {
	n := t.param(f.c.c.Name)
	if f.c.labeler != nil {
		return []string{n}
	}
	switch f.c.c.Kind {
	case column.Integer, column.Number, column.Date, column.Time:
		return []string{n + ".from", n + ".to"}
	}
	return []string{n}
}

func rangeCheck[T any](g *forms.Form, from, to *forms.Field[T], after func(a, b T) bool) {
	g.Check(func(context.Context) error {
		a, okA := from.Get()
		b, okB := to.Get()
		if okA && okB && after(a, b) {
			return ErrRange
		}
		return nil
	})
}

var (
	fromLabel = i18n.Key("ui.components.table.from", "From")
	toLabel   = i18n.Key("ui.components.table.to", "To")
)

func (t *Table[R]) filterForm() (*forms.Form, []filter[R]) {
	build := func(g *forms.Form) []filter[R] {
		var out []filter[R]
		for _, c := range t.cols {
			if !c.filter {
				continue
			}
			x := filter[R]{c: c}
			n, l := c.c.Name, c.c.Label
			switch {
			case c.labeler != nil && c.c.Kind == column.Text:
				x.text = g.Text(n).Label(l).Remote(c.search, c.labeler)
			case c.labeler != nil && c.c.Kind == column.Integer:
				x.i64 = g.Number[int64](n).Label(l).Remote(c.search, c.labeler)
			case c.c.Kind == column.Text:
				x.text = g.Text(n).Label(l)
			case c.c.Kind == column.Enum:
				// A select of the column's options, which refuses a value not
				// offered; the value is the option's string (finding 11).
				x.text = g.Text(n).Label(l).Options(c.c.Options...)
			case c.c.Kind == column.Bool:
				x.b = g.Bool(n).Label(l)
			case c.c.Kind == column.Integer:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[int] {
					r.Label(l)
					a, b := r.Int("from").Label(fromLabel), r.Int("to").Label(toLabel)
					rangeCheck(r, a, b, func(x, y int) bool { return x > y })
					return [2]*forms.Field[int]{a, b}
				})
				x.ifrom, x.ito = p[0], p[1]
			case c.c.Kind == column.Number:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[float64] {
					r.Label(l)
					a, b := r.Float("from").Label(fromLabel), r.Float("to").Label(toLabel)
					rangeCheck(r, a, b, func(x, y float64) bool { return x > y })
					return [2]*forms.Field[float64]{a, b}
				})
				x.ffrom, x.fto = p[0], p[1]
			case c.c.Kind == column.Date, c.c.Kind == column.Time:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[time.Time] {
					r.Label(l)
					a, b := r.Date("from").Label(fromLabel), r.Date("to").Label(toLabel)
					rangeCheck(r, a, b, func(x, y time.Time) bool { return x.After(y) })
					return [2]*forms.Field[time.Time]{a, b}
				})
				x.dfrom, x.dto = p[0], p[1]
			}
			out = append(out, x)
		}
		return out
	}
	f := forms.New()
	if t.prefixed {
		return f, f.Group(t.name, build)
	}
	return f, build(f)
}
