package elements

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

func panics(t *testing.T, what string, c templ.Component) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s rendered", what)
		}
	}()
	c.Render(context.Background(), &strings.Builder{})
}

func TestPanics(t *testing.T) {
	panics(t, "an alert in brand", Alert().Tone(Brand))
	panics(t, "a neutral alert", Alert().Tone(Neutral))
	panics(t, "a heading of level 7", Heading(7))
	panics(t, "a heading of level 0", Heading(0))
	panics(t, "a tooltip without a trigger", Tooltip(i18n.Static("x")))
	panics(t, "an avatar without a name", Avatar(nil))
	panics(t, "an avatar with a blank name", Avatar(i18n.Static("  ")))
	panics(t, "a button of an unknown variant", Button().Variant("bogus"))
	panics(t, "a button of an unknown tone", Button().Tone("bogus"))
	panics(t, "a badge of an unknown tone", Badge().Tone("bogus"))
	panics(t, "a spinner of an unknown size", Spinner().Size("xl"))
	panics(t, "a button with a blank label", Button().Label(i18n.Static(" ")))
	panics(t, "a tooltip with blank text", Tooltip(i18n.Static("")))
}

func TestBlankTextsFallBack(t *testing.T) {
	if html := render(t, Alert().Title(i18n.Static(" "))); strings.Contains(html, "ui-alert-title") {
		t.Errorf("a blank title rendered: %s", html)
	}
	if html := render(t, Spinner().Label(i18n.Static(""))); !strings.Contains(html, "Loading") {
		t.Errorf("a blank spinner label didn't fall back: %s", html)
	}
}

func TestMergeOneSided(t *testing.T) {
	got := Merge(nil, nil, templ.Attributes{"aria-controls": " a b  a "})
	if got["aria-controls"] != "a b" {
		t.Errorf("caller only: %q", got["aria-controls"])
	}
	defer func() {
		if recover() == nil {
			t.Error("a non-string class in own was accepted")
		}
	}()
	Merge(templ.Attributes{"class": 1}, nil, nil)
}

func TestTones(t *testing.T) {
	cases := []struct {
		c    templ.Component
		want string // the data-tone, or "" for none
	}{
		{Button(), ""},
		{Button().Tone(Brand), ""},
		{Button().Tone(Neutral), "neutral"},
		{Badge(), ""},
		{Badge().Tone(Neutral), ""},
		{Badge().Tone(Brand), "brand"},
		{Alert(), ""},
		{Alert().Tone(Info), ""},
		{Alert().Tone(Danger), "danger"},
	}
	for i, c := range cases {
		html := render(t, c.c)
		got := ""
		if i := strings.Index(html, `data-tone="`); i >= 0 {
			got = html[i+len(`data-tone="`):]
			got = got[:strings.IndexByte(got, '"')]
		}
		if got != c.want {
			t.Errorf("case %d: data-tone %q, want %q in %s", i, got, c.want, html)
		}
	}
}

func TestButtonLabel(t *testing.T) {
	html := render(t, Button().Label(i18n.Static("Close")).Attrs(templ.Attributes{"aria-label": "Other"}))
	if !strings.Contains(html, `aria-label="Close"`) || strings.Contains(html, "Other") {
		t.Errorf("Label isn't owned: %s", html)
	}
}

func TestInitials(t *testing.T) {
	for in, want := range map[string]string{"Nour Hassan": "NH", "omar": "O", "  layla  adel  fathy": "LA", "نور حسن": "نح", "": "", "e\u0301mile zola": "E\u0301Z"} {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMergeJoinsControls(t *testing.T) {
	got := Merge(templ.Attributes{"aria-controls": "a"}, nil, templ.Attributes{"aria-controls": "b a"})
	if got["aria-controls"] != "a b" {
		t.Errorf("aria-controls %q", got["aria-controls"])
	}
}
