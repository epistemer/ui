package icons

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

// Compile-time checks: IconSpec implements templ.Component, and every
// pre-A9 icon name and Labelled still exist.
var (
	_ templ.Component = IconSpec{}
	_                 = []func() IconSpec{
		X, Plus, Minus, Check, ChevronDown, ChevronUp, ChevronStart, ChevronEnd,
		ArrowStart, ArrowEnd, More, Menu, Search, Info, Warning, CircleCheck,
		CircleX, Eye, EyeOff, Sun, Moon, User,
	}
	_ func(templ.Component, i18n.Ter) templ.Component = Labelled
)

func renderIcon(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestOldIconsByteForByte(t *testing.T) {
	cases := []struct {
		name string
		fn   func() IconSpec
		want string
	}{
		{"X", X, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18"></path> <path d="m6 6 12 12"></path></svg>`},
		{"Plus", Plus, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"></path> <path d="M12 5v14"></path></svg>`},
		{"Minus", Minus, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"></path></svg>`},
		{"Check", Check, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg>`},
		{"ChevronDown", ChevronDown, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 9 6 6 6-6"></path></svg>`},
		{"ChevronUp", ChevronUp, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m18 15-6-6-6 6"></path></svg>`},
		{"ChevronStart", ChevronStart, `<svg class="ui-icon" data-mirror viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6"></path></svg>`},
		{"ChevronEnd", ChevronEnd, `<svg class="ui-icon" data-mirror viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6"></path></svg>`},
		{"ArrowStart", ArrowStart, `<svg class="ui-icon" data-mirror viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m12 19-7-7 7-7"></path> <path d="M19 12H5"></path></svg>`},
		{"ArrowEnd", ArrowEnd, `<svg class="ui-icon" data-mirror viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14"></path> <path d="m12 5 7 7-7 7"></path></svg>`},
		{"More", More, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="1"></circle> <circle cx="19" cy="12" r="1"></circle> <circle cx="5" cy="12" r="1"></circle></svg>`},
		{"Menu", Menu, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 5h16"></path> <path d="M4 12h16"></path> <path d="M4 19h16"></path></svg>`},
		{"Search", Search, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m21 21-4.34-4.34"></path> <circle cx="11" cy="11" r="8"></circle></svg>`},
		{"Info", Info, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle> <path d="M12 16v-4"></path> <path d="M12 8h.01"></path></svg>`},
		{"Warning", Warning, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"></path> <path d="M12 9v4"></path> <path d="M12 17h.01"></path></svg>`},
		{"CircleCheck", CircleCheck, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle> <path d="m16 9-5.5 5.5L8 12"></path></svg>`},
		{"CircleX", CircleX, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle> <path d="m15 9-6 6"></path> <path d="m9 9 6 6"></path></svg>`},
		{"Eye", Eye, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"></path> <circle cx="12" cy="12" r="3"></circle></svg>`},
		{"EyeOff", EyeOff, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M10.733 5.076a10.744 10.744 0 0 1 11.205 6.575 1 1 0 0 1 0 .696 10.747 10.747 0 0 1-1.444 2.49"></path> <path d="M14.084 14.158a3 3 0 0 1-4.242-4.242"></path> <path d="M17.479 17.499a10.75 10.75 0 0 1-15.417-5.151 1 1 0 0 1 0-.696 10.75 10.75 0 0 1 4.446-5.143"></path> <path d="m2 2 20 20"></path></svg>`},
		{"Sun", Sun, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="4"></circle> <path d="M12 2v2"></path> <path d="M12 20v2"></path> <path d="m4.93 4.93 1.41 1.41"></path> <path d="m17.66 17.66 1.41 1.41"></path> <path d="M2 12h2"></path> <path d="M20 12h2"></path> <path d="m6.34 17.66-1.41 1.41"></path> <path d="m19.07 4.93-1.41 1.41"></path></svg>`},
		{"Moon", Moon, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.985 12.486a9 9 0 1 1-9.473-9.472c.405-.022.617.46.402.803a6 6 0 0 0 8.268 8.268c.344-.215.825-.004.803.401"></path></svg>`},
		{"User", User, `<svg class="ui-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"></path> <circle cx="12" cy="7" r="4"></circle></svg>`},
	}

	for _, tc := range cases {
		got := renderIcon(t, tc.fn())
		if got != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}

	if renderIcon(t, More()) != renderIcon(t, Ellipsis()) {
		t.Error("More() != Ellipsis()")
	}
	if renderIcon(t, Warning()) != renderIcon(t, TriangleAlert()) {
		t.Error("Warning() != TriangleAlert()")
	}
	if strings.Contains(renderIcon(t, ArrowLeft()), "data-mirror") {
		t.Error("ArrowLeft() should not carry data-mirror")
	}
	if strings.Contains(renderIcon(t, ArrowRight()), "data-mirror") {
		t.Error("ArrowRight() should not carry data-mirror")
	}
}

func TestIconOptions(t *testing.T) {
	// Size
	for _, tc := range []struct {
		s    Size
		want string
	}{
		{SizeText, ""},
		{SizeS, ` data-size="s"`},
		{SizeM, ` data-size="m"`},
		{SizeL, ` data-size="l"`},
	} {
		got := renderIcon(t, Check().Size(tc.s))
		if tc.want == "" && strings.Contains(got, "data-size") {
			t.Errorf("Size(%d) rendered data-size: %s", tc.s, got)
		}
		if tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("Size(%d) missing %q in %s", tc.s, tc.want, got)
		}
	}

	// Color
	for _, tc := range []struct {
		c    Color
		want string
	}{
		{ColorCurrent, ""},
		{ColorMuted, ` data-color="muted"`},
		{ColorBrand, ` data-color="brand"`},
		{ColorDanger, ` data-color="danger"`},
		{ColorWarning, ` data-color="warning"`},
		{ColorSuccess, ` data-color="success"`},
		{ColorInfo, ` data-color="info"`},
	} {
		got := renderIcon(t, Check().Color(tc.c))
		if tc.want == "" && strings.Contains(got, "data-color") {
			t.Errorf("Color(%d) rendered data-color: %s", tc.c, got)
		}
		if tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("Color(%d) missing %q in %s", tc.c, tc.want, got)
		}
	}

	// Stroke
	if got := renderIcon(t, Check().Stroke(1.5)); !strings.Contains(got, `stroke-width="1.5"`) {
		t.Errorf("Stroke(1.5): %s", got)
	}
	if got := renderIcon(t, Check().Stroke(2)); !strings.Contains(got, `stroke-width="2"`) {
		t.Errorf("Stroke(2): %s", got)
	}

	// Label
	labelled := renderIcon(t, Check().Label(i18n.Static("Verified")))
	if strings.Contains(labelled, "aria-hidden") || !strings.Contains(labelled, `role="img" aria-label="Verified"`) {
		t.Errorf("Label: %s", labelled)
	}

	// Attrs merge + immutability + owned protection
	base := Check().Size(SizeS).Attrs(templ.Attributes{"class": "base"})
	custom := base.Color(ColorBrand).Attrs(templ.Attributes{
		"CLASS":       "extra base",
		"Data-Id":     "42",
		"viewBox":     "0 0 10 10",
		"data-size":   "l",
		"data-color":  "danger",
		"aria-hidden": "false",
	})
	got := renderIcon(t, custom)
	for _, want := range []string{`class="ui-icon base extra"`, `data-size="s"`, `data-color="brand"`, `data-id="42"`, `viewBox="0 0 24 24"`, `aria-hidden="true"`} {
		if !strings.Contains(got, want) {
			t.Errorf("Attrs missing %s in %s", want, got)
		}
	}
	if strings.Contains(got, "danger") || strings.Contains(got, "0 0 10 10") {
		t.Errorf("Attrs leaked owned attribute: %s", got)
	}
	if gotBase := renderIcon(t, base); strings.Contains(gotBase, "extra") || strings.Contains(gotBase, "data-color") {
		t.Errorf("base mutated: %s", gotBase)
	}

	// New 1.77.0 icons used by Confab exist and render.
	for _, fn := range []func() IconSpec{Pencil, Trash2, ExternalLink, Upload, Download, Calendar} {
		if got := renderIcon(t, fn()); !strings.HasPrefix(got, `<svg class="ui-icon"`) {
			t.Errorf("unexpected markup: %s", got)
		}
	}
}

func TestIconPanics(t *testing.T) {
	mustPanicFn := func(name string, fn func()) {
		t.Helper()
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s: expected panic, rendered normally", name)
				return
			}
			if msg, ok := r.(string); !ok || !strings.HasPrefix(msg, "icons:") {
				t.Errorf("%s: panic %v should start with \"icons:\"", name, r)
			}
		}()
		fn()
	}
	mustPanic := func(name string, c templ.Component) {
		t.Helper()
		mustPanicFn(name, func() {
			c.Render(context.Background(), &strings.Builder{})
		})
	}

	mustPanic("zero IconSpec", IconSpec{})
	mustPanic("bad Size", Check().Size(Size(99)))
	mustPanic("bad Color", Check().Color(Color(99)))
	mustPanic("Stroke 0", Check().Stroke(0))
	mustPanic("Stroke negative", Check().Stroke(-1))
	mustPanic("Stroke > 4", Check().Stroke(4.5))
	mustPanic("Stroke NaN", Check().Stroke(math.NaN()))
	mustPanic("Stroke +Inf", Check().Stroke(math.Inf(1)))
	mustPanic("Label nil", Check().Label(nil))
	mustPanic("Label empty", Check().Label(i18n.Static("")))
	mustPanic("Label blank", Check().Label(i18n.Static("   ")))
	mustPanicFn("Attrs non-string class", func() {
		Check().Attrs(templ.Attributes{"class": 123}).Render(context.Background(), &strings.Builder{})
	})
	mustPanic("Labelled nil icon", Labelled(nil, i18n.Static("x")))
	mustPanic("Labelled blank text", Labelled(Check(), i18n.Static(" ")))
}
