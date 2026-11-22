// Package tables is the spike of ui/tables (server-state-design.md §6):
// a table declared once from generated columns, its state bound from the
// URL, and a view that describes a page of rows. It renders nothing.
package tables

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/col"
)

// ErrRange is a range filter whose start is after its end.
var ErrRange = errors.Key("errors.tables.range", "the start is after the end", errors.Invalid)

// Dir is a sort's direction.
type Dir bool

const (
	Asc  Dir = false
	Desc Dir = true
)

// Column is a column of a table: the generated column and what the page
// says about it (S22).
type Column[R any] struct {
	c          col.Column[R]
	searchable bool
	sortable   bool
	filter     bool
	hidden     bool
	link       func(R) ui.Link
	amount     func(R) i18n.Ter
}

// Col starts a table column from a generated one.
func Col[R any](c col.Column[R]) Column[R] { return Column[R]{c: c} }

func (c Column[R]) Searchable() Column[R] { c.searchable = true; return c }
func (c Column[R]) Sortable() Column[R]   { c.sortable = true; return c }
func (c Column[R]) Filter() Column[R]     { c.filter = true; return c }
func (c Column[R]) Hidden() Column[R]     { c.hidden = true; return c }
func (c Column[R]) Link(f func(R) ui.Link) Column[R] {
	c.link = f
	return c
}
func (c Column[R]) Amount(f func(R) i18n.Ter) Column[R] {
	c.amount = f
	return c
}

// Table is declared once and only read afterwards.
type Table[R any] struct {
	name     string
	key      col.Column[R]
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
}

var (
	tableName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reserved  = []string{"q", "sort", "page", "size"}
)

// New declares a table named name whose rows are keyed by key.
func New[R any](name string, key col.Column[R], cols ...Column[R]) *Table[R] {
	if !tableName.MatchString(name) {
		panic(fmt.Sprintf("tables: New(%q): a name is [a-z][a-z0-9-]*", name))
	}
	if len(cols) == 0 {
		panic(fmt.Sprintf("tables: New(%q) with no columns", name))
	}
	if key.Kind != col.Text && key.Kind != col.Integer {
		panic(fmt.Sprintf("tables: New(%q): the key %q is neither text nor an integer", name, key.Name))
	}
	seen := map[string]bool{}
	for _, c := range cols {
		n := c.c.Name
		switch {
		case seen[n]:
			panic(fmt.Sprintf("tables: %s: column %q listed twice", name, n))
		case c.filter && slices.Contains(reserved, n):
			panic(fmt.Sprintf("tables: %s: a filter on %q, a reserved parameter (S13)", name, n))
		case c.filter && !fieldName.MatchString(n):
			panic(fmt.Sprintf("tables: %s: a filter on %q, which forms can't take as a field's name", name, n))
		case c.amount != nil && c.c.Kind != col.Integer && c.c.Kind != col.Number:
			panic(fmt.Sprintf("tables: %s: Amount on %q, which isn't a number", name, n))
		case c.searchable && c.c.Kind != col.Text:
			panic(fmt.Sprintf("tables: %s: Searchable on %q, which isn't text", name, n))
		}
		seen[n] = true
	}
	return &Table[R]{name: name, key: key, cols: cols, sizes: []int{20, 50, 100}}
}

func (t *Table[R]) Label(l i18n.Ter) *Table[R] { t.label = l; return t }

// RowLabel names the column whose text names a row.
func (t *Table[R]) RowLabel(c col.Column[R]) *Table[R] { t.rowLabel = c.Name; return t }

// Sort is the order when the URL names none.
func (t *Table[R]) Sort(c col.Column[R], d Dir) *Table[R] { t.sort, t.desc = c.Name, bool(d); return t }
func (t *Table[R]) Sizes(n ...int) *Table[R] {
	if len(n) == 0 || slices.ContainsFunc(n, func(x int) bool { return x <= 0 }) {
		panic(fmt.Sprintf("tables: %s: Sizes(%v): at least one size, each above 0", t.name, n))
	}
	t.sizes = n
	return t
}
func (t *Table[R]) Unpaged() *Table[R]  { t.unpaged = true; return t }
func (t *Table[R]) Prefixed() *Table[R] { t.prefixed = true; return t }
func (t *Table[R]) Actions(f func(context.Context, R) []actions.Item) *Table[R] {
	t.actions = f
	return t
}
func (t *Table[R]) Bulk(f func(context.Context) []actions.Item) *Table[R] { t.bulk = f; return t }

