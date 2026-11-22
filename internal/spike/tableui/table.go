package tableui

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/col"
	"github.com/epistemer/ui/internal/spike/tables"
)

var (
	searchLabel  = i18n.Key("ui.components.table.search", "Search")
	applyLabel   = i18n.Key("ui.components.table.apply", "Apply")
	clearLabel   = i18n.Key("ui.components.table.clear", "Clear filters")
	removeLabel  = i18n.Key("ui.components.table.remove", "Remove filter")
	selectLabel  = i18n.Key("ui.components.table.select", "Select")
	selectAll    = i18n.Key("ui.components.table.select_all", "Select every row on this page")
	bulkLabel    = i18n.Key("ui.components.table.bulk", "With the selected rows")
	noRows       = i18n.Key("ui.components.table.empty", "Nothing here yet.")
	noMatch      = i18n.Key("ui.components.table.no_match", "Nothing matches these filters.")
	pastEnd      = i18n.Key("ui.components.table.past_end", "This page is past the last one.")
	goLast       = i18n.Key("ui.components.table.go_last", "Go to the last page")
	selectedText = i18n.Key("ui.components.table.selected", "%d selected")
)

// TableView is a table's rendering, declared once so a page and an
// action's answer render a row alike (§6.6).
type TableView[R any] struct {
	t     *tables.Table[R]
	cells map[string]func(R) templ.Component
}

// TableOf starts a table's rendering.
func TableOf[R any](t *tables.Table[R]) *TableView[R] {
	return &TableView[R]{t: t, cells: map[string]func(R) templ.Component{}}
}

// Cell replaces one column's cell.
func (tv *TableView[R]) Cell(c col.Column[R], f func(R) templ.Component) *TableView[R] {
	tv.cells[c.Name] = f
	return tv
}

// Page renders the whole table.
func (tv *TableView[R]) Page(v tables.View[R]) templ.Component {
	return tableTmpl(tv.data(v))
}

// Row renders one row, for an action's answer.
func (tv *TableView[R]) Row(ctx context.Context, row R) templ.Component {
	r := tv.t.Row(ctx, row)
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return rowTmpl(tv.tableIDs(), tv.row(r), tv.t.HasBulk(), tv.t.HasActions()).Render(ctx, w)
	})
}

// data is the view without its row type, for templ.
type data struct {
	ids        ids
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
	sort       [2]string
	pager      tables.Pager
	bulk       []actions.Item
	empty      tables.Empty
	emptyLink  templ.SafeURL
	status     string
	hasActions bool
	own        string
}

type ids struct{ table, state, results, sel, caption, status string }

type rowData struct {
	id, key, label, labelID string
	cells                   []cellData
	actions                 []actions.Item
}

type cellData struct {
	c       tables.Cell
	custom  templ.Component
	labelID string
}

func (tv *TableView[R]) tableIDs() ids {
	t := tv.t
	return ids{t.ID(), t.StateID(), t.ResultsID(), t.SelectID(), t.CaptionID(), t.StatusID()}
}

func (tv *TableView[R]) data(v tables.View[R]) data {
	d := data{
		ids: tv.tableIDs(), label: v.Label, url: v.URL, path: strings.SplitN(v.URL, "?", 2)[0],
		rejected: v.Rejected, heads: v.Columns, search: v.Search, filters: v.Filters, chips: v.Chips,
		clear: v.Clear, hidden: v.Hidden, sort: v.Sort, pager: v.Pager, bulk: v.Bulk,
		empty: v.Empty, emptyLink: v.EmptyLink, status: v.Status, hasActions: tv.t.HasActions(),
		own: strings.Join(v.Own, " "),
	}
	for _, r := range v.Rows {
		d.rows = append(d.rows, tv.row(r))
	}
	return d
}

func (tv *TableView[R]) row(r tables.Row[R]) rowData {
	rd := rowData{id: r.ID, key: r.Key, label: r.Label, labelID: r.ID + "-label", actions: r.Actions}
	for _, c := range r.Cells {
		cd := cellData{c: c}
		if f, ok := tv.cells[c.Column]; ok {
			cd.custom = f(r.Row)
		}
		if c.Label {
			cd.labelID = rd.labelID
		}
		rd.cells = append(rd.cells, cd)
	}
	return rd
}

// swap is what a control that changes the table's state carries with htmx
// (§6.8): it targets the results, or the whole table for the state form.
func swap(d data, target string, get string) templ.Attributes {
	return templ.Attributes{
		"hx-get":      get,
		"hx-target":   "#" + target,
		"hx-select":   "#" + target,
		"hx-swap":     "outerHTML",
		"hx-push-url": "true",
		"hx-sync":     "#" + d.ids.table + ":replace",
	}
}

func linkSwap(d data, href templ.SafeURL) templ.Attributes {
	return swap(d, d.ids.results, string(href))
}

func searchAttrs(d data) templ.Attributes {
	a := swap(d, d.ids.results, d.path)
	a["hx-trigger"] = "input changed delay:300ms, search"
	a["hx-include"] = "#" + d.ids.state
	return a
}

func ariaSort(h tables.Head) string {
	switch {
	case !h.Sorted:
		return ""
	case h.Desc:
		return "descending"
	}
	return "ascending"
}

// marked splits text around the search's hits, case ignored.
func marked(text, q string) []struct {
	Text string
	Hit  bool
} {
	var out []struct {
		Text string
		Hit  bool
	}
	if q == "" {
		return append(out, struct {
			Text string
			Hit  bool
		}{text, false})
	}
	lt, lq := strings.ToLower(text), strings.ToLower(q)
	// Byte offsets in the lowered text are offsets in the text only when
	// lowering changed no length; otherwise, no marks (review 0014).
	if len(lt) != len(text) || len(lq) != len(q) {
		return append(out, struct {
			Text string
			Hit  bool
		}{text, false})
	}
	for {
		i := strings.Index(lt, lq)
		if i < 0 {
			out = append(out, struct {
				Text string
				Hit  bool
			}{text, false})
			return out
		}
		if i > 0 {
			out = append(out, struct {
				Text string
				Hit  bool
			}{text[:i], false})
		}
		out = append(out, struct {
			Text string
			Hit  bool
		}{text[i : i+len(q)], true})
		text, lt = text[i+len(q):], lt[i+len(q):]
		if text == "" {
			return out
		}
	}
}

func kindName(k col.Kind) string {
	switch k {
	case col.Integer, col.Number:
		return "number"
	case col.Bool:
		return "bool"
	case col.Date, col.Time:
		return "date"
	case col.Enum:
		return "enum"
	}
	return "text"
}

func headAttrs(h tables.Head) templ.Attributes {
	if s := ariaSort(h); s != "" {
		return templ.Attributes{"aria-sort": s}
	}
	return nil
}

func cellAttrs(c cellData) templ.Attributes {
	if c.labelID != "" {
		return templ.Attributes{"id": c.labelID}
	}
	return nil
}

func current(c bool) templ.Attributes {
	if c {
		return templ.Attributes{"aria-current": "true"}
	}
	return nil
}
