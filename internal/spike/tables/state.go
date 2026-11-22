package tables

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/spike/col"
)

// State is a request's state of a table: search, filters, sort, page and
// size, bound from the query string. It is a listing.Source.
type State[R any] struct {
	t       *Table[R]
	ctx     context.Context
	path    string
	query   url.Values
	search  string
	sort    string
	desc    bool
	page    int
	size    int
	form    *forms.Form
	filters []filter[R]
	valid   bool
}

// Bind reads r's query string.
func (t *Table[R]) Bind(r *http.Request) *State[R] {
	q := r.URL.Query()
	st := &State[R]{t: t, ctx: r.Context(), path: r.URL.Path, query: q, page: 1}
	st.search = strings.TrimSpace(q.Get(t.param("q")))
	if utf8.RuneCountInString(st.search) > 200 {
		st.search = string([]rune(st.search)[:200])
	}
	st.sort, st.desc = t.sort, t.desc
	if s := q.Get(t.param("sort")); s != "" {
		name, desc := strings.CutPrefix(s, "-")
		if c, ok := t.column(name); ok && c.sortable {
			st.sort, st.desc = name, desc
		}
	}
	if n, err := strconv.Atoi(q.Get(t.param("page"))); err == nil && n > 1 {
		st.page = n
	}
	st.size = t.sizes[0]
	if n, err := strconv.Atoi(q.Get(t.param("size"))); err == nil && slices.Contains(t.sizes, n) {
		st.size = n
	}
	st.form, st.filters = t.filterForm()
	st.form.Bind(forms.FromValues(q))
	st.valid = st.form.Validate(r.Context())
	return st
}

// Valid is false when a filter's value doesn't parse.
func (st *State[R]) Valid() bool { return st.valid }

// Size is the rows a page; 0 when unpaged.
func (st *State[R]) Size() int {
	if st.t.unpaged {
		return 0
	}
	return st.size
}

// ListingQuery is the query models runs.
func (st *State[R]) ListingQuery() listing.Query {
	q := listing.Query{Search: st.search}
	for _, c := range st.t.cols {
		if c.searchable {
			q.SearchIn = append(q.SearchIn, c.c.Name)
		}
	}
	if q.Search == "" {
		q.SearchIn = nil
	}
	for _, f := range st.filters {
		n := f.c.c.Name
		add := func(op listing.Op, v any) { q.Filters = append(q.Filters, listing.Filter{Column: n, Op: op, Value: v}) }
		switch {
		case f.text != nil:
			if v, ok := f.text.Get(); ok && v != "" {
				add(listing.Eq, v)
			}
		case f.b != nil:
			if v, ok := f.b.Get(); ok {
				add(listing.Eq, v)
			}
		case f.ifrom != nil:
			if v, ok := f.ifrom.Get(); ok {
				add(listing.Ge, v)
			}
			if v, ok := f.ito.Get(); ok {
				add(listing.Le, v)
			}
		case f.ffrom != nil:
			if v, ok := f.ffrom.Get(); ok {
				add(listing.Ge, v)
			}
			if v, ok := f.fto.Get(); ok {
				add(listing.Le, v)
			}
		case f.dfrom != nil:
			if v, ok := f.dfrom.Get(); ok {
				add(listing.Ge, v)
			}
			if v, ok := f.dto.Get(); ok {
				if f.c.c.Kind == col.Time { // before the day after the last
					add(listing.Lt, v.AddDate(0, 0, 1))
				} else {
					add(listing.Le, v)
				}
			}
		}
	}
	if st.sort != "" {
		q.Sort = append(q.Sort, listing.Sort{Column: st.sort, Desc: st.desc})
	}
	// The key ends every sort, in the sort's direction, so paging is stable.
	if st.sort != st.t.key.Name {
		q.Sort = append(q.Sort, listing.Sort{Column: st.t.key.Name, Desc: st.desc})
	}
	if !st.t.unpaged {
		q.Limit = st.size
		q.Offset = (st.page - 1) * st.size
	}
	return q
}

// active reports whether any filter or the search is set.
func (st *State[R]) active() bool {
	if st.search != "" {
		return true
	}
	for _, f := range st.filters {
		for _, p := range f.params(st.t) {
			if strings.TrimSpace(st.query.Get(p)) != "" {
				return true
			}
		}
	}
	return false
}

// with is the current URL with params set (a "" value deletes it); a
// change to anything but the page goes back to page 1.
func (st *State[R]) with(set map[string]string) string {
	q := url.Values{}
	for k, v := range st.query {
		q[k] = slices.Clone(v)
	}
	if _, paging := set[st.t.param("page")]; !paging {
		q.Del(st.t.param("page"))
	}
	for k, v := range set {
		if v == "" {
			q.Del(k)
		} else {
			q.Set(k, v)
		}
	}
	if len(q) == 0 {
		return st.path
	}
	return st.path + "?" + q.Encode()
}

// Errors are the filters' errors.
func (st *State[R]) Errors() iter.Seq[*forms.FieldError] { return st.form.Errors() }
