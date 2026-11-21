package forms_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

type color string

func (c color) Value() string    { return string(c) }
func (c color) Label() i18n.TKer { return labels[c] }

var labels = map[color]i18n.TKer{
	"red":  i18n.Key("test.color.red", "Red"),
	"blue": i18n.Key("test.color.blue", "Blue"),
}

var errTaken = errors.Key("errors.test.taken", "is taken", errors.Conflict)

var bg = context.Background()

func bind(f *forms.Form, q string) {
	v, err := url.ParseQuery(q)
	if err != nil {
		panic(err)
	}
	f.Bind(forms.FromValues(v))
}

func TestPresence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		declare func(f *forms.Form) func() (any, bool)
		q       string
		present bool
		want    any
	}{
		{"text missing", text, "", false, ""},
		{"text blank", text, "x=+++", false, ""},
		{"text trimmed", text, "x=+a+", true, "a"},
		{"text last value", text, "x=a&x=b&x=", true, "b"},
		{"raw keeps space", func(f *forms.Form) func() (any, bool) { h := f.Text("x").Raw(); return get(h) }, "x=+a+", true, " a "},
		{"texts none", texts, "x=&x=+", false, []string(nil)},
		{"texts items", texts, "x=a&x=&x=b", true, []string{"a", "b"}},
		{"int", ints, "x=+42+", true, 42},
		{"int empty", ints, "x=", false, 0},
		{"float", floats, "x=1.5", true, 1.5},
		{"bool on", bools, "x=on", true, true},
		{"bool false", bools, "x=false", true, false},
		{"bool missing", bools, "", false, false},
		{"checkbox missing", boxes, "", true, false},
		{"checkbox on", boxes, "x=on", true, true},
		{"checkbox hidden then ticked", boxes, "x=false&x=true", true, true},
		{"checkbox hidden only", boxes, "x=false", true, false},
		{"date", dates, "x=2026-11-21", true, time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)},
		{"enum", enums, "x=red", true, color("red")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := forms.New()
			got := tc.declare(f)
			bind(f, tc.q)
			f.Validate(bg)
			v, ok := got()
			if ok != tc.present {
				t.Errorf("present %v, want %v", ok, tc.present)
			}
			if !equal(v, tc.want) {
				t.Errorf("value %#v, want %#v", v, tc.want)
			}
		})
	}
}

func get[T any](h *forms.Field[T]) func() (any, bool) {
	return func() (any, bool) { v, ok := h.Get(); return v, ok }
}
func text(f *forms.Form) func() (any, bool)   { return get(f.Text("x")) }
func texts(f *forms.Form) func() (any, bool)  { return get(f.Texts("x")) }
func ints(f *forms.Form) func() (any, bool)   { return get(f.Int("x")) }
func floats(f *forms.Form) func() (any, bool) { return get(f.Float("x")) }
func bools(f *forms.Form) func() (any, bool)  { return get(f.Bool("x")) }
func boxes(f *forms.Form) func() (any, bool)  { return get(f.Checkbox("x")) }
func dates(f *forms.Form) func() (any, bool)  { return get(f.Date("x")) }
func enums(f *forms.Form) func() (any, bool) {
	return get(f.Enum("x", []color{"red", "blue"}))
}

