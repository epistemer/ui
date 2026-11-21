package forms_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/models/option"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

var errCustom = errors.Key("errors.test.custom", "is custom", errors.Invalid)
var errLen = errors.Key1[int]("errors.test.len", "needs %d", errors.Invalid)

func TestEmptyOptionsRejectEverything(t *testing.T) {
	var none []option.SelectOption
	f := forms.New()
	role := f.Text("role").Options(none...)
	kind := f.Enum("kind", []color{"red", "blue"}).Options(none...)
	optional := f.Text("note").Options(none...)
	bind(f, "role=admin&kind=red")
	f.Validate(bg)
	if !errors.Is(role.Err(), forms.ErrNotAnOption) || !errors.Is(kind.Err(), forms.ErrNotAnOption) {
		t.Errorf("role %v, kind %v: empty options must reject every value", role.Err(), kind.Err())
	}
	if optional.Err() != nil {
		t.Errorf("an absent optional field with empty options: %v", optional.Err())
	}
}

func TestEnumNeverYieldsAnUncheckedValue(t *testing.T) {
	f := forms.New()
	var seen color
	var seenOK bool
	f.Text("a").Is(rule.New(func(context.Context, string) error { return nil }))
	k := f.Enum("k", []color{"red", "blue"})
	f.Text("b").Is(rule.New(func(context.Context, string) error { seen, seenOK = k.Get(); return nil }))
	bind(f, "a=x&k=garbage&b=y")
	if _, ok := k.Get(); ok {
		t.Error("an enum outside its values is readable after Bind")
	}
	f.Validate(bg)
	if seenOK || seen != "" {
		t.Errorf("a rule read %q from the enum", seen)
	}
	if !errors.Is(k.Err(), forms.ErrNotAnOption) {
		t.Errorf("k: %v", k.Err())
	}
	g := forms.New()
	narrowed := g.Enum("k", []color{"red", "blue"}).Options(color("blue"))
	bind(g, "k=red")
	g.Validate(bg)
	if !errors.Is(narrowed.Err(), forms.ErrNotAnOption) {
		t.Errorf("Options must replace an enum's choices: %v", narrowed.Err())
	}
}

func TestConversionErrorAtBind(t *testing.T) {
	f := forms.New()
	n := f.Int("n")
	bind(f, "n=abc")
	if !errors.Is(n.Err(), forms.ErrInvalid) || f.Err() == nil {
		t.Errorf("after Bind: field %v, form %v", n.Err(), f.Err())
	}
}

func TestInvalidAndRequiredWith(t *testing.T) {
	f := forms.New()
	n := f.Int("n").InvalidWith(errCustom)
	m := f.Int("m").InvalidWith(errLen.With(3))
	r := f.Text("r").RequiredWith(errCustom)
	o := f.Int("o").InvalidWith(errCustom).Options(color("1"))
	bind(f, "n=x&m=y&o=x")
	f.Validate(bg)
	for name, tc := range map[string]struct{ got, want error }{
		"InvalidWith":             {n.Err(), errCustom},
		"InvalidWith occurrence":  {m.Err(), errLen},
		"RequiredWith":            {r.Err(), errCustom},
		"options beat conversion": {o.Err(), forms.ErrNotAnOption},
	} {
		if !errors.Is(tc.got, tc.want) {
			t.Errorf("%s: %v, want %v", name, tc.got, tc.want)
		}
	}
	if errors.Is(n.Err(), forms.ErrInvalid) {
		t.Error("InvalidWith must replace ErrInvalid")
	}
}

func TestMoreConstructors(t *testing.T) {
	f := forms.New()
	tm := f.Time("t", "2006-01-02T15:04")
	bad := f.Time("bad", "2006-01-02T15:04")
	fl := f.Float("f")
	nan := f.Float("nan")
	inf := f.Float("inf")
	b := f.Bool("b")
	badBool := f.Bool("bb")
	box := f.Checkbox("box").Required()
	bind(f, "t=2026-11-21T08:30&bad=yesterday&f=2.5&nan=NaN&inf=-Inf&b=0&bb=maybe&box=false")
	f.Validate(bg)
	if want := time.Date(2026, 11, 21, 8, 30, 0, 0, time.UTC); !tm.Value().Equal(want) || tm.Value().Location() != time.UTC {
		t.Errorf("Time %v", tm.Value())
	}
	for name, h := range map[string]error{"bad time": bad.Err(), "NaN": nan.Err(), "-Inf": inf.Err(), "bad bool": badBool.Err()} {
		if !errors.Is(h, forms.ErrInvalid) {
			t.Errorf("%s: %v, want ErrInvalid", name, h)
		}
	}
	if fl.Value() != 2.5 || b.Value() != false || b.Err() != nil {
		t.Errorf("float %v, bool %v %v", fl.Value(), b.Value(), b.Err())
	}
	if !errors.Is(box.Err(), forms.ErrRequired) {
		t.Errorf("a required checkbox sent only its hidden false: %v", box.Err())
	}
}

