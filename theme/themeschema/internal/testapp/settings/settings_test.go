package settings

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epistemer/cfgx"
	"github.com/epistemer/cfgx/cfgxtest"

	"github.com/epistemer/ui/theme"
)

// V1's done-when: a generated config satisfies theme.Config.
var _ theme.Config = (*Serve)(nil)

func load(t *testing.T, env map[string]string) (*Serve, error) {
	t.Helper()
	if env == nil {
		env = map[string]string{}
	}
	env["SHOP_STORAGE_STORE"] = filepath.Join(t.TempDir(), "shop.db")
	s, err := LoadServe(context.Background(), cfgx.WithEnv(env))
	if err == nil {
		t.Cleanup(func() { s.Close() })
	}
	return s, err
}

// The group keeps theme.Defaults: loaded with nothing set, every getter
// gives the token table's value.
func TestDefaults(t *testing.T) {
	s, err := load(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	d := theme.Defaults{}
	pairs := [][2]string{
		{s.ThemeBrandColor(), d.ThemeBrandColor()}, {s.ThemeDangerColor(), d.ThemeDangerColor()},
		{s.ThemeWarningColor(), d.ThemeWarningColor()}, {s.ThemeSuccessColor(), d.ThemeSuccessColor()},
		{s.ThemeInfoColor(), d.ThemeInfoColor()}, {s.ThemeFontSans(), d.ThemeFontSans()},
		{s.ThemeFontMono(), d.ThemeFontMono()}, {s.ThemeRadius(), d.ThemeRadius()},
	}
	for i, p := range pairs {
		if p[0] != p[1] {
			t.Errorf("getter %d: %q, want %q", i, p[0], p[1])
		}
	}
}

// Every key runs its check: a bad value is refused when cfgx loads, with
// theme's own error.
func TestChecks(t *testing.T) {
	cases := map[string]struct {
		bad  string
		want error
	}{
		"SHOP_THEME_BRAND_COLOR":   {"red", theme.ErrColor},
		"SHOP_THEME_DANGER_COLOR":  {"#abcd", theme.ErrColor},
		"SHOP_THEME_WARNING_COLOR": {"rgb(0 0 0)", theme.ErrColor},
		"SHOP_THEME_SUCCESS_COLOR": {"#12345", theme.ErrColor},
		"SHOP_THEME_INFO_COLOR":    {"blue", theme.ErrColor},
		"SHOP_THEME_FONT_SANS":     {"x; } body {", theme.ErrFont},
		"SHOP_THEME_FONT_MONO":     {"inherit", theme.ErrFont},
		"SHOP_THEME_RADIUS":        {"3rem", theme.ErrRadius},
	}
	for key, c := range cases {
		_, err := load(t, map[string]string{key: c.bad})
		if err == nil {
			t.Errorf("%s=%q loaded", key, c.bad)
			continue
		}
		if !errors.Is(err, c.want) && !strings.Contains(err.Error(), c.want.Error()) {
			t.Errorf("%s=%q: %v, want %v", key, c.bad, err, c.want)
		}
	}
}

// Every key is mutable: an admin's update takes effect, and a bad one is
// refused with the check's error.
func TestMutable(t *testing.T) {
	s, err := load(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = s.Update(ctx, func(u *Update) error {
		u.SetThemeBrandColor("#b4f601")
		u.SetThemeDangerColor("#aa0000")
		u.SetThemeWarningColor("#ffaa00")
		u.SetThemeSuccessColor("#00aa44")
		u.SetThemeInfoColor("#0088aa")
		u.SetThemeFontSans("Inter, sans-serif")
		u.SetThemeFontMono("monospace")
		u.SetThemeRadius("4px")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.ThemeBrandColor() != "#b4f601" || s.ThemeFontSans() != "Inter, sans-serif" || s.ThemeRadius() != "4px" {
		t.Errorf("after update: %q %q %q", s.ThemeBrandColor(), s.ThemeFontSans(), s.ThemeRadius())
	}
	if err := s.Update(ctx, func(u *Update) error { u.SetThemeRadius("9rem"); return nil }); err == nil {
		t.Error("a bad radius was stored")
	}
}

func TestGenerated(t *testing.T) { cfgxtest.AssertGenerated(t) }
