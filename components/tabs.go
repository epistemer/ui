package components

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"
)

// TabsSpec is a row of navigation tabs (§7.4): the navigation pattern,
// links with aria-current, not APG Tabs. Children are Tab components.
//
//	@components.Tabs(label) {
//		@components.Tab("/stores/1").Current(here == "info") { Details }
//		@components.Tab("/stores/1/orders") { Orders @elements.Badge() { 12 } }
//	}
type TabsSpec struct {
	label i18n.Ter
	attrs templ.Attributes
}

// Tabs starts a row of tabs named label (the <nav>'s aria-label).
func Tabs(label i18n.Ter) TabsSpec { return TabsSpec{label: label} }

// Attrs adds attributes to the <nav>, merged as elements.Merge does.
func (t TabsSpec) Attrs(a templ.Attributes) TabsSpec {
	t.attrs = elements.Merge(t.attrs, nil, a)
	return t
}

func (t TabsSpec) Render(ctx context.Context, w io.Writer) error {
	if t.label == nil || strings.TrimSpace(t.label.T(ctx)) == "" {
		panic("components: Tabs with no label: a row of tabs needs its name")
	}
	own := templ.Attributes{"class": "ui-tabs", "aria-label": t.label.T(ctx)}
	return tabs(elements.Merge(own, []string{"aria-label"}, t.attrs)).Render(ctx, w)
}

// TabSpec is one tab: a link in its own list item. Children are its text,
// a count badge included.
type TabSpec struct {
	href    templ.SafeURL
	current bool
	attrs   templ.Attributes
}

// Tab starts a tab leading to href.
func Tab(href templ.SafeURL) TabSpec { return TabSpec{href: href} }

// Current marks the tab of the page shown (aria-current="page").
func (t TabSpec) Current(c bool) TabSpec { t.current = c; return t }

// Attrs adds attributes to the <a>, merged as elements.Merge does; the tab
// keeps href and aria-current.
func (t TabSpec) Attrs(a templ.Attributes) TabSpec {
	t.attrs = elements.Merge(t.attrs, nil, a)
	return t
}

func (t TabSpec) Render(ctx context.Context, w io.Writer) error {
	own := templ.Attributes{"class": "ui-tab", "href": t.href}
	if t.current {
		own["aria-current"] = "page"
	}
	return tab(elements.Merge(own, []string{"href", "aria-current"}, t.attrs)).Render(ctx, w)
}
