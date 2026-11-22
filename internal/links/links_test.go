package links

import (
	"testing"

	"github.com/a-h/templ"

	"github.com/epistemer/ui"
)

type route struct{ path string }

func (r route) URL() string { return r.path }

type ptrRoute struct{ path string }

func (r *ptrRoute) URL() string { return r.path }

func TestHref(t *testing.T) {
	for _, c := range []struct {
		l    ui.Link
		want templ.SafeURL
	}{
		{nil, ""},
		{ui.URL(""), ""},
		{ui.URL("  "), ""},
		{ui.URL("javascript:alert(1)"), ""},
		{ui.URL("/orders?page=2"), "/orders?page=2"},
		{ui.URL("https://example.com/a"), "https://example.com/a"},
		{ui.URL("mailto:shop@example.com"), "mailto:shop@example.com"},
		{route{"/orders/7"}, "/orders/7"},
		{(*ptrRoute)(nil), ""},
		{(*route)(nil), ""},
		{&ptrRoute{"/toys"}, "/toys"},
		{ui.URL("data:image/png;base64,AAAA"), ""},
	} {
		if got := Href(c.l); got != c.want {
			t.Errorf("Href(%v) = %q, want %q", c.l, got, c.want)
		}
	}
}
