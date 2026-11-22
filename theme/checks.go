package theme

import (
	"strings"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/internal/themehook"
)

// The errors an admin sees when cfgx refuses a value (visual-interface.md
// §3.2, §11).
var (
	ErrColor  = errors.Key("errors.theme.color", "must be a colour such as #2f5bd3", errors.Invalid)
	ErrFont   = errors.Key("errors.theme.font", "must be a list of font families", errors.Invalid)
	ErrRadius = errors.Key("errors.theme.radius", "must be 0 or a length up to 32px or 2rem", errors.Invalid)
)

// CheckColor accepts #rgb or #rrggbb. It has the shape cfgx's Validate
// takes.
func CheckColor(s string) error {
	if !isColor(s) {
		return ErrColor
	}
	return nil
}

// CheckFont accepts 1 to 12 comma-separated families, each a quoted name
// of ASCII letters, digits, spaces, '.', '_' and '-', or a bare word such
// as system-ui, other than a CSS-wide keyword.
func CheckFont(s string) error {
	if !isFont(s) {
		return ErrFont
	}
	return nil
}

// CheckRadius accepts 0, or an unsigned length in px (at most 32), rem or
// em (at most 2).
func CheckRadius(s string) error {
	if !isRadius(s) {
		return ErrRadius
	}
	return nil
}

// cssWide are the keywords a bare family name can't be.
var cssWide = map[string]bool{"inherit": true, "initial": true, "unset": true, "revert": true, "revert-layer": true, "default": true}

func init() {
	themehook.Stylesheet = func() []byte {
		b, err := newBundle()
		if err != nil {
			panic(err)
		}
		return b.Files["ui.css"]
	}
	themehook.Tokens = func(values map[string]string) [][2]string {
		return overrides(valuesConfig(values))
	}
}

// valuesConfig reads themeschema's key names, falling back to the defaults.
type valuesConfig map[string]string

func (v valuesConfig) get(key, def string) string {
	if s := strings.TrimSpace(v[key]); s != "" {
		return s
	}
	return def
}

func (v valuesConfig) ThemeBrandColor() string {
	return v.get("brand_color", Defaults{}.ThemeBrandColor())
}
func (v valuesConfig) ThemeDangerColor() string {
	return v.get("danger_color", Defaults{}.ThemeDangerColor())
}
func (v valuesConfig) ThemeWarningColor() string {
	return v.get("warning_color", Defaults{}.ThemeWarningColor())
}
func (v valuesConfig) ThemeSuccessColor() string {
	return v.get("success_color", Defaults{}.ThemeSuccessColor())
}
func (v valuesConfig) ThemeInfoColor() string {
	return v.get("info_color", Defaults{}.ThemeInfoColor())
}
func (v valuesConfig) ThemeFontSans() string { return v.get("font_sans", Defaults{}.ThemeFontSans()) }
func (v valuesConfig) ThemeFontMono() string { return v.get("font_mono", Defaults{}.ThemeFontMono()) }
func (v valuesConfig) ThemeRadius() string   { return v.get("radius", Defaults{}.ThemeRadius()) }
