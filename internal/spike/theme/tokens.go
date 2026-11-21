// Package theme is the D4 spike of ui/theme (visual-design.md §4): the
// token table, the stylesheet that joins ui's CSS layers, and the run-time
// theme whose accent partners are derived for contrast. Nothing here is
// public API; the spike's findings amend the design.
package theme

// Kind is what a token's value is.
type Kind int

const (
	KindColor Kind = iota + 1
	KindLength
	KindFont
	KindShadow
	KindDuration
	KindEasing
)

// Token is one design token, written as --ui-<Name>.
type Token struct {
	Name        string
	Kind        Kind
	Light, Dark string // Dark empty: the same in both schemes; only colours differ
	Mutable     bool   // an admin may change it at run time (Q4)
}

// Accent is a colour role with six tokens: the fill, text on the fill, the
// fill's hover and active shades, the role as text on a surface, and a
// subtle background. Only the fill is set; Derive computes the rest.
type Accent struct {
	Name string // brand, danger, warning, success, info
	Fill string // #rrggbb, the same in both schemes
}

// Accents are the five accent roles and their default fills. All five can
// change at run time (Q4).
var Accents = []Accent{
	{"brand", "#2f5bd3"},
	{"danger", "#c62828"},
	{"warning", "#f2b01e"},
	{"success", "#1e8a4c"},
	{"info", "#0f7c9e"},
}

// Surfaces are the neutral colours accents are measured against.
var Surfaces = []Token{
	{Name: "color-surface", Kind: KindColor, Light: "#f6f7f9", Dark: "#0f1115"},
	{Name: "color-surface-raised", Kind: KindColor, Light: "#ffffff", Dark: "#1a1d23"},
	{Name: "color-text", Kind: KindColor, Light: "#16181d", Dark: "#e8eaee"},
	{Name: "color-text-muted", Kind: KindColor, Light: "#585e6b", Dark: "#a3a9b5"},
	{Name: "color-border", Kind: KindColor, Light: "#d5d9e0", Dark: "#2f343e"},
}

// Scales are every other token.
var Scales = []Token{
	{Name: "space-1", Kind: KindLength, Light: "0.25rem"},
	{Name: "space-2", Kind: KindLength, Light: "0.5rem"},
	{Name: "space-3", Kind: KindLength, Light: "0.75rem"},
	{Name: "space-4", Kind: KindLength, Light: "1rem"},
	{Name: "space-5", Kind: KindLength, Light: "1.5rem"},
	{Name: "space-6", Kind: KindLength, Light: "2rem"},
	{Name: "radius", Kind: KindLength, Light: "0.5rem", Mutable: true}, // the scale's base: s, m, l derive from it
	{Name: "border-width", Kind: KindLength, Light: "1px"},
	{Name: "focus-width", Kind: KindLength, Light: "2px"},
	{Name: "focus-offset", Kind: KindLength, Light: "2px"},
	{Name: "text-xs", Kind: KindLength, Light: "0.75rem"},
	{Name: "text-s", Kind: KindLength, Light: "0.875rem"},
	{Name: "text-m", Kind: KindLength, Light: "1rem"},
	{Name: "text-l", Kind: KindLength, Light: "1.125rem"},
	{Name: "text-xl", Kind: KindLength, Light: "1.375rem"},
	{Name: "font-sans", Kind: KindFont, Light: `system-ui, "Segoe UI", "Roboto", "Noto Sans", "Noto Sans Arabic", sans-serif`, Mutable: true},
	{Name: "font-mono", Kind: KindFont, Light: `ui-monospace, "SFMono-Regular", "Menlo", "Consolas", monospace`, Mutable: true},
	{Name: "shadow-1", Kind: KindShadow, Light: "0 1px 2px rgb(0 0 0 / 0.08)"},
	{Name: "shadow-2", Kind: KindShadow, Light: "0 4px 12px rgb(0 0 0 / 0.12)"},
	{Name: "shadow-3", Kind: KindShadow, Light: "0 12px 32px rgb(0 0 0 / 0.18)"},
	{Name: "duration-s", Kind: KindDuration, Light: "120ms"},
	{Name: "duration-m", Kind: KindDuration, Light: "200ms"},
	{Name: "ease", Kind: KindEasing, Light: "cubic-bezier(0.2, 0, 0, 1)"},
}

// HoverShare and ActiveShare are the fill's share of its hover and active
// shades, mixed in OKLab away from the on-colour: towards black under white
// text, towards white under black text. Moving away from the on-colour only
// raises its contrast, so it holds in every state (spike finding: mixing
// towards a fixed shade per scheme dropped it to 3.56:1).
const (
	HoverShare  = 0.88
	ActiveShare = 0.76
)
