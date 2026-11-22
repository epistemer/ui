// The token table (visual-interface.md §3.1): token names are public API,
// their values aren't.

package theme

// tokenKind is what a token's value is.
type tokenKind int

const (
	kindColor tokenKind = iota + 1
	kindLength
	kindFont
	kindShadow
	kindDuration
	kindEasing
)

// token is one design token, written as --ui-<Name>.
type token struct {
	Name        string
	kind        tokenKind
	light, dark string // dark empty: the same in both schemes; only colours differ
	Mutable     bool   // an admin may change it at run time (Q4)
}

// accent is a colour role with six tokens: the fill, text on the fill, the
// fill's hover and active shades, the role as text on a surface, and a
// subtle background. Only the fill is set; derive computes the rest.
type accent struct {
	Name string // brand, danger, warning, success, info
	Fill string // #rrggbb, the same in both schemes
}

// accents are the five accent roles and their default fills. All five can
// change at run time (Q4).
var accents = []accent{
	{"brand", "#2f5bd3"},
	{"danger", "#c62828"},
	{"warning", "#f2b01e"},
	{"success", "#1e8a4c"},
	{"info", "#0f7c9e"},
}

// surfaces are the neutral colours accents are measured against.
var surfaces = []token{
	{Name: "color-surface", kind: kindColor, light: "#f6f7f9", dark: "#0f1115"},
	{Name: "color-surface-raised", kind: kindColor, light: "#ffffff", dark: "#1a1d23"},
	{Name: "color-text", kind: kindColor, light: "#16181d", dark: "#e8eaee"},
	{Name: "color-text-muted", kind: kindColor, light: "#585e6b", dark: "#a3a9b5"},
	{Name: "color-border", kind: kindColor, light: "#d5d9e0", dark: "#2f343e"},
	// The edge of a control people type into or tick: WCAG 1.4.11 wants
	// 3:1 against what's around it, which a decorative border doesn't reach.
	{Name: "color-border-strong", kind: kindColor, light: "#7d8390", dark: "#6f7582"},
}

// scales are every other token.
var scales = []token{
	{Name: "space-1", kind: kindLength, light: "0.25rem"},
	{Name: "space-2", kind: kindLength, light: "0.5rem"},
	{Name: "space-3", kind: kindLength, light: "0.75rem"},
	{Name: "space-4", kind: kindLength, light: "1rem"},
	{Name: "space-5", kind: kindLength, light: "1.5rem"},
	{Name: "space-6", kind: kindLength, light: "2rem"},
	{Name: "radius", kind: kindLength, light: "0.5rem", Mutable: true}, // the scale's base: s, m, l derive from it
	{Name: "border-width", kind: kindLength, light: "1px"},
	{Name: "focus-width", kind: kindLength, light: "2px"},
	{Name: "focus-offset", kind: kindLength, light: "2px"},
	{Name: "text-xs", kind: kindLength, light: "0.75rem"},
	{Name: "text-s", kind: kindLength, light: "0.875rem"},
	{Name: "text-m", kind: kindLength, light: "1rem"},
	{Name: "text-l", kind: kindLength, light: "1.125rem"},
	{Name: "text-xl", kind: kindLength, light: "1.375rem"},
	{Name: "font-sans", kind: kindFont, light: `system-ui, "Segoe UI", "Roboto", "Noto Sans", "Noto Sans Arabic", sans-serif`, Mutable: true},
	{Name: "font-mono", kind: kindFont, light: `ui-monospace, "SFMono-Regular", "Menlo", "Consolas", monospace`, Mutable: true},
	{Name: "shadow-1", kind: kindShadow, light: "0 1px 2px rgb(0 0 0 / 0.08)"},
	{Name: "shadow-2", kind: kindShadow, light: "0 4px 12px rgb(0 0 0 / 0.12)"},
	{Name: "shadow-3", kind: kindShadow, light: "0 12px 32px rgb(0 0 0 / 0.18)"},
	{Name: "duration-s", kind: kindDuration, light: "120ms"},
	{Name: "duration-m", kind: kindDuration, light: "200ms"},
	{Name: "ease", kind: kindEasing, light: "cubic-bezier(0.2, 0, 0, 1)"},
}

// hoverShare and activeShare are the fill's share of its hover and active
// shades, mixed in OKLab away from the on-colour: towards black under white
// text, towards white under black text. Moving away from the on-colour only
// raises its contrast, so it holds in every state (visual-design.md finding 1: mixing
// towards a fixed shade per scheme dropped it to 3.56:1).
const (
	hoverShare  = 0.88
	activeShare = 0.76
)