// Ids, keyed from the name (§6.6).
func (t *Table[R]) ID() string              { return t.name }
func (t *Table[R]) StateID() string         { return t.name + "-state" }
func (t *Table[R]) ResultsID() string       { return t.name + "-results" }
func (t *Table[R]) SelectID() string        { return t.name + "-select" }
func (t *Table[R]) CaptionID() string       { return t.name + "-caption" }
func (t *Table[R]) StatusID() string        { return t.name + "-status" }
func (t *Table[R]) RowID(key string) string { return t.name + "-row-" + key }
func (t *Table[R]) sortID(c string) string {
	return t.name + "-sort-" + strings.ReplaceAll(c, "_", "-")
}
func (t *Table[R]) pageID(p string) string { return t.name + "-page-" + p }
func (t *Table[R]) param(p string) string {
	if t.prefixed {
		return t.name + "." + p
	}
	return p
}

// Targeted reports whether an htmx request targets the table or its
// results, so the handler may render the table alone.
func Targeted[R any](r *http.Request, t *Table[R]) bool {
	h := r.Header.Get("HX-Target")
	return h == t.ID() || h == t.ResultsID()
}

func (t *Table[R]) column(name string) (Column[R], bool) {
	i := slices.IndexFunc(t.cols, func(c Column[R]) bool { return c.c.Name == name })
	if i < 0 {
		return Column[R]{}, false
	}
	return t.cols[i], true
}

// filter is one filterable column's fields.
type filter[R any] struct {
	c          Column[R]
	text       *forms.Field[string]
	b          *forms.Field[bool]
	ifrom, ito *forms.Field[int]
	ffrom, fto *forms.Field[float64]
	dfrom, dto *forms.Field[time.Time]
}

func (f filter[R]) params(t *Table[R]) []string {
	n := t.param(f.c.c.Name)
	switch f.c.c.Kind {
	case col.Integer, col.Number, col.Date, col.Time:
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

func (t *Table[R]) filterForm() (*forms.Form, []filter[R]) {
	f := forms.New()
	var fs []filter[R]
	build := func(g *forms.Form) []filter[R] {
		var out []filter[R]
		for _, c := range t.cols {
			if !c.filter {
				continue
			}
			x := filter[R]{c: c}
			n := c.c.Name
			switch c.c.Kind {
			case col.Text:
				x.text = g.Text(n).Label(c.c.Label)
			case col.Enum:
				x.text = g.Text(n).Label(c.c.Label).Options(c.c.Options...)
			case col.Bool:
				x.b = g.Bool(n).Label(c.c.Label)
			case col.Integer:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[int] {
					r.Label(c.c.Label)
					a, b := r.Int("from").Label(fromLabel), r.Int("to").Label(toLabel)
					rangeCheck(r, a, b, func(x, y int) bool { return x > y })
					return [2]*forms.Field[int]{a, b}
				})
				x.ifrom, x.ito = p[0], p[1]
			case col.Number:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[float64] {
					r.Label(c.c.Label)
					a, b := r.Float("from").Label(fromLabel), r.Float("to").Label(toLabel)
					rangeCheck(r, a, b, func(x, y float64) bool { return x > y })
					return [2]*forms.Field[float64]{a, b}
				})
				x.ffrom, x.fto = p[0], p[1]
			case col.Date, col.Time:
				p := g.Group(n, func(r *forms.Form) [2]*forms.Field[time.Time] {
					r.Label(c.c.Label)
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
	if t.prefixed {
		fs = f.Group(t.name, build)
	} else {
		fs = build(f)
	}
	return f, fs
}

// Hand-written labels of a range's ends (a spike's; the contract names keys).
var (
	fromLabel = i18n.Key("ui.components.table.from", "From")
	toLabel   = i18n.Key("ui.components.table.to", "To")
)

// HasBulk and HasActions report whether rows get a checkbox and a menu.
func (t *Table[R]) HasBulk() bool    { return t.bulk != nil }
func (t *Table[R]) HasActions() bool { return t.actions != nil }
