package components

import (
	"context"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/tables"
)

var (
	tableSearch    = i18n.Key("ui.components.table.search", "Search")
	tableApply     = i18n.Key("ui.components.table.apply", "Apply")
	tableClear     = i18n.Key("ui.components.table.clear", "Clear filters")
	tableRemove    = i18n.Key("ui.components.table.remove", "Remove filter")
	tableSelect    = i18n.Key("ui.components.table.select", "Select")
	tableSelectAll = i18n.Key("ui.components.table.select_all", "Select every row on this page")
	tableSelected  = i18n.Key("ui.components.table.selected", "%d selected")
	tableBulk      = i18n.Key("ui.components.table.bulk", "With the selected rows")
	tableEmpty     = i18n.Key("ui.components.table.empty", "Nothing here yet.")
	tableNoMatch   = i18n.Key("ui.components.table.no_match", "Nothing matches these filters.")
	tablePastEnd   = i18n.Key("ui.components.table.past_end", "This page is past the last one.")
	tableGoLast    = i18n.Key("ui.components.table.go_last", "Go to the last page")
	actionsLabel   = i18n.Key("ui.components.actions.label", "Actions")
)

// TableView is a table's rendering (server-state-interface.md §4.1),
// declared once at package level, so a page and an action's answer render
// a row alike.
type TableView[R any] struct {
	t     *tables.Table[R]
	cells map[string]func(R) templ.Component
	attrs templ.Attributes
}

// TableOf starts a table's rendering.
func TableOf[R any](t *tables.Table[R]) *TableView[R] {
	return &TableView[R]{t: t, cells: map[string]func(R) templ.Component{}}
}

// Cell replaces one column's cell, as a toned Badge for an enum.
func (tv *TableView[R]) Cell(c column.Column[R], f func(R) templ.Component) *TableView[R] {
	tv.cells[c.Name] = f
	return tv
}

// Attrs adds attributes to <ui-table>, merged as elements.Merge does.
func (tv *TableView[R]) Attrs(a templ.Attributes) *TableView[R] {
	tv.attrs = elements.Merge(tv.attrs, nil, a)
	return tv
}

// Page renders the whole table.
func (tv *TableView[R]) Page(v tables.View[R]) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if v.Err != nil {
			return v.Err
		}
		return tableTmpl(tv.data(v)).Render(ctx, w)
	})
}

// Row renders one row, for an action's answer, resolved with the render's
// context.
func (tv *TableView[R]) Row(row R) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		r := tv.t.Row(ctx, row)
		return rowTmpl(tv.ids(), tv.row(r), r.Selectable, r.Actions != nil).Render(ctx, w)
	})
}

// tableData is a view without its row type, for the templates.
type tableData struct {
	ids        tableIDs
	attrs      templ.Attributes
	own        string
	label      i18n.Ter
	url, path  string
	rejected   bool
	heads      []tables.Head
	rows       []rowData
	search     tables.Search
	filters    []forms.Control
	chips      []tables.Chip
	clear      templ.SafeURL
	hidden     [][2]string
	sort, size [2]string
	pager      tables.Pager
	bulk       []actions.Item
	empty      tables.Empty
	emptyLink  templ.SafeURL
	status     string
	selectable bool
	hasActions bool
}

type tableIDs struct{ table, state, results, sel, caption, status, selection string }

type rowData struct {
	id, key, labelID string
	cells            []cellData
	actions          []actions.Item
}

type cellData struct {
	c       tables.Cell
	custom  templ.Component
	labelID string
}

func (tv *TableView[R]) ids() tableIDs {
	n := tv.t.ID()
	return tableIDs{n, n + "-state", n + "-results", n + "-select", n + "-caption", n + "-status", tv.t.SelectionName()}
}

func (tv *TableView[R]) data(v tables.View[R]) tableData {
	d := tableData{
		ids: tv.ids(), attrs: tv.attrs, own: strings.Join(v.Own, " "), label: v.Label,
		url: v.URL, path: strings.SplitN(v.URL, "?", 2)[0], rejected: v.Rejected,
		heads: v.Columns, search: v.Search, chips: v.Chips, clear: v.Clear, hidden: v.Hidden,
		sort: v.Sort, size: v.Size, pager: v.Pager, bulk: v.Bulk, empty: v.Empty,
		emptyLink: v.EmptyLink, status: v.Status,
	}
	d.filters = v.Filters
	for _, r := range v.Rows {
		d.rows = append(d.rows, tv.row(r))
		d.selectable = d.selectable || r.Selectable
		d.hasActions = d.hasActions || r.Actions != nil
	}
	return d
}

func (tv *TableView[R]) row(r tables.Row[R]) rowData {
	rd := rowData{id: r.ID, key: r.Key, labelID: r.LabelID, actions: r.Actions}
	for _, c := range r.Cells {
		cd := cellData{c: c}
		if f, ok := tv.cells[c.Column]; ok {
			cd.custom = f(r.Row)
		}
		if c.Label {
			cd.labelID = r.LabelID
		}
		rd.cells = append(rd.cells, cd)
	}
	return rd
}

