package tables

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/links"
	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/col"
)

// View describes a page of a table, and holds no request.
type View[R any] struct {
	ID, StateID, ResultsID, SelectID, CaptionID, StatusID string

	Label    i18n.Ter
	URL      string // the current address, for a refresh
	Rejected bool   // a filter didn't parse: the form and its errors only

	Columns []Head
	Rows    []Row[R]

	Search  Search
	Filters []forms.Control
	Chips   []Chip
	Clear   templ.SafeURL // removes the search and every filter; "" when none is set
	Hidden  [][2]string   // the query's other parameters, for the state form
	Sort    [2]string     // the sort's parameter and value

	Pager Pager
	Bulk  []actions.Item

	Empty     Empty
	EmptyLink templ.SafeURL

	Status string // the sentence said after a change

	// Own are the query parameters the table owns, so the script can take
	// every other one from the live address (a spike finding).
	Own []string
}

// Head is a column's head.
type Head struct {
	Name     string
	Label    i18n.Ter
	Kind     col.Kind
	Sortable bool
	Sorted   bool
	Desc     bool
	Link     templ.SafeURL // sorts by it next
	ID       string
}

// Row is one row.
type Row[R any] struct {
	Row     R
	Key, ID string
	Label   string // names the row's controls
	Cells   []Cell
	Actions []actions.Item
}

// Cell is one cell.
type Cell struct {
	Column string
	Kind   col.Kind
	Value  col.Value
	Link   templ.SafeURL
	Amount i18n.Ter
	Mark   string // the search text to mark, in a searchable text cell
	Label  bool   // the row's label cell
}

// Search is the search box.
type Search struct{ Name, Text string }

// Chip is one active filter.
type Chip struct {
	Label  i18n.Ter
	Value  string
	Remove templ.SafeURL
}

// Empty says why there are no rows.
type Empty int

const (
	NotEmpty Empty = iota
	NoRows         // none at all
	NoMatch        // none for these filters: EmptyLink clears them
	PastEnd        // a page past the last: EmptyLink goes to the last
)

// Pager is the pagination's description.
type Pager struct {
	Label                           i18n.Ter
	From, To, Total                 int
	Page, Pages                     int
	First, Prev, Next, Last         templ.SafeURL // "" at an end
	FirstID, PrevID, NextID, LastID string
	Window                          []PageLink
	Sizes                           []SizeLink
	SizeParam                       string
	Unpaged                         bool
}

// PageLink is a numbered page.
type PageLink struct {
	N       int
	URL     templ.SafeURL
	Current bool
	ID      string
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
)

func (st *State[R]) base() View[R] {
	t := st.t
	v := View[R]{
		ID: t.ID(), StateID: t.StateID(), ResultsID: t.ResultsID(), SelectID: t.SelectID(),
		CaptionID: t.CaptionID(), StatusID: t.StatusID(),
		Label: t.label, URL: st.with(map[string]string{t.param("page"): st.query.Get(t.param("page"))}),
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
	// The query's parameters the state form doesn't hold go in it hidden.
	own := map[string]bool{t.param("q"): true, t.param("sort"): true, t.param("page"): true, t.param("size"): true}
	for _, f := range st.filters {
		for _, p := range f.params(t) {
			own[p] = true
		}
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
	for _, f := range st.filters {
		for _, p := range f.params(t) {
			if val := strings.TrimSpace(st.query.Get(p)); val != "" {
				v.Chips = append(v.Chips, Chip{Label: f.c.c.Label, Value: chipText(st.ctx, f.c, p, val), Remove: templ.SafeURL(st.with(map[string]string{p: ""}))})
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

func chipText[R any](ctx context.Context, c Column[R], param, val string) string {
	if c.c.Kind == col.Enum {
		for _, o := range c.c.Options {
			if o.Value() == val {
				val = o.Label().T(ctx)
			}
		}
	}
	switch {
	case strings.HasSuffix(param, ".from"):
		return "≥ " + val
	case strings.HasSuffix(param, ".to"):
		return "≤ " + val
	}
	return val
}

// Rejected is the view of a state that isn't valid: the filter form with
// its errors, and no rows, empty state or pager (S12).
func (st *State[R]) Rejected() View[R] {
	v := st.base()
	v.Rejected = true
	v.Columns = st.heads()
	return v
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

// Page is the view of rows, the page of total.
func (st *State[R]) Page(rows []R, total int) View[R] {
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
		if v.Pager.Pages == 1 {
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

func (st *State[R]) pager(total int) Pager {
	t := st.t
	p := Pager{Label: pagerLabel{t.label}, Total: total, Page: st.page, Unpaged: t.unpaged, SizeParam: t.param("size")}
	if t.unpaged {
		p.From, p.To, p.Pages = min(1, total), total, 1
		return p
	}
	p.Pages = max(1, (total+st.size-1)/st.size)
	if st.page <= p.Pages && total > 0 {
		p.From = (st.page-1)*st.size + 1
		p.To = min(total, st.page*st.size)
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
	lo, hi := max(1, st.page-2), min(p.Pages, st.page+2)
	for n := lo; n <= hi; n++ {
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

// Row describes one row outside any page, for an action's answer.
func (t *Table[R]) Row(ctx context.Context, r R) Row[R] { return t.row(ctx, r, "") }

func (t *Table[R]) row(ctx context.Context, r R, search string) Row[R] {
	key := keyText(t.key.Get(r))
	row := Row[R]{Row: r, Key: key, ID: t.RowID(key)}
	labelCol := t.rowLabel
	if labelCol == "" {
		for _, c := range t.cols {
			if !c.hidden && c.c.Kind == col.Text {
				labelCol = c.c.Name
				break
			}
		}
	}
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

func keyText(v col.Value) string {
	if v.Text != "" {
		return v.Text
	}
	return fmt.Sprint(v.Int)
}

// pagerLabel names a table's pager after the table, so two pagers on a
// page are distinct landmarks (a spike finding).
type pagerLabel struct{ table i18n.Ter }

func (p pagerLabel) T(ctx context.Context, _ ...any) string {
	if p.table == nil {
		return pagesLabel.T(ctx, "")
	}
	return pagesLabel.T(ctx, p.table)
}
