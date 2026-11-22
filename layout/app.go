package layout

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/links"
)

// The app shell's slots (§8.2). AppMain is required; the others render
// what they're filled with, and an unfilled AppNav leaves out the
// navigation and its toggle.
var (
	AppBrand = ui.NewSlot()
	AppNav   = ui.NewSlot()
	AppBar   = ui.NewSlot()
	AppMain  = ui.NewSlot()
)

// AppSpec is the app shell being described: a bar with the brand, the
// navigation toggle and the bar's end; the navigation; and #ui-main. The
// confirmation dialog, the toast region, and the action form and dialog
// (amendment A6) follow, outside #ui-main.
//
// On narrow screens the navigation is a drawer from the start edge, a
// non-modal disclosure the ui script opens through the toggle's
// aria-expanded and closes on Esc, a click outside it, choosing an item,
// htmx:afterSwap and the screen widening. Until the script has run, the
// navigation is a plain block above the content. On wide screens it is a
// static column.
type AppSpec struct {
	attrs templ.Attributes
}

// App starts the app shell.
func App() AppSpec { return AppSpec{} }

// Attrs adds attributes to the outer <div class="ui-app">, merged as
// elements.Merge does.
func (a AppSpec) Attrs(at templ.Attributes) AppSpec {
	a.attrs = elements.Merge(a.attrs, nil, at)
	return a
}

func (a AppSpec) Render(ctx context.Context, w io.Writer) error {
	return ui.Slotted(app(a), AppBrand, AppNav, AppBar, AppMain).Render(ctx, w)
}

func requireMain(ctx context.Context) error {
	if !AppMain.Filled(ctx) {
		panic("layout: App without AppMain: fill it with the page's content")
	}
	return nil
}

// NavItemSpec is a link in the app's navigation. Children are its text
// and icon.
type NavItemSpec struct {
	href    templ.SafeURL
	current bool
	attrs   templ.Attributes
}

// NavItem starts a link to to. Rendering one with no address panics.
func NavItem(to ui.Link) NavItemSpec { return NavItemSpec{href: links.Href(to)} }

// Current marks the item as the page shown when c is true
// (aria-current="page").
func (n NavItemSpec) Current(c bool) NavItemSpec { n.current = c; return n }

// Attrs adds attributes to the <a>, merged as elements.Merge does; the
// item keeps href and aria-current.
func (n NavItemSpec) Attrs(a templ.Attributes) NavItemSpec {
	n.attrs = elements.Merge(n.attrs, nil, a)
	return n
}

var navItemOwned = []string{"href", "aria-current"}

func (n NavItemSpec) Render(ctx context.Context, w io.Writer) error {
	if n.href == "" {
		panic("layout: NavItem with no address: a navigation item is a link")
	}
	own := templ.Attributes{"class": "ui-nav-item", "href": n.href}
	if n.current {
		own["aria-current"] = "page"
	}
	return navLink(elements.Merge(own, navItemOwned, n.attrs)).Render(ctx, w)
}

var navSectionID = ui.NewID("nav-section")

// NavSectionSpec is a named group of navigation items. Children are the
// items.
type NavSectionSpec struct {
	label i18n.Ter
	attrs templ.Attributes
}

// NavSection starts a group named label.
func NavSection(label i18n.Ter) NavSectionSpec { return NavSectionSpec{label: label} }

// Attrs adds attributes to the group's <div>, merged as elements.Merge
// does; the group keeps role and aria-labelledby.
func (n NavSectionSpec) Attrs(a templ.Attributes) NavSectionSpec {
	n.attrs = elements.Merge(n.attrs, nil, a)
	return n
}

var navSectionOwned = []string{"role", "aria-labelledby"}

func (n NavSectionSpec) Render(ctx context.Context, w io.Writer) error {
	if n.label == nil || strings.TrimSpace(n.label.T(ctx)) == "" {
		panic("layout: NavSection with no label: a group needs its name")
	}
	return navSection(n).Render(ctx, w)
}

func (n NavSectionSpec) attributes(ctx context.Context) templ.Attributes {
	own := templ.Attributes{"class": "ui-nav-section", "role": "group", "aria-labelledby": navSectionID.Part(ctx, "label")}
	return elements.Merge(own, navSectionOwned, n.attrs)
}
