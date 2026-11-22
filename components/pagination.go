package components

import (
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/tables"
)

var (
	pageFirst = i18n.Key("ui.components.pagination.first", "First")
	pagePrev  = i18n.Key("ui.components.pagination.previous", "Previous")
	pageNext  = i18n.Key("ui.components.pagination.next", "Next")
	pageLast  = i18n.Key("ui.components.pagination.last", "Last")
	pageN     = i18n.Key("ui.components.pagination.page", "Page %d")
	pageSize  = i18n.Key("ui.components.pagination.size", "Rows a page")
	pageSizeN = i18n.Key("ui.components.pagination.size_n", "%d rows a page")
	pageRange = i18n.Key("ui.components.pagination.range", "%d–%d of %d")
)

// PaginationSpec is a table's pager being described (§4.1).
type PaginationSpec struct {
	p     tables.Pager
	attrs templ.Attributes
}

// Pagination renders a table's pager: a named <nav>, the range, first,
// previous, a window of numbered pages, next and last, and the sizes on
// offer. Its links swap the pager's Target when it has one. Building a
// pager without a table isn't offered yet.
func Pagination(p tables.Pager) PaginationSpec { return PaginationSpec{p: p} }

// Attrs adds attributes to the <nav>, merged as elements.Merge does; it
// keeps its name.
func (s PaginationSpec) Attrs(a templ.Attributes) PaginationSpec {
	s.attrs = elements.Merge(s.attrs, nil, a)
	return s
}

func (s PaginationSpec) Render(ctx context.Context, w io.Writer) error {
	return paginationTmpl(s.p, s.attrs).Render(ctx, w)
}

// pageLinkAttrs are a page or size link's attributes: aria-current when
// it's the current one, and the swap of the pager's Target.
func pageLinkAttrs(p tables.Pager, u templ.SafeURL, current bool, currentValue string) templ.Attributes {
	a := templ.Attributes{}
	if current && currentValue != "" {
		a["aria-current"] = currentValue
	}
	if p.Target != "" {
		table := strings.TrimSuffix(p.Target, "-results")
		for k, v := range swapAttrs(table, p.Target, string(u)) {
			a[k] = v
		}
	}
	return a
}

// sizeID keys a size link to its table, so focus finds it after a swap.
func sizeID(p tables.Pager, n int) templ.Attributes {
	if p.Target == "" {
		return nil
	}
	return templ.Attributes{"id": strings.TrimSuffix(p.Target, "-results") + "-size-" + strconv.Itoa(n)}
}
