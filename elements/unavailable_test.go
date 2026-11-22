package elements

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
)

func TestUnavailable(t *testing.T) {
	b := Button().Href(ui.URL("/x")).Submit().Tone(Danger).Variant(Outline).
		Attrs(templ.Attributes{"form": "f", "formaction": "/y", "hx-post": "/y", "data-ui-confirm": "q", "class": "mine", "aria-describedby": "note"}).
		Unavailable(i18n.Static("only a paid order can be refunded"))
	html := render(t, b)
	for _, gone := range []string{"href", `form="f"`, "formaction", "hx-post", "data-ui-confirm", `type="submit"`} {
		if strings.Contains(html, gone) {
			t.Errorf("an unavailable button keeps %s:\n%s", gone, html)
		}
	}
	for _, want := range []string{`aria-disabled="true"`, `type="button"`, `data-tone="danger"`, `class="ui-button mine"`, "only a paid order can be refunded", "ui-tooltip"} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s:\n%s", want, html)
		}
	}
	if !strings.Contains(html, `aria-describedby="note tooltip--`) && !strings.Contains(html, `aria-describedby="tooltip--`) {
		t.Errorf("not described by its reason:\n%s", html)
	}
	if strings.Contains(html, " disabled") {
		t.Error("an unavailable button is natively disabled, so it can't be focused")
	}
	panics(t, "a blank reason", Button().Unavailable(i18n.Static(" ")))
}
