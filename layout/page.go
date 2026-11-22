package layout

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/links"
)

// Crumb is one step of a page's breadcrumbs.
type Crumb struct {
	Href ui.Link // no address (visual-interface.md §2.7) only for the current page, the last crumb
	Text i18n.Ter
}

// PageHeaderSpec is a page's header being described (§8.4): breadcrumbs,
// the page's <h1>, a description, and the children, the page's actions,
// which render only when they write something.
type PageHeaderSpec struct {
	title       i18n.Ter
	description i18n.Ter
	crumbs      []Crumb
	attrs       templ.Attributes
}

// PageHeader starts a header titled title.
func PageHeader(title i18n.Ter) PageHeaderSpec { return PageHeaderSpec{title: title} }

// Description adds a line under the title.
func (p PageHeaderSpec) Description(t i18n.Ter) PageHeaderSpec { p.description = t; return p }

// Breadcrumbs sets the trail above the title, from the top down; the last
// crumb is the page itself (aria-current="page").
func (p PageHeaderSpec) Breadcrumbs(c ...Crumb) PageHeaderSpec {
	p.crumbs = slices.Clone(c)
	return p
}

// Attrs adds attributes to the <header>, merged as elements.Merge does.
func (p PageHeaderSpec) Attrs(a templ.Attributes) PageHeaderSpec {
	p.attrs = elements.Merge(p.attrs, nil, a)
	return p
}

func (p PageHeaderSpec) Render(ctx context.Context, w io.Writer) error {
	if p.title == nil || strings.TrimSpace(p.title.T(ctx)) == "" {
		panic("layout: PageHeader with no title: a page needs its heading")
	}
	// Each crumb's address is resolved once, before anything is written.
	hrefs := make([]templ.SafeURL, len(p.crumbs))
	for i, c := range p.crumbs {
		if c.Text == nil || strings.TrimSpace(c.Text.T(ctx)) == "" {
			panic("layout: PageHeader with a crumb that has no text")
		}
		hrefs[i] = links.Href(c.Href)
		if hrefs[i] == "" && i < len(p.crumbs)-1 {
			panic("layout: PageHeader with a crumb above the page that has no address: only the last crumb, the page itself, may have none")
		}
	}
	// The actions render only when the children write something, so an
	// empty block leaves no empty box.
	var actions bytes.Buffer
	children, ctx := templ.GetChildren(ctx), templ.ClearChildren(ctx)
	if err := children.Render(ctx, &actions); err != nil {
		return err
	}
	if strings.TrimSpace(actions.String()) == "" {
		actions.Reset()
	}
	hasDescription := p.description != nil && strings.TrimSpace(p.description.T(ctx)) != ""
	return pageHeader(p, hrefs, templ.Raw(actions.String()), actions.Len() > 0, hasDescription).Render(ctx, w)
}
