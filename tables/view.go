package tables

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"

	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/links"
)

// View describes a page of a table and holds no request (§3.5). Every part
// is exported for an app that renders the table itself.
type View[R any] struct {
	ID, StateID, ResultsID, SelectID, CaptionID, StatusID string

	Label     i18n.Ter
	URL       string   // the current address, which a refresh reloads
	Own       []string // the query parameters the table owns
	Rejected  bool
	Columns   []Head
	Rows      []Row[R]
	Search    Search
	Filters   []forms.Control
	Chips     []Chip
	Clear     templ.SafeURL // removes the search and every filter; "" when none is set
	Hidden    [][2]string   // the query's other parameters, for the state form
	Sort      [2]string     // the sort's parameter and value; "" for the table's own order
	Size      [2]string     // the size's parameter and value; "" for the first size
	Pager     Pager
	Bulk      []actions.Item
	Empty     Empty
	EmptyLink templ.SafeURL
	Status    string // the sentence said after a change
}

// Head is a column's head.
type Head struct {
	Name     string
	Label    i18n.Ter
	Kind     column.Kind
	Sortable bool
	Sorted   bool
	Desc     bool
	Link     templ.SafeURL // sorts by it next
	ID       string        // "<table>-sort-<name>"
}

// Row is one row.
type Row[R any] struct {
	Row        R
	Key, ID    string // ID: "<table>-row-<key>"
	Label      string // the row label cell's text
	LabelID    string // "<table>-row-<key>-label"
	Selectable bool   // the table has bulk actions
	Cells      []Cell
	Actions    []actions.Item
}

// Cell is one cell.
type Cell struct {
	Column string
	Kind   column.Kind
	Value  column.Value
	Link   templ.SafeURL
	Amount i18n.Ter
	Src    templ.SafeURL // an Image column's address
	Alt    i18n.Ter      // its alternative text
	Mark   string        // the search text, in a searchable text cell
	Label  bool          // the row's label cell
}

// Search is the search box.
type Search struct{ Name, Text string }

// Chip is one active filter.
type Chip struct {
	Label  i18n.Ter
	Value  string // the bound value, formatted
	Remove templ.SafeURL
}

// Empty says why there are no rows.
type Empty int

const (
	NotEmpty Empty = iota
	NoRows         // none at all
	NoMatch        // none for these filters; EmptyLink clears them
	PastEnd        // a page past the last; EmptyLink goes to the last
)

// Pager is the pagination's description.
type Pager struct {
	Label                           i18n.Ter // "Pages of <table>"
	From, To                        int
	Total                           int64
	Page, Pages                     int
	First, Prev, Next, Last         templ.SafeURL // "" at an end
	FirstID, PrevID, NextID, LastID string
	Window                          []PageLink
	Sizes                           []SizeLink
	Unpaged                         bool
	Target                          string // the results' id its links swap; "" for none
}

// PageLink is a numbered page.
type PageLink struct {
	N       int
	URL     templ.SafeURL
	Current bool
	ID      string // "<table>-page-<n>"
}

// SizeLink is a page size on offer.
type SizeLink struct {
	N       int
	URL     templ.SafeURL
	Current bool
}

var (
	statusRange = i18n.Key("ui.components.table.status", "%d–%d of %d")
	statusNone  = i18n.Key("ui.components.table.none", "No results")
	pagesLabel  = i18n.Key("ui.components.pagination.label", "Pages of %s")
	boolYes     = i18n.Key("ui.elements.bool.yes", "Yes")
	boolNo      = i18n.Key("ui.elements.bool.no", "No")
)

// pagerLabel names a pager after its table: two pagers on a page are two
// landmarks.
type pagerLabel struct{ table i18n.Ter }

func (p pagerLabel) T(ctx context.Context, _ ...any) string { return pagesLabel.T(ctx, p.table) }

func (t *Table[R]) sortID(c string) string { return t.name + "-sort-" + escape(c) }
func (t *Table[R]) pageID(p string) string { return t.name + "-page-" + p }

