package elements

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestMerge(t *testing.T) {
	own := templ.Attributes{"class": "ui-x", "type": "button", "aria-describedby": "a"}
	got := Merge(own, []string{"type"}, templ.Attributes{
		"CLASS": "mine ui-x", "Type": "submit", "aria-describedby": "b a", "hx-get": "/y",
	})
	want := templ.Attributes{"class": "ui-x mine", "type": "button", "aria-describedby": "a b", "hx-get": "/y"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %v", got)
	}
	if own["class"] != "ui-x" {
		t.Error("Merge changed its input")
	}
}

func TestMergeRejectsNonStringClass(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a non-string class was accepted")
		}
	}()
	Merge(templ.Attributes{"class": "ui-x"}, nil, templ.Attributes{"class": []string{"a"}})
}

func TestButtonOwnsItsAttributes(t *testing.T) {
	html := render(t, Button().Variant(Outline).Attrs(templ.Attributes{
		"data-variant": "ghost", "data-tone": "danger", "type": "submit", "role": "tab", "class": "mine",
	}))
	for _, want := range []string{`class="ui-button mine"`, `data-variant="outline"`, `type="button"`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in %s", want, html)
		}
	}
	for _, bad := range []string{"ghost", "danger", "submit", `role="tab"`} {
		if strings.Contains(html, bad) {
			t.Errorf("caller's %s reached %s", bad, html)
		}
	}
}

func TestButtonSpecIsImmutable(t *testing.T) {
	base := Button().Variant(Ghost).Attrs(templ.Attributes{"class": "base"})
	a := base.Attrs(templ.Attributes{"class": "a"}).Tone(Danger)
	b := base.Attrs(templ.Attributes{"class": "b"})
	ha, hb, hbase := render(t, a), render(t, b), render(t, base)
	if !strings.Contains(ha, `class="ui-button base a"`) || !strings.Contains(ha, `data-tone="danger"`) {
		t.Errorf("a: %s", ha)
	}
	if !strings.Contains(hb, `class="ui-button base b"`) || strings.Contains(hb, "danger") {
		t.Errorf("b: %s", hb)
	}
	if !strings.Contains(hbase, `class="ui-button base"`) {
		t.Errorf("base changed: %s", hbase)
	}
}

func TestButtonHrefSubmitPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a submitting link rendered")
		}
	}()
	render(t, Button().Href("/x").Submit())
}
