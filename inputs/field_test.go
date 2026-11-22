package inputs

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/forms"
)

var errAddress = errors.Key("errors.shop.address", "the address is incomplete", errors.Invalid)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	ctx := ui.WithIDs(context.Background(), ui.Sequence("t"))
	if err := c.Render(ctx, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestHiddenTakesAttrs(t *testing.T) {
	f := forms.New()
	f.Text("ref").Hint(forms.KindHidden)
	var c forms.Control
	for x := range f.Controls() {
		c = x
	}
	html := render(t, Field(c).Attrs(templ.Attributes{"form": "other", "name": "stolen"}))
	if !strings.Contains(html, `form="other"`) || strings.Contains(html, "stolen") || !strings.Contains(html, `name="ref"`) {
		t.Errorf("hidden field: %s", html)
	}
}

func TestGroupErrorIsDescribed(t *testing.T) {
	f := forms.New()
	g := f.Group("address", func(g *forms.Form) *forms.Form {
		g.Text("street").Label(i18n.Static("Street"))
		g.Check(func(context.Context) error { return errAddress })
		return g
	})
	g.Label(i18n.Static("Address"))
	f.Bind(forms.FromValues(url.Values{"address.street": {"x"}}))
	f.Validate(context.Background())
	var c forms.Control
	for x := range f.Controls() {
		c = x
	}
	html := render(t, Field(c))
	if !strings.Contains(html, `aria-describedby="field--t-1-error"`) || !strings.Contains(html, `id="field--t-1-error"`) {
		t.Errorf("group error not described: %s", html)
	}
}
