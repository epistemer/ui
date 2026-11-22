package theme

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Config is what an admin may change at run time (Q4): the five accent
// fills, the font families and the radius scale's base. cfgx's generated
// settings satisfy it through themeschema's group.
type Config interface {
	ThemeBrandColor() string
	ThemeDangerColor() string
	ThemeWarningColor() string
	ThemeSuccessColor() string
	ThemeInfoColor() string
	ThemeFontSans() string
	ThemeFontMono() string
	ThemeRadius() string
}

// Defaults is the token table's own values.
type Defaults struct{}

func (Defaults) ThemeBrandColor() string   { return accentDefault("brand") }
func (Defaults) ThemeDangerColor() string  { return accentDefault("danger") }
func (Defaults) ThemeWarningColor() string { return accentDefault("warning") }
func (Defaults) ThemeSuccessColor() string { return accentDefault("success") }
func (Defaults) ThemeInfoColor() string    { return accentDefault("info") }
func (Defaults) ThemeFontSans() string     { return scaleDefault("font-sans") }
func (Defaults) ThemeFontMono() string     { return scaleDefault("font-mono") }
func (Defaults) ThemeRadius() string       { return scaleDefault("radius") }

func accentDefault(name string) string {
	for _, a := range accents {
		if a.Name == name {
			return a.Fill
		}
	}
	panic("theme: no accent " + name)
}

func scaleDefault(name string) string {
	for _, t := range scales {
		if t.Name == name {
			return t.light
		}
	}
	panic("theme: no token " + name)
}

func fillOf(c Config, accent string) string {
	switch accent {
	case "brand":
		return c.ThemeBrandColor()
	case "danger":
		return c.ThemeDangerColor()
	case "warning":
		return c.ThemeWarningColor()
	case "success":
		return c.ThemeSuccessColor()
	case "info":
		return c.ThemeInfoColor()
	}
	panic("theme: no accent " + accent)
}

// Validation by kind (design §4). A stored value reaches a stylesheet the
// CSP trusts, so anything that isn't plainly a colour, a length or a list
// of font families falls back to the default.

var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// isColor reports whether s is a colour the theme accepts.
func isColor(s string) bool { return hexColor.MatchString(s) }

func validColor(s, def string) string {
	if isColor(s) {
		return s
	}
	return def
}

var length = regexp.MustCompile(`^(0|\d{1,2}(\.\d{1,3})?(px|rem|em))$`)

// isRadius reports whether s is a radius base the theme accepts: 0, or
// a length in px, rem or em of at most 32px or 2rem.
func isRadius(s string) bool {
	if !length.MatchString(s) {
		return false
	}
	if s == "0" {
		return true
	}
	unit := strings.TrimLeft(s, "0123456789.")
	n, _ := strconv.ParseFloat(strings.TrimSuffix(s, unit), 64)
	if unit == "px" {
		return n <= 32
	}
	return n <= 2
}

func validLength(s, def string) string {
	if isRadius(s) {
		return s
	}
	return def
}

var (
	quotedFamily = regexp.MustCompile(`^("[A-Za-z0-9 ._-]{1,64}"|'[A-Za-z0-9 ._-]{1,64}')$`)
	plainFamily  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,63}$`) // Inter, system-ui, sans-serif
)

// isFont reports whether s is a font-family list the theme accepts:
// quoted family names and single-word names, separated by commas.
func isFont(s string) bool {
	parts := strings.Split(s, ",")
	if len(parts) > 12 {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !quotedFamily.MatchString(p) && (!plainFamily.MatchString(p) || cssWide[strings.ToLower(p)]) {
			return false
		}
	}
	return true
}

func validFont(s, def string) string {
	if isFont(s) {
		return s
	}
	return def
}

// runtimeTheme serves the theme stylesheet built from a Config (design §4). It
// rebuilds the stylesheet when the values it reads differ from the last
// ones, so it needs no change callback from cfgx.
type runtimeTheme struct {
	cfg Config

	mu   sync.Mutex
	key  string
	css  []byte
	hash string
}

// newRuntime returns a runtimeTheme reading c.
func newRuntime(c Config) *runtimeTheme { return &runtimeTheme{cfg: c} }

// current returns the theme stylesheet for c's current values and its hash.
func (r *runtimeTheme) current() (css []byte, hash string) {
	// Read the eight values once: the stylesheet and its cache key come from
	// the same snapshot, even if the config changes meanwhile.
	snap := snapshot(r.cfg)
	key := strings.Join(snap[:], "\x00")
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != r.key || r.css == nil {
		var b strings.Builder
		b.WriteString("@layer ui.tokens {\n:root {\n")
		for _, kv := range overrides(snap) {
			b.WriteString("\t" + kv[0] + ": " + kv[1] + ";\n")
		}
		b.WriteString("}\n}\n")
		sum := sha256.Sum256([]byte(b.String()))
		r.key, r.css, r.hash = key, []byte(b.String()), hex.EncodeToString(sum[:])[:12]
	}
	return r.css, r.hash
}

// values is a Config's eight values, read once.
type values [8]string

func snapshot(c Config) values {
	return values{
		c.ThemeBrandColor(), c.ThemeDangerColor(), c.ThemeWarningColor(), c.ThemeSuccessColor(), c.ThemeInfoColor(),
		c.ThemeFontSans(), c.ThemeFontMono(), c.ThemeRadius(),
	}
}

func (v values) ThemeBrandColor() string   { return v[0] }
func (v values) ThemeDangerColor() string  { return v[1] }
func (v values) ThemeWarningColor() string { return v[2] }
func (v values) ThemeSuccessColor() string { return v[3] }
func (v values) ThemeInfoColor() string    { return v[4] }
func (v values) ThemeFontSans() string     { return v[5] }
func (v values) ThemeFontMono() string     { return v[6] }
func (v values) ThemeRadius() string       { return v[7] }
