package forms_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	uierrors "github.com/epistemer/errors"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

var seconds = forms.Conv[int, time.Duration]{
	Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Second, nil },
	Format: func(d time.Duration) int { return int(d / time.Second) },
}

var errTooLong = uierrors.Key("errors.test.too_long", "is too long", uierrors.Invalid)

var hours = forms.Conv[time.Duration, int]{
	Parse: func(d time.Duration) (int, error) {
		if d%time.Hour != 0 {
			return 0, errors.New("not whole hours") // private: becomes ErrInvalid
		}
		return int(d / time.Hour), nil
	},
	Format: func(h int) time.Duration { return time.Duration(h) * time.Hour },
}

var declared = forms.Conv[int, int]{
	Parse:  func(n int) (int, error) { return 0, errTooLong },
	Format: func(n int) int { return n },
}

func TestAsPipeline(t *testing.T) {
	for _, tc := range []struct {
		q    string
		want error
		val  time.Duration
	}{
		{"c=30", nil, 30 * time.Second},
		{"c=-1", rule.ErrMin, 0},       // a rule before As
		{"c=7200", rule.ErrMax, 0},     // a rule after As
		{"c=abc", forms.ErrInvalid, 0}, // the base conversion
		{"", forms.ErrRequired, 0},     // required carries over
	} {
		f := forms.New()
		c := f.Int("c").Required().Is(rule.Min(0)).As(seconds).Is(rule.Max(time.Hour))
		bind(f, tc.q)
		f.Validate(bg)
		if tc.want == nil && c.Err() != nil || tc.want != nil && !uierrors.Is(c.Err(), tc.want) {
			t.Errorf("%q: %v, want %v", tc.q, c.Err(), tc.want)
		}
		if c.Value() != tc.val {
			t.Errorf("%q: value %v, want %v", tc.q, c.Value(), tc.val)
		}
	}
}

func TestStageRulesBeforeConversionError(t *testing.T) {
	// 90 minutes: Min passes on the int, the duration rule fails before
	// the hours conversion, which would also fail.
	f := forms.New()
	h := f.Int("m").Is(rule.Min(0)).
		As(forms.Conv[int, time.Duration]{
			Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Minute, nil },
			Format: func(d time.Duration) int { return int(d / time.Minute) },
		}).
		Is(rule.Max(time.Hour)).
		As(hours)
	bind(f, "m=90")
	f.Validate(bg)
	if !uierrors.Is(h.Err(), rule.ErrMax) {
		t.Errorf("got %v: an earlier stage's rule must win over a later conversion error", h.Err())
	}
	g := forms.New()
	h2 := g.Int("m").As(forms.Conv[int, time.Duration]{
		Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Minute, nil },
		Format: func(d time.Duration) int { return int(d / time.Minute) },
	}).As(hours)
	bind(g, "m=90")
	g.Validate(bg)
	if !uierrors.Is(h2.Err(), forms.ErrInvalid) || !strings.Contains(h2.Err().Error(), "not whole hours") {
		t.Errorf("got %v: a private Parse error is the cause of ErrInvalid", h2.Err())
	}
	k := forms.New()
	h3 := k.Int("n").As(declared)
	bind(k, "n=1")
	k.Validate(bg)
	if !uierrors.Is(h3.Err(), errTooLong) || uierrors.Is(h3.Err(), forms.ErrInvalid) {
		t.Errorf("got %v: a declared Parse error is shown as it is", h3.Err())
	}
}

func TestAsSetFormatsBack(t *testing.T) {
	f := forms.New()
	c := f.Int("c").As(seconds)
	c.Set(90 * time.Second)
	if c.Value() != 90*time.Second {
		t.Errorf("value %v", c.Value())
	}
	// The text a renderer shows is the base's: checked through Control in F5.
}

