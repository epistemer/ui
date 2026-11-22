package settings

import (
	"context"
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

// Each key runs theme's check: a bad value is refused when cfgx loads.
func TestChecks(t *testing.T) {
	for key, bad := range map[string]string{
		"SHOP_THEME_BRAND_COLOR": "red",
		"SHOP_THEME_FONT_SANS":   "x; } body {",
		"SHOP_THEME_RADIUS":      "3rem",
	} {
		if _, err := load(t, map[string]string{key: bad}); err == nil {
			t.Errorf("%s=%q loaded", key, bad)
		} else if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(strings.TrimPrefix(key, "SHOP_THEME_"))) &&
			!strings.Contains(err.Error(), "theme") {
			t.Errorf("%s: the problem doesn't name the key: %v", key, err)
		}
	}
	s, err := load(t, map[string]string{"SHOP_THEME_BRAND_COLOR": "#b4f601", "SHOP_THEME_RADIUS": "4px"})
	if err != nil {
		t.Fatal(err)
	}
	if s.ThemeBrandColor() != "#b4f601" || s.ThemeRadius() != "4px" {
		t.Errorf("got %q, %q", s.ThemeBrandColor(), s.ThemeRadius())
	}
}

func TestGenerated(t *testing.T) { cfgxtest.AssertGenerated(t) }
