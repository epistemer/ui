package theme

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// The CSS checks of ADR 0004: outside the token layer, declarations read
// tokens, not literal colours, lengths or shadows; and nothing names a
// physical side. They rule out the two mistakes zp's templates show most;
// they don't prove a page looks right.

type decl struct {
	line      int
	selectors []string // enclosing preludes, outermost first
	prop      string
	value     string
	important bool
}

var propName = regexp.MustCompile(`^(--[a-zA-Z0-9_-]+|-?[a-zA-Z][a-zA-Z0-9-]*)$`)

// scanCSS lists the declarations of src, following nesting.
func scanCSS(src string) ([]decl, error) {
	var (
		out   []decl
		stack []string
		buf   strings.Builder
		line  = 1
		start = 1 // line where buf's statement began
		paren int
	)
	flush := func() {
		s := strings.TrimSpace(buf.String())
		buf.Reset()
		if s == "" || strings.HasPrefix(s, "@") {
			return
		}
		name, value, ok := strings.Cut(s, ":")
		name = strings.TrimSpace(name)
		if !ok || !propName.MatchString(name) {
			return
		}
		value = strings.TrimSpace(value)
		imp := false
		if v, ok := strings.CutSuffix(value, "!important"); ok {
			value, imp = strings.TrimSpace(v), true
		}
		out = append(out, decl{line: start, selectors: append([]string(nil), stack...), prop: strings.ToLower(name), value: value, important: imp})
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		if buf.Len() == 0 && strings.TrimSpace(string(c)) != "" {
			start = line
		}
		switch {
		case c == '\n':
			line++
			buf.WriteByte(c)
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("line %d: unclosed comment", line)
			}
			line += strings.Count(src[i:i+2+end+2], "\n")
			i += end + 3
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			buf.WriteString(src[i:min(j+1, len(src))])
			i = j
		case c == '(':
			paren++
			buf.WriteByte(c)
		case c == ')':
			paren--
			buf.WriteByte(c)
		case paren > 0:
			buf.WriteByte(c)
		case c == '{':
			stack = append(stack, strings.TrimSpace(buf.String()))
			buf.Reset()
		case c == ';':
			flush()
		case c == '}':
			flush()
			if len(stack) == 0 {
				return nil, fmt.Errorf("line %d: unbalanced }", line)
			}
			stack = stack[:len(stack)-1]
		default:
			buf.WriteByte(c)
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unclosed block %q", stack[len(stack)-1])
	}
	return out, nil
}

// topLevel splits v on white space outside parentheses.
func topLevel(v string) []string {
	var parts []string
	depth, from := 0, 0
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case (r == ' ' || r == '\t' || r == '\n') && depth == 0:
			if from < i {
				parts = append(parts, v[from:i])
			}
			from = i + 1
		}
	}
	if from < len(v) {
		parts = append(parts, v[from:])
	}
	return parts
}