func (st *State[R]) base() View[R] {
	t := st.t
	v := View[R]{
		ID: t.ID(), StateID: t.name + "-state", ResultsID: t.resultsID(), SelectID: t.name + "-select",
		CaptionID: t.name + "-caption", StatusID: t.name + "-status",
		Label:  t.label,
		URL:    st.with(map[string]string{t.param("page"): st.query.Get(t.param("page"))}), // page as bound
		Search: Search{Name: t.param("q"), Text: st.search},
		Own:    []string{t.param("q"), t.param("sort"), t.param("page"), t.param("size")},
	}
	for _, f := range st.filters {
		v.Own = append(v.Own, f.params(t)...)
	}
	for c := range st.form.Controls() {
		if t.prefixed {
			for cc := range c.Children() {
				v.Filters = append(v.Filters, cc)
			}
			continue
		}
		v.Filters = append(v.Filters, c)
	}
	own := map[string]bool{}
	for _, p := range v.Own {
		own[p] = true
	}
	keys := make([]string, 0, len(st.query))
	for k := range st.query {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if own[k] {
			continue
		}
		for _, val := range st.query[k] {
			v.Hidden = append(v.Hidden, [2]string{k, val})
		}
	}
	sortValue := ""
	if st.sort != "" && (st.sort != t.sort || st.desc != t.desc) {
		sortValue = st.sort
		if st.desc {
			sortValue = "-" + sortValue
		}
	}
	v.Sort = [2]string{t.param("sort"), sortValue}
	sizeValue := ""
	if st.size != t.sizes[0] {
		sizeValue = strconv.Itoa(st.size)
	}
	v.Size = [2]string{t.param("size"), sizeValue}
	if st.valid {
		for _, f := range st.filters {
			for _, ch := range chips(st.ctx, f, t) {
				ch.Remove = templ.SafeURL(st.with(map[string]string{ch.param: ""}))
				v.Chips = append(v.Chips, ch.Chip)
			}
		}
	}
	if st.active() {
		clear := map[string]string{t.param("q"): ""}
		for _, f := range st.filters {
			for _, p := range f.params(t) {
				clear[p] = ""
			}
		}
		v.Clear = templ.SafeURL(st.with(clear))
	}
	return v
}

type chip struct {
	Chip
	param string
}

// chips are a filter's set values as people read them, from what the
// form bound: an option's label, a bool's word, a date as Date renders it,
// a number; a range's end says which.
func chips[R any](ctx context.Context, f filter[R], t *Table[R]) []chip {
	params := f.params(t)
	one := func(param, val string) chip { return chip{Chip{Label: f.c.c.Label, Value: val}, param} }
	end := func(param, prefix, val string) chip { return one(param, prefix+" "+val) }
	date := func(d time.Time) string { return i18n.StaticDate(d).T(ctx) }
	var out []chip
	switch {
	case f.text != nil:
		if v, ok := f.text.Get(); ok && v != "" {
			for _, o := range f.c.c.Options {
				if o.Value() == v {
					v = o.Label().T(ctx)
				}
			}
			out = append(out, one(params[0], v))
		}
	case f.b != nil:
		if v, ok := f.b.Get(); ok {
			word := boolNo.T(ctx)
			if v {
				word = boolYes.T(ctx)
			}
			out = append(out, one(params[0], word))
		}
	case f.ifrom != nil:
		if v, ok := f.ifrom.Get(); ok {
			out = append(out, end(params[0], "≥", strconv.Itoa(v)))
		}
		if v, ok := f.ito.Get(); ok {
			out = append(out, end(params[1], "≤", strconv.Itoa(v)))
		}
	case f.ffrom != nil:
		if v, ok := f.ffrom.Get(); ok {
			out = append(out, end(params[0], "≥", strconv.FormatFloat(v, 'f', -1, 64)))
		}
		if v, ok := f.fto.Get(); ok {
			out = append(out, end(params[1], "≤", strconv.FormatFloat(v, 'f', -1, 64)))
		}
	case f.dfrom != nil:
		if v, ok := f.dfrom.Get(); ok {
			out = append(out, end(params[0], "≥", date(v)))
		}
		if v, ok := f.dto.Get(); ok {
			out = append(out, end(params[1], "≤", date(v)))
		}
	}
	return out
}

func (st *State[R]) heads() []Head {
	t := st.t
	var hs []Head
	for _, c := range t.cols {
		if c.hidden {
			continue
		}
		h := Head{Name: c.c.Name, Label: c.c.Label, Kind: c.c.Kind, Sortable: c.sortable, ID: t.sortID(c.c.Name)}
		if c.sortable {
			h.Sorted = st.sort == c.c.Name
			h.Desc = h.Sorted && st.desc
			next := c.c.Name
			if h.Sorted && !st.desc {
				next = "-" + next
			}
			h.Link = templ.SafeURL(st.with(map[string]string{t.param("sort"): next}))
		}
		hs = append(hs, h)
	}
	return hs
}

// Rejected is the view of a state that isn't valid: the filters with
// their errors, and neither rows, an empty state nor a pager (S12).
func (st *State[R]) Rejected() View[R] {
	v := st.base()
	v.Rejected = true
	v.Columns = st.heads()
	v.Status = statusNone.T(st.ctx)
	return v
}