func equal(a, b any) bool {
	if as, ok := a.([]string); ok {
		bs, _ := b.([]string)
		return strings.Join(as, "\x00") == strings.Join(bs, "\x00") && (as == nil) == (bs == nil)
	}
	if at, ok := a.(time.Time); ok {
		return at.Equal(b.(time.Time))
	}
	return a == b
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		q    string
		want map[string]error // path → error; absent paths must be fine
	}{
		{"all good", "name=Ada&age=30&color=red&agree=on", nil},
		{"required missing", "age=30&color=red&agree=on", map[string]error{"name": forms.ErrRequired}},
		{"rule fails", "name=A&color=red&agree=on", map[string]error{"name": rule.ErrMinLen}},
		{"conversion fails", "name=Ada&age=abc&color=red&agree=on", map[string]error{"age": forms.ErrInvalid}},
		{"rule on a number", "name=Ada&age=-1&color=red&agree=on", map[string]error{"age": rule.ErrMin}},
		{"not an option", "name=Ada&color=green&agree=on", map[string]error{"color": forms.ErrNotAnOption}},
		{"checkbox not ticked", "name=Ada&color=red", map[string]error{"agree": forms.ErrRequired}},
		{"several", "age=x&agree=on", map[string]error{"name": forms.ErrRequired, "age": forms.ErrInvalid, "color": forms.ErrRequired}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := forms.New()
			f.Text("name").Required().Is(rule.MinLen(2))
			f.Int("age").Is(rule.Min(0))
			f.Enum("color", []color{"red", "blue"}).Required()
			f.Checkbox("agree").Required()
			bind(f, tc.q)
			ok := f.Validate(bg)
			if ok != (len(tc.want) == 0) || f.Valid() != ok {
				t.Errorf("Validate = %v, Valid = %v, want %v", ok, f.Valid(), len(tc.want) == 0)
			}
			got := map[string]error{}
			for fe := range f.Errors() {
				got[fe.Path] = fe.Err
			}
			if len(got) != len(tc.want) {
				t.Errorf("errors %v, want %v", got, tc.want)
			}
			for path, want := range tc.want {
				if !errors.Is(got[path], want) {
					t.Errorf("%s: %v, want %v", path, got[path], want)
				}
			}
			if errors.KindOf(f.Err()) != errors.Invalid && len(tc.want) > 0 {
				t.Errorf("kind %v, want invalid", errors.KindOf(f.Err()))
			}
		})
	}
}

func TestErrOrderAndText(t *testing.T) {
	f := forms.New()
	f.Text("a").Required()
	f.Text("b").Required()
	bind(f, "")
	f.Validate(bg)
	f.Fail(errTaken)
	var paths []string
	for fe := range f.Errors() {
		paths = append(paths, fe.Path)
	}
	if strings.Join(paths, ",") != ",a,b" {
		t.Errorf("order %q, want the form first, then a, b", paths)
	}
	if got := f.Err().Error(); got != "is taken\na: is required\nb: is required" {
		t.Errorf("Err() = %q", got)
	}
	var texts []string
	for p := range errors.All(f.Err()) {
		texts = append(texts, p.T(bg))
	}
	if strings.Join(texts, "|") != "is taken|is required|is required" {
		t.Errorf("All: %q", texts)
	}
}

func TestRedisplayAndValue(t *testing.T) {
	f := forms.New()
	n := f.Int("n")
	bind(f, "n=abc")
	f.Validate(bg)
	if !errors.Is(n.Err(), forms.ErrInvalid) || n.Value() != 0 || n.Ptr() != nil {
		t.Errorf("err %v, value %d", n.Err(), n.Value())
	}
}

func TestCheckRunsLast(t *testing.T) {
	ran := 0
	f := forms.New()
	pw, confirm := f.Text("pw").Raw(), f.Text("confirm").Raw()
	f.Check(func(context.Context) error {
		ran++
		if pw.Value() != confirm.Value() {
			return errTaken
		}
		return nil
	})
	bind(f, "pw=a&confirm=b")
	if f.Validate(bg) || ran != 1 || !errors.Is(f.Err(), errTaken) {
		t.Errorf("valid %v, ran %d, err %v", f.Valid(), ran, f.Err())
	}
	g := forms.New()
	g.Text("x").Required()
	g.Check(func(context.Context) error { ran++; return nil })
	bind(g, "")
	g.Validate(bg)
	if ran != 1 {
		t.Error("a check ran although a field failed")
	}
}

func TestSetForAnEditPage(t *testing.T) {
	f := forms.New()
	c := f.Enum("color", []color{"red", "blue"})
	d := f.Text("desc")
	c.Set("blue")
	var none *string
	d.SetPtr(none)
	if c.Value() != "blue" || d.Ptr() != nil {
		t.Errorf("color %q, desc %v", c.Value(), d.Ptr())
	}
	bind(f, "desc=x") // the submission wins: color is now absent
	f.Validate(bg)
	if _, ok := c.Get(); ok {
		t.Error("an initial value survived Bind")
	}
}