var (
	physicalProps = regexp.MustCompile(`^(left|right|(margin|padding|scroll-margin|scroll-padding)-(left|right)|border-(left|right)(-.*)?|border-(top|bottom)-(left|right)-radius)$`)
	physicalValue = map[string]*regexp.Regexp{
		"float":      regexp.MustCompile(`^(left|right)$`),
		"clear":      regexp.MustCompile(`^(left|right)$`),
		"text-align": regexp.MustCompile(`^(left|right)$`),
		"resize":     regexp.MustCompile(`^(horizontal)$`), // fine either way; listed to show the shape
	}
	fourSided     = regexp.MustCompile(`^(margin|padding|inset|scroll-margin|scroll-padding|border-width|border-style|border-color)$`)
	gradientSide  = regexp.MustCompile(`\bto\s+((top|bottom)\s+)?(left|right)\b`)
	translateX    = regexp.MustCompile(`\btranslate(X|3d)?\(`)
	hexLiteral    = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`)
	colorFunc     = regexp.MustCompile(`\b(rgba?|hsla?|hwb|lab|lch|oklab|oklch|color)\(`)
	namedColor    = regexp.MustCompile(`(?i)\b(black|white|red|green|blue|gray|grey|yellow|orange|purple|pink|brown|navy|teal|silver|maroon|olive|lime|aqua|fuchsia)\b`)
	lengthLiteral = regexp.MustCompile(`(-?\d*\.?\d+)(px|rem|pt|pc|cm|mm|in|q)\b`)
	varCall       = regexp.MustCompile(`var\(--[a-z0-9-]+\)`)
)

// cssProblems reports what the checks reject in one file.
func cssProblems(file, src string) []string {
	decls, err := scanCSS(src)
	if err != nil {
		return []string{file + ": " + err.Error()}
	}
	var probs []string
	report := func(d decl, format string, args ...any) {
		probs = append(probs, fmt.Sprintf("%s:%d: %s: %s", file, d.line, d.prop, fmt.Sprintf(format, args...)))
	}
	for _, d := range decls {
		if strings.HasPrefix(d.prop, "--") {
			// A component may alias tokens privately (--_fill: var(--ui-…)),
			// so a tone is one block of aliases rather than a copy of every
			// variant; anything else belongs in the token layer.
			if !strings.HasPrefix(d.prop, "--_") || strings.TrimSpace(varCall.ReplaceAllString(d.value, "")) != "" {
				report(d, "custom properties belong in the token layer; a component may only alias tokens as --_name: var(--ui-…)")
			}
			continue
		}
		inDir := false
		for _, s := range d.selectors {
			if strings.Contains(s, ":dir(") {
				inDir = true
			}
		}
		if d.important && !(file == "reset.css" && d.prop == "display" && d.value == "none" && len(d.selectors) == 1 && d.selectors[0] == "[hidden]") {
			report(d, "!important is only for [hidden] in reset.css")
		}
		if physicalProps.MatchString(d.prop) {
			report(d, "a physical side; use the logical property")
		}
		if re, ok := physicalValue[d.prop]; ok && d.prop != "resize" && re.MatchString(d.value) {
			report(d, "a physical value %q; use start or end", d.value)
		}
		parts := topLevel(d.value)
		if fourSided.MatchString(d.prop) && len(parts) == 4 && parts[1] != parts[3] {
			report(d, "different left and right values; use the logical longhands")
		}
		if d.prop == "border-radius" && (len(parts) > 1 || strings.Contains(d.value, "/")) {
			report(d, "a multi-value radius names physical corners; use border-start-start-radius and friends")
		}
		if (translateX.MatchString(d.value) || d.prop == "translate") && !inDir {
			report(d, "a horizontal translation outside a :dir() rule")
		}
		if gradientSide.MatchString(d.value) {
			report(d, "a gradient towards a physical side")
		}
		v := varCall.ReplaceAllString(d.value, "")
		if hexLiteral.MatchString(v) || colorFunc.MatchString(v) || namedColor.MatchString(v) {
			report(d, "a literal colour; read a token")
		}
		for _, m := range lengthLiteral.FindAllStringSubmatch(v, -1) {
			if m[0] == "1px" {
				continue // hairlines
			}
			report(d, "a literal length %s; read a token", m[0])
		}
	}
	return probs
}

func TestCSSChecksOnUIStylesheets(t *testing.T) {
	files, err := fs.Glob(assetFS, "css/*.css")
	if err != nil || len(files) == 0 {
		t.Fatalf("no stylesheets: %v", err)
	}
	for _, f := range files {
		b, _ := assetFS.ReadFile(f)
		for _, p := range cssProblems(strings.TrimPrefix(f, "css/"), string(b)) {
			t.Error(p)
		}
	}
}

func TestCSSChecksCatchMistakes(t *testing.T) {
	bad := map[string]string{
		"physical property":     ".a { margin-left: var(--ui-space-1); }",
		"physical value":        ".a { text-align: left; }",
		"float":                 ".a { float: right; }",
		"asymmetric shorthand":  ".a { padding: 0 var(--ui-space-2) 0 var(--ui-space-4); }",
		"multi-value radius":    ".a { border-radius: var(--ui-radius-s) 0; }",
		"translateX":            ".a { transform: translateX(100%); }",
		"translate property":    ".a { translate: 100% 0; }",
		"gradient side":         ".a { background-image: linear-gradient(to right, var(--ui-color-brand), transparent); }",
		"hex":                   ".a { color: #fff; }",
		"rgb":                   ".a { color: rgb(0 0 0); }",
		"named":                 ".a { border-color: white; }",
		"rem":                   ".a { gap: 0.5rem; }",
		"px":                    ".a { inline-size: 12px; }",
		"nested":                ".a { &:hover { margin-right: 0; } }",
		"important":             ".a { display: none !important; }",
		"custom property":       ".a { --x: 1; }",
		"private literal":       ".a { --_x: 4px; }",
		"colour inside a shade": ".a { background: color-mix(in oklab, var(--ui-color-brand) 88%, black); }",
	}
	for name, css := range bad {
		if len(cssProblems("x.css", css)) == 0 {
			t.Errorf("%s: %q passed the checks", name, css)
		}
	}
	good := map[string]string{
		"logical":              ".a { margin-inline-start: var(--ui-space-1); padding-block: var(--ui-space-2); }",
		"symmetric shorthand":  ".a { padding: var(--ui-space-1) var(--ui-space-2); margin: 0 auto 0 auto; }",
		"hairline":             ".a { border: 1px solid var(--ui-color-border); }",
		"relative units":       ".a { inline-size: 1em; max-inline-size: 60ch; max-block-size: 90dvh; flex: 1 1 0%; }",
		"translate in dir":     ".a { &:dir(rtl) { transform: scaleX(-1) translateX(0); } }",
		"vertical translate":   ".a { transform: translateY(var(--ui-space-1)); }",
		"mix of tokens":        ".a { background: color-mix(in oklab, var(--ui-color-brand) 88%, var(--ui-color-text)); }",
		"media query literals": "@media (min-width: 40rem) { .a { gap: var(--ui-space-2); } }",
		"hidden":               "[hidden] { display: none !important; }",
		"private alias":        ".a { --_fill: var(--ui-color-brand); }",
	}
	for name, css := range good {
		file := "x.css"
		if name == "hidden" {
			file = "reset.css"
		}
		if p := cssProblems(file, css); len(p) != 0 {
			t.Errorf("%s: %q rejected: %v", name, css, p)
		}
	}
}