func TestFiles(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for _, name := range []string{"a.txt", "", "b.txt"} {
		fw, _ := w.CreateFormFile("docs", name)
		fw.Write([]byte("x"))
	}
	for _, name := range []string{"first.png", "last.png"} {
		fw, _ := w.CreateFormFile("logo", name)
		fw.Write([]byte("x"))
	}
	w.CreateFormFile("empty", "")
	w.Close()
	r := httptest.NewRequest(http.MethodPost, "/", &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	f := forms.New()
	docs, logo, empty := f.Files("docs"), f.File("logo"), f.File("empty")
	if !f.Submit(bg, r) {
		t.Fatal(f.Err())
	}
	if len(docs.Value()) != 2 || docs.Value()[1].Filename != "b.txt" {
		t.Errorf("docs %v", docs.Value())
	}
	if logo.Value().Filename != "last.png" {
		t.Errorf("logo %q: the last file wins", logo.Value().Filename)
	}
	if _, ok := empty.Get(); ok {
		t.Error("a part with no file name is a file")
	}
}

func TestFromRequestMethods(t *testing.T) {
	for _, tc := range []struct {
		method, target, ct, body string
		ok                       bool
	}{
		{http.MethodPut, "/", "application/x-www-form-urlencoded", "name=Ada", true},
		{http.MethodPatch, "/", "application/x-www-form-urlencoded", "name=Ada", true},
		{http.MethodHead, "/?name=Ada", "", "", true},
		{http.MethodGet, "/?name=%ZZ", "", "", false},
		{http.MethodDelete, "/?name=Ada", "", "", false},
		{http.MethodPost, "/", "", "name=Ada", false},
		{http.MethodPost, "/", "text/plain", "name=Ada", false},
		{http.MethodPost, "/", "multipart/form-data; boundary=x", "garbage", false},
	} {
		r := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
		if tc.ct != "" {
			r.Header.Set("Content-Type", tc.ct)
		}
		src, err := forms.FromRequest(r)
		if (err == nil) != tc.ok {
			t.Errorf("%s %s %q: err %v", tc.method, tc.target, tc.ct, err)
			continue
		}
		if tc.ok && strings.Join(src.Values("name"), "") != "Ada" {
			t.Errorf("%s: name %q", tc.method, src.Values("name"))
		}
	}
}

func TestNames(t *testing.T) {
	src := forms.FromValues(map[string][]string{"a": {"1"}, "b": {"2"}})
	n := 0
	for range src.Names() {
		n++
		break
	}
	if n != 1 {
		t.Error("Names must stop when the loop breaks")
	}
}

func TestSubmitUnreadableClearsAndRefusesAgain(t *testing.T) {
	f := forms.New()
	name := f.Text("name")
	name.Set("initial")
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	if f.Submit(bg, r) {
		t.Fatal("valid")
	}
	if _, ok := name.Get(); ok {
		t.Error("an initial value survived an unreadable Submit")
	}
	defer func() {
		if recover() == nil {
			t.Error("a second Submit did not panic")
		}
	}()
	f.Submit(bg, r)
}

func TestFailReplaces(t *testing.T) {
	f := forms.New()
	bind(f, "")
	f.Validate(bg)
	f.Fail(errCustom)
	f.Fail(errTaken)
	n := 0
	for range errors.All(f.Err()) {
		n++
	}
	if n != 1 || !errors.Is(f.Err(), errTaken) {
		t.Errorf("%d errors, %v: the form holds one error", n, f.Err())
	}
}

func TestKindAndHint(t *testing.T) {
	if forms.KindMultiSelect.String() != "multiselect" || forms.Kind(0).String() != "" || forms.Kind(99).String() != "" {
		t.Error("Kind.String")
	}
}

func TestMorePanics(t *testing.T) {
	bound := forms.New()
	h := bound.Text("x")
	bound.Bind(forms.FromValues(nil))
	var nilPtr *string
	for name, fn := range map[string]func(){
		"SetPtr(nil) after Bind": func() { h.SetPtr(nilPtr) },
		"Bind(nil)":              func() { forms.New().Bind(nil) },
		"Placeholder nil":        func() { forms.New().Text("a").Placeholder(nil) },
		"Help nil":               func() { forms.New().Text("a").Help(nil) },
		"Form.Fail nil":          func() { forms.New().Fail(nil) },
		"Field.Fail nil":         func() { forms.New().Text("a").Fail(nil) },
		"Options nil item":       func() { forms.New().Text("a").Options(nil) },
		"Hint group":             func() { forms.New().Text("a").Hint(forms.KindGroup) },
		"Time no layout":         func() { forms.New().Time("a", "") },
		"Raw on Texts":           func() { forms.New().Texts("a").Raw() },
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
	defer func() {
		if msg, _ := recover().(string); !strings.Contains(msg, "field name_x") {
			t.Errorf("panic %q does not name the field", msg)
		}
	}()
	forms.New().Text("name_x").RequiredWith(errors.New("plain"))
}