// Page is the view of rows, a page of total, as List<Type> gives them.
func (st *State[R]) Page(rows []R, total int64) View[R] {
	t := st.t
	v := st.base()
	v.Columns = st.heads()
	for _, r := range rows {
		v.Rows = append(v.Rows, t.row(st.ctx, r, st.search))
	}
	if t.bulk != nil {
		v.Bulk = t.bulk(st.ctx)
	}
	v.Pager = st.pager(total)
	switch {
	case total == 0 && !st.active():
		v.Empty = NoRows
	case total == 0:
		v.Empty, v.EmptyLink = NoMatch, v.Clear
	case len(rows) == 0:
		v.Empty, v.EmptyLink = PastEnd, v.Pager.Last
		if v.EmptyLink == "" {
			v.EmptyLink = templ.SafeURL(st.with(map[string]string{}))
		}
	}
	if v.Empty == NotEmpty {
		v.Status = statusRange.T(st.ctx, v.Pager.From, v.Pager.To, v.Pager.Total)
	} else {
		v.Status = statusNone.T(st.ctx)
	}
	return v
}

func (st *State[R]) pager(total int64) Pager {
	t := st.t
	total = max(total, 0)
	p := Pager{Label: pagerLabel{t.label}, Total: total, Page: st.page, Unpaged: t.unpaged, Target: t.resultsID()}
	if t.unpaged {
		p.From, p.To, p.Pages = min(1, int(total)), int(total), 1
		return p
	}
	p.Pages = max(1, int((total+int64(st.size)-1)/int64(st.size)))
	if st.page <= p.Pages && total > 0 {
		p.From = (st.page-1)*st.size + 1
		p.To = int(min(total, int64(st.page)*int64(st.size)))
	}
	link := func(n int) templ.SafeURL {
		val := strconv.Itoa(n)
		if n == 1 {
			val = ""
		}
		return templ.SafeURL(st.with(map[string]string{t.param("page"): val}))
	}
	p.FirstID, p.PrevID, p.NextID, p.LastID = t.pageID("first"), t.pageID("prev"), t.pageID("next"), t.pageID("last")
	if st.page > 1 {
		p.First, p.Prev = link(1), link(min(st.page-1, p.Pages))
	}
	if st.page < p.Pages {
		p.Next, p.Last = link(st.page+1), link(p.Pages)
	}
	if st.page > p.Pages {
		p.Last = link(p.Pages)
	}
	for n := max(1, st.page-2); n <= min(p.Pages, st.page+2); n++ {
		p.Window = append(p.Window, PageLink{N: n, URL: link(n), Current: n == st.page, ID: t.pageID(strconv.Itoa(n))})
	}
	if len(t.sizes) > 1 {
		for _, n := range t.sizes {
			val := strconv.Itoa(n)
			if n == t.sizes[0] {
				val = ""
			}
			p.Sizes = append(p.Sizes, SizeLink{N: n, URL: templ.SafeURL(st.with(map[string]string{t.param("size"): val})), Current: n == st.size})
		}
	}
	return p
}

// Row describes one row outside any page, for an action's answer. Its
// first call seals the table, as Bind's does.
func (t *Table[R]) Row(ctx context.Context, r R) Row[R] {
	t.use()
	return t.row(ctx, r, "")
}

// keyText is a row's key, by the key column's kind: an empty text key is
// "", never "0".
func (t *Table[R]) keyText(v column.Value) string {
	if t.key.Kind == column.Text {
		return v.Text
	}
	return strconv.FormatInt(v.Int, 10)
}

func (t *Table[R]) row(ctx context.Context, r R, search string) Row[R] {
	key := t.keyText(t.key.Get(r))
	id := t.RowID(key)
	row := Row[R]{Row: r, Key: key, ID: id, LabelID: id + "-label", Selectable: t.bulk != nil}
	labelCol := t.labelColumn()
	for _, c := range t.cols {
		if c.hidden {
			continue
		}
		cell := Cell{Column: c.c.Name, Kind: c.c.Kind, Value: c.c.Get(r), Label: c.c.Name == labelCol}
		if c.link != nil {
			cell.Link = links.Href(c.link(r))
		}
		if c.amount != nil {
			cell.Amount = c.amount(r)
		}
		if c.src != nil {
			cell.Src, cell.Alt = links.Href(c.src(r)), c.alt(r)
		}
		if c.searchable {
			cell.Mark = search
		}
		if cell.Label {
			row.Label = cell.Value.Text
		}
		row.Cells = append(row.Cells, cell)
	}
	if t.actions != nil {
		row.Actions = t.actions(ctx, r)
	}
	return row
}