func TestDeadHandle(t *testing.T) {
	f := forms.New()
	raw := f.Int("c")
	_ = raw.As(seconds)
	for name, fn := range map[string]func(){
		"Value": func() { raw.Value() },
		"Is":    func() { raw.Is(rule.Min(0)) },
		"As":    func() { raw.As(seconds) },
		"nil Parse": func() {
			forms.New().Int("x").As(forms.Conv[int, int]{Format: func(n int) int { return n }})
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if msg, _ := recover().(string); msg == "" {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

type depth int16

func TestNumber(t *testing.T) {
	f := forms.New()
	a := f.Number[int32]("a").Is(rule.Min[int32](1))
	b := f.Number[uint8]("b")
	c := f.Number[depth]("c")
	d := f.Number[float32]("d")
	e := f.Number[int64]("e")
	bind(f, "a=0&b=300&c=-12&d=1.5&e=9223372036854775807")
	f.Validate(bg)
	if !uierrors.Is(a.Err(), rule.ErrMin) {
		t.Errorf("a: %v", a.Err())
	}
	if !uierrors.Is(b.Err(), forms.ErrInvalid) {
		t.Errorf("b: 300 must not fit uint8: %v", b.Err())
	}
	if c.Value() != -12 || d.Value() != 1.5 || e.Value() != 9223372036854775807 {
		t.Errorf("c %v d %v e %v", c.Value(), d.Value(), e.Value())
	}
}

type code struct{ s string }

func (c code) MarshalText() ([]byte, error) { return []byte(strings.ToUpper(c.s)), nil }
func (c *code) UnmarshalText(b []byte) error {
	if len(b) != 3 {
		return errors.New("a code has 3 letters")
	}
	c.s = strings.ToLower(string(b))
	return nil
}

func TestParsed(t *testing.T) {
	f := forms.New()
	ok := f.Parsed[code]("ok")
	bad := f.Parsed[code]("bad")
	bind(f, "ok=egp&bad=toolong")
	f.Validate(bg)
	if ok.Value().s != "egp" || !uierrors.Is(bad.Err(), forms.ErrInvalid) {
		t.Errorf("ok %v, bad %v", ok.Value(), bad.Err())
	}
}

type namer interface{ Name() string }

type person string

func (p person) Name() string { return string(p) }

func TestAsWithInterfaceTypes(t *testing.T) {
	toNamer := forms.Conv[string, namer]{
		Parse: func(s string) (namer, error) {
			if s == "nobody" {
				return nil, nil
			}
			return person(s), nil
		},
		Format: func(n namer) string {
			if n == nil {
				return ""
			}
			return n.Name()
		},
	}
	upper := forms.Conv[namer, string]{
		Parse:  func(n namer) (string, error) { return strings.ToUpper(fmt.Sprint(n)), nil },
		Format: func(s string) namer { return person(s) },
	}
	f := forms.New()
	n := f.Text("n").As(toNamer).Is(rule.New(func(_ context.Context, v namer) error { return nil }))
	g := forms.New()
	u := g.Text("u").As(toNamer).As(upper)
	bind(f, "n=nobody")
	bind(g, "u=nobody")
	f.Validate(bg)
	g.Validate(bg)
	if n.Err() != nil || n.Value() != nil || u.Value() != "<NIL>" {
		t.Errorf("n %v %v, u %q", n.Value(), n.Err(), u.Value())
	}
	h := forms.New().Text("h").As(toNamer)
	h.Set(nil)
	if h.Value() != nil {
		t.Errorf("Set(nil) on an interface field: %v", h.Value())
	}
}

func TestFormatThroughStages(t *testing.T) {
	var formatted []string
	minutes := forms.Conv[int, time.Duration]{
		Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Minute, nil },
		Format: func(d time.Duration) int { formatted = append(formatted, "minutes"); return int(d / time.Minute) },
	}
	wrap := forms.Conv[time.Duration, [1]time.Duration]{
		Parse:  func(d time.Duration) ([1]time.Duration, error) { return [1]time.Duration{d}, nil },
		Format: func(a [1]time.Duration) time.Duration { formatted = append(formatted, "wrap"); return a[0] },
	}
	f := forms.New()
	c := f.Int("c").As(minutes).As(wrap)
	c.Set([1]time.Duration{90 * time.Minute})
	if strings.Join(formatted, ",") != "wrap,minutes" {
		t.Errorf("Set formatted through %v; want the last stage first", formatted)
	}
}

func TestAsMisuse(t *testing.T) {
	bound := forms.New()
	b := bound.Int("b")
	bound.Bind(forms.FromValues(nil))
	for name, fn := range map[string]func(){
		"As after Set":  func() { h := forms.New().Int("x"); h.Set(1); h.As(seconds) },
		"As after Bind": func() { b.As(seconds) },
		"nil Format": func() {
			forms.New().Int("x").As(forms.Conv[int, int]{Parse: func(n int) (int, error) { return n, nil }})
		},
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

func TestNumberEdges(t *testing.T) {
	f := forms.New()
	cases := map[string]*forms.Field[float32]{}
	for _, n := range []string{"nan", "inf", "big"} {
		cases[n] = f.Number[float32](n)
	}
	neg := f.Number[uint16]("neg")
	over := f.Number[int8]("over")
	under := f.Number[int8]("under")
	bind(f, "nan=NaN&inf=%2BInf&big=1e39&neg=-1&over=128&under=-129")
	f.Validate(bg)
	for name, h := range cases {
		if !uierrors.Is(h.Err(), forms.ErrInvalid) {
			t.Errorf("%s: %v", name, h.Err())
		}
	}
	for name, err := range map[string]error{"neg": neg.Err(), "over": over.Err(), "under": under.Err()} {
		if !uierrors.Is(err, forms.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	s := forms.New().Number[depth]("d")
	s.Set(-3)
	if s.Value() != -3 {
		t.Errorf("Set on a named number type: %v", s.Value())
	}
}

var errBadCode = uierrors.Key("errors.test.bad_code", "is not a code", uierrors.Invalid)

type strictCode struct{ s string }

func (c strictCode) MarshalText() ([]byte, error) { return []byte(c.s), nil }
func (c *strictCode) UnmarshalText(b []byte) error {
	if len(b) != 3 {
		return errBadCode
	}
	c.s = string(b)
	return nil
}

func TestParsedMore(t *testing.T) {
	f := forms.New()
	spaced := f.Parsed[code]("spaced")
	blank := f.Parsed[code]("blank")
	strict := f.Parsed[strictCode]("strict")
	bind(f, "spaced=+egp+&blank=+++&strict=toolong")
	f.Validate(bg)
	if spaced.Value().s != "egp" {
		t.Errorf("spaced: %q, %v: Parsed values are trimmed", spaced.Value().s, spaced.Err())
	}
	if _, ok := blank.Get(); ok || blank.Err() != nil {
		t.Errorf("blank: white space alone is absent")
	}
	if !uierrors.Is(strict.Err(), errBadCode) || uierrors.Is(strict.Err(), forms.ErrInvalid) {
		t.Errorf("strict: %v: a declared UnmarshalText error is shown as it is", strict.Err())
	}
	g := forms.New().Parsed[code]("g")
	g.Set(code{"usd"})
	if g.Value().s != "usd" {
		t.Errorf("Set on Parsed: %v", g.Value())
	}
}
