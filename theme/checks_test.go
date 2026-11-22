package theme

import (
	"testing"

	"github.com/epistemer/errors"
)

var _ Config = Defaults{}

func TestChecks(t *testing.T) {
	cases := []struct {
		check func(string) error
		err   error
		good  []string
		bad   []string
	}{
		{CheckColor, ErrColor, []string{"#abc", "#2F5BD3"}, []string{"", "red", "#abcd", "rgb(0 0 0)", "#fff;"}},
		{CheckFont, ErrFont, []string{"Inter, sans-serif", `"Noto Sans Arabic", 'Segoe UI', system-ui`},
			[]string{"", "inherit", "sans-serif, unset", "Comic Sans", `x; }`, "a,b,c,d,e,f,g,h,i,j,k,l,m", `url(x)`}},
		{CheckRadius, ErrRadius, []string{"0", "4px", "32px", "0.5rem", "2em"}, []string{"", "-1px", "33px", "3rem", "1", ".5rem", "1vw"}},
	}
	for _, c := range cases {
		for _, s := range c.good {
			if err := c.check(s); err != nil {
				t.Errorf("%q refused: %v", s, err)
			}
		}
		for _, s := range c.bad {
			if err := c.check(s); !errors.Is(err, c.err) {
				t.Errorf("%q: got %v, want %v", s, err, c.err)
			}
		}
	}
	// The defaults pass their own checks.
	d := Defaults{}
	for _, s := range []string{d.ThemeBrandColor(), d.ThemeDangerColor(), d.ThemeWarningColor(), d.ThemeSuccessColor(), d.ThemeInfoColor()} {
		if CheckColor(s) != nil {
			t.Errorf("default colour %q fails", s)
		}
	}
	if CheckFont(d.ThemeFontSans()) != nil || CheckFont(d.ThemeFontMono()) != nil || CheckRadius(d.ThemeRadius()) != nil {
		t.Error("a default font or radius fails its check")
	}
}