// swapAttrs are a state control's htmx attributes (§4.1): they target the
// results, or the whole table for the state form, and replace a request
// still in flight for the same table.
func swapAttrs(table, target, get string) templ.Attributes {
	return templ.Attributes{
		"hx-get":      get,
		"hx-target":   "#" + target,
		"hx-select":   "#" + target,
		"hx-swap":     "outerHTML",
		"hx-push-url": "true",
		"hx-sync":     "#" + table + ":replace",
	}
}

func (d tableData) link(u templ.SafeURL) templ.Attributes {
	return swapAttrs(d.ids.table, d.ids.results, string(u))
}

func (d tableData) searchAttrs() templ.Attributes {
	a := swapAttrs(d.ids.table, d.ids.results, d.path)
	a["hx-trigger"] = "input changed delay:300ms, search"
	a["hx-include"] = "#" + d.ids.state
	return a
}

func headAttrs(h tables.Head) templ.Attributes {
	a := templ.Attributes{"scope": "col", "data-kind": kindName(h.Kind)}
	switch {
	case h.Sorted && h.Desc:
		a["aria-sort"] = "descending"
	case h.Sorted:
		a["aria-sort"] = "ascending"
	}
	return a
}

func cellAttrs(c cellData) templ.Attributes {
	a := templ.Attributes{"data-kind": kindName(c.c.Kind)}
	if c.labelID != "" {
		a["id"] = c.labelID
	}
	return a
}

func kindName(k column.Kind) string {
	switch k {
	case column.Integer, column.Number:
		return "number"
	case column.Bool:
		return "bool"
	case column.Date, column.Time:
		return "date"
	case column.Enum:
		return "enum"
	}
	return "text"
}

// numberText is an integer's or a number's text: a numeric's exact
// digits when it has them.
func numberText(c tables.Cell) string {
	switch {
	case c.Kind == column.Integer:
		return strconv.FormatInt(c.Value.Int, 10)
	case c.Value.Text != "":
		return c.Value.Text
	}
	return strconv.FormatFloat(c.Value.Num, 'f', -1, 64)
}

type piece struct {
	Text string
	Hit  bool
}

// marked splits text around the search's hits, case ignored; no marks
// where lowering a rune changes its length in bytes.
func marked(text, q string) []piece {
	if q == "" || !utf8.ValidString(text) || !utf8.ValidString(q) || !sameLengths(text) || !sameLengths(q) {
		return []piece{{text, false}}
	}
	lt, lq := strings.ToLower(text), strings.ToLower(q)
	var out []piece
	for {
		i := strings.Index(lt, lq)
		if i < 0 {
			if text != "" {
				out = append(out, piece{text, false})
			}
			return out
		}
		if i > 0 {
			out = append(out, piece{text[:i], false})
		}
		out = append(out, piece{text[i : i+len(q)], true})
		text, lt = text[i+len(q):], lt[i+len(q):]
	}
}

// sameLengths reports whether lowering s keeps every rune's length in
// bytes, so offsets in the lowered text are offsets in s.
func sameLengths(s string) bool {
	for _, r := range s {
		if utf8.RuneLen(unicode.ToLower(r)) != utf8.RuneLen(r) {
			return false
		}
	}
	return true
}

// bulkAttrs make a bulk item a submit button of the selection form (§4.1).
func bulkAttrs(ctx context.Context, d tableData, it actions.Item, n int) templ.Attributes {
	a := templ.Attributes{
		"id":         d.ids.table + "-bulk-" + strconv.Itoa(n),
		"form":       d.ids.sel,
		"formaction": string(it.URL),
		"hx-include": "#" + d.ids.sel,
	}
	if it.Kind == actions.Open {
		a["formmethod"] = "get"
	} else if it.Method != "POST" {
		a["formmethod"] = "post"
	}
	return merge(a, htmxAttrs(ctx, it))
}

func bulkButton(ctx context.Context, d tableData, it actions.Item, n int) elements.ButtonSpec {
	bt := elements.Button().Variant(elements.Outline).Tone(elements.Neutral)
	switch {
	case it.Primary:
		bt = elements.Button()
	case it.Danger:
		bt = bt.Tone(elements.Danger)
	}
	bt = bt.Attrs(it.Attrs)
	if it.Reason != nil {
		return bt.Unavailable(reasonTer{it.Reason}).Attrs(templ.Attributes{"id": d.ids.table + "-bulk-" + strconv.Itoa(n)})
	}
	return bt.Submit().Attrs(bulkAttrs(ctx, d, it, n))
}

func rowMenu(r rowData) ActionMenuSpec {
	return ActionMenu(actionsLabel, r.actions).ID(r.id + "-actions").LabelledBy(r.labelID)
}

func imageLink(u templ.SafeURL) ui.Link { return ui.URL(string(u)) }
