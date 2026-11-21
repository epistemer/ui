package forms_test

import (
	"errors"
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
