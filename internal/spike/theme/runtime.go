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
	BrandColor() string
	DangerColor() string
	WarningColor() string
	SuccessColor() string
	InfoColor() string
	FontSans() string
	FontMono() string
	Radius() string
}

// Defaults is the token table's own values.
type Defaults struct{}

func (Defaults) BrandColor() string   { return accentDefault("brand") }
func (Defaults) DangerColor() string  { return accentDefault("danger") }
func (Defaults) WarningColor() string { return accentDefault("warning") }
func (Defaults) SuccessColor() string { return accentDefault("success") }
func (Defaults) InfoColor() string    { return accentDefault("info") }
func (Defaults) FontSans() string     { return scaleDefault("font-sans") }
func (Defaults) FontMono() string     { return scaleDefault("font-mono") }
func (Defaults) Radius() string       { return scaleDefault("radius") }

func accentDefault(name string) string {
	for _, a := range Accents {
		if a.Name == name {
			return a.Fill
		}
	}
	panic("theme: no accent " + name)
}

func scaleDefault(name string) string {
	for _, t := range Scales {
		if t.Name == name {
			return t.Light
		}
	}
	panic("theme: no token " + name)
}

func fillOf(c Config, accent string) string {
	switch accent {
	case "brand":
		return c.BrandColor()
	case "danger":
		return c.DangerColor()
	case "warning":
		return c.WarningColor()
	case "success":
		return c.SuccessColor()
	case "info":
		return c.InfoColor()
	}
	panic("theme: no accent " + accent)
}

// Validation by kind (design §4). A stored value reaches a stylesheet the
// CSP trusts, so anything that isn't plainly a colour, a length or a list
// of font families falls back to the default.

var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// ValidColor reports whether s is a colour the theme accepts.
func ValidColor(s string) bool { return hexColor.MatchString(s) }

func validColor(s, def string) string {
	if ValidColor(s) {
		return s
	}
	return def
}

var length = regexp.MustCompile(`^(0|\d{1,2}(\.\d{1,3})?(px|rem|em))$`)

// ValidLength reports whether s is a radius base the theme accepts: 0, or
// a length in px, rem or em of at most 32px or 2rem.
func ValidLength(s string) bool {
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
	if ValidLength(s) {
		return s
	}
	return def
}

var (
	quotedFamily = regexp.MustCompile(`^("[A-Za-z0-9 ._-]{1,64}"|'[A-Za-z0-9 ._-]{1,64}')$`)
	plainFamily  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,63}$`) // Inter, system-ui, sans-serif
)

// ValidFont reports whether s is a font-family list the theme accepts:
// quoted family names and single-word names, separated by commas.
func ValidFont(s string) bool {
	parts := strings.Split(s, ",")
	if len(parts) > 12 {
		return false
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !quotedFamily.MatchString(p) && !plainFamily.MatchString(p) {
			return false
		}
	}
	return true
}

func validFont(s, def string) string {
	if ValidFont(s) {
		return s
	}
	return def
}

// Runtime serves the theme stylesheet built from a Config (design §4). It
// rebuilds the stylesheet when the values it reads differ from the last
// ones, so it needs no change callback from cfgx.
type Runtime struct {
	cfg Config

	mu   sync.Mutex
	key  string
	css  []byte
	hash string
}

// NewRuntime returns a Runtime reading c.
func NewRuntime(c Config) *Runtime { return &Runtime{cfg: c} }

// Current returns the theme stylesheet for c's current values and its hash.
func (r *Runtime) Current() (css []byte, hash string) {
	c := r.cfg
	key := strings.Join([]string{
		c.BrandColor(), c.DangerColor(), c.WarningColor(), c.SuccessColor(), c.InfoColor(),
		c.FontSans(), c.FontMono(), c.Radius(),
	}, "\x00")
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != r.key || r.css == nil {
		var b strings.Builder
		b.WriteString("@layer ui.tokens {\n:root {\n")
		for _, kv := range Overrides(c) {
			b.WriteString("\t" + kv[0] + ": " + kv[1] + ";\n")
		}
		b.WriteString("}\n}\n")
		sum := sha256.Sum256([]byte(b.String()))
		r.key, r.css, r.hash = key, []byte(b.String()), hex.EncodeToString(sum[:])[:12]
	}
	return r.css, r.hash
}
