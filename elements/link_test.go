package elements

import (
	"strings"
	"testing"

	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
)

type route struct{ path string }

func (r route) URL() string { return r.path }

// No address (visual-interface.md §2.7): nil, empty, or refused by
// templ.URL.
func TestNoAddress(t *testing.T) {
	for name, l := range map[string]ui.Link{"nil": nil, "empty": ui.URL(""), "refused": ui.URL("javascript:alert(1)")} {
		if html := render(t, Button().Href(l)); strings.Contains(html, "href") || !strings.Contains(html, `<button`) {
			t.Errorf("%s: a button with no address isn't a button: %s", name, html)
		}
		if html := render(t, Link(l, i18n.Static("Docs"))); !strings.Contains(html, "ui-empty") {
			t.Errorf("%s: Link with no address isn't Empty: %s", name, html)
		}
		if html := render(t, Image(l, i18n.Static("Logo"))); !strings.Contains(html, "ui-empty") {
			t.Errorf("%s: Image with no address isn't Empty: %s", name, html)
		}
		if html := render(t, Avatar(i18n.Static("Layla Adel")).Src(l)); strings.Contains(html, "<img") || !strings.Contains(html, "LA") {
			t.Errorf("%s: an avatar with no address doesn't show initials: %s", name, html)
		}
	}
}

func TestRouteLink(t *testing.T) {
	html := render(t, Button().Href(route{"/orders/7?tab=items"}))
	if !strings.Contains(html, `href="/orders/7?tab=items"`) {
		t.Errorf("a route link: %s", html)
	}
}