func TestFieldFail(t *testing.T) {
	f := forms.New()
	n := f.Text("name")
	bind(f, "name=Ada")
	f.Validate(bg)
	n.Fail(errTaken)
	if f.Valid() || !errors.Is(n.Err(), errTaken) || n.Value() != "" {
		t.Errorf("valid %v err %v value %q", f.Valid(), n.Err(), n.Value())
	}
}

func TestDateRulesCompareDates(t *testing.T) {
	f := forms.New()
	d := f.Date("d").Is(rule.NotBefore(time.Now()))
	bind(f, "d="+time.Now().UTC().Format(time.DateOnly))
	if !f.Validate(bg) {
		t.Errorf("today failed NotBefore(now): %v", d.Err())
	}
}

func TestDeclarationPanics(t *testing.T) {
	bound := forms.New()
	h := bound.Text("x")
	bound.Bind(forms.FromValues(nil))
	for name, fn := range map[string]func(){
		"bad name":             func() { forms.New().Text("Name") },
		"dotted name":          func() { forms.New().Text("a.b") },
		"duplicate":            func() { f := forms.New(); f.Text("a"); f.Int("a") },
		"Raw on Int":           func() { forms.New().Int("a").Raw() },
		"builder after Bind":   func() { h.Required() },
		"field after Bind":     func() { bound.Text("y") },
		"Set after Bind":       func() { h.Set("v") },
		"second Bind":          func() { bound.Bind(forms.FromValues(nil)) },
		"Validate before Bind": func() { forms.New().Validate(bg) },
		"RequiredWith private": func() { forms.New().Text("a").RequiredWith(errors.New("plain")) },
		"InvalidWith Def1":     func() { forms.New().Text("a").InvalidWith(forms.ErrMinItems) },
		"Label nil":            func() { forms.New().Text("a").Label(nil) },
		"Enum empty":           func() { forms.New().Enum("a", []color{}) },
		"Enum duplicate":       func() { forms.New().Enum("a", []color{"red", "red"}) },
		"Check nil":            func() { forms.New().Check(nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

func TestSubmit(t *testing.T) {
	newForm := func() (*forms.Form, *forms.Field[string], *forms.Field[*multipart.FileHeader]) {
		f := forms.New()
		return f, f.Text("name").Required(), f.File("logo")
	}
	t.Run("urlencoded ignores the query", func(t *testing.T) {
		f, name, _ := newForm()
		r := httptest.NewRequest(http.MethodPost, "/?name=fromquery", strings.NewReader("name=Ada"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if !f.Submit(bg, r) || name.Value() != "Ada" {
			t.Errorf("valid %v, name %q", f.Valid(), name.Value())
		}
		f2, _, _ := newForm()
		r2 := httptest.NewRequest(http.MethodPost, "/?name=fromquery", strings.NewReader(""))
		r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if f2.Submit(bg, r2) {
			t.Error("a POST took its value from the query")
		}
	})
	t.Run("GET reads the query", func(t *testing.T) {
		f, name, _ := newForm()
		if !f.Submit(bg, httptest.NewRequest(http.MethodGet, "/?name=Ada", nil)) || name.Value() != "Ada" {
			t.Errorf("valid %v, name %q", f.Valid(), name.Value())
		}
	})
	t.Run("multipart with a file and an empty part", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		w.WriteField("name", "Ada")
		fw, _ := w.CreateFormFile("logo", "logo.png")
		fw.Write([]byte("\x89PNG\r\n\x1a\n"))
		w.CreateFormFile("logo", "") // an empty file input
		w.Close()
		r := httptest.NewRequest(http.MethodPost, "/", &body)
		r.Header.Set("Content-Type", w.FormDataContentType())
		f, _, logo := newForm()
		if !f.Submit(bg, r) || logo.Value() == nil || logo.Value().Filename != "logo.png" {
			t.Errorf("valid %v, logo %v", f.Valid(), logo.Value())
		}
	})
	t.Run("unreadable", func(t *testing.T) {
		f, _, _ := newForm()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		if f.Submit(bg, r) || !errors.Is(f.Err(), forms.ErrUnreadable) {
			t.Errorf("valid %v, err %v", f.Valid(), f.Err())
		}
	})
}
