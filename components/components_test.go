package components

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(ui.WithIDs(context.Background(), ui.Sequence("t")), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func panics(t *testing.T, what string, c templ.Component) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s rendered", what)
		}
	}()
	c.Render(ui.WithIDs(context.Background(), ui.Sequence("t")), &strings.Builder{})
}

func TestPanics(t *testing.T) {
	panics(t, "a dialog without a body", Dialog(i18n.Static("x")))
	panics(t, "a dialog without a title", Dialog(nil))
	panics(t, "a menu without items", Menu(i18n.Static("x")))
	panics(t, "a submitting link item", MenuItem().Href("/x").Submit("f"))
	panics(t, "a toast in brand", Toast(i18n.Static("x")).Tone(elements.Brand))
	panics(t, "a toast without text", Toast(i18n.Static(" ")))
	panics(t, "tabs without a name", Tabs(nil))
	panics(t, "a detail row without a label", DetailRow(nil))
	defer func() {
		if recover() == nil {
			t.Error("Confirm with no question")
		}
	}()
	Confirm(context.Background(), i18n.Static(""))
}

func TestOwned(t *testing.T) {
	html := render(t, MenuItem().Submit("f").Attrs(templ.Attributes{"type": "button", "role": "x", "class": "mine"}))
	for _, want := range []string{`type="submit"`, `form="f"`, `role="menuitem"`, `class="ui-menu-item mine"`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s: %s", want, html)
		}
	}
	html = render(t, MenuItem().Href("/x").Disabled(true))
	if strings.Contains(html, "href") || !strings.Contains(html, `aria-disabled="true"`) {
		t.Errorf("disabled link item: %s", html)
	}
	html = render(t, Tab("/a").Current(true).Attrs(templ.Attributes{"href": "/b", "aria-current": "false"}))
	if !strings.Contains(html, `href="/a"`) || !strings.Contains(html, `aria-current="page"`) {
		t.Errorf("tab: %s", html)
	}
	html = render(t, Toast(i18n.Static("Saved")).Tone(elements.Warning))
	if strings.Contains(html, "data-ui-timeout") || !strings.Contains(html, `data-tone="warning"`) {
		t.Errorf("a warning toast times out: %s", html)
	}
	if html := render(t, Toast(i18n.Static("Saved"))); !strings.Contains(html, `data-ui-timeout="6000"`) {
		t.Errorf("an info toast doesn't time out: %s", html)
	}
}
