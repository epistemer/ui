// Package a11y holds the checks behind uitest's CheckNames and
// CheckSelectors (interface amendment A4, visual-interface.md §13.1):
// accessible names, and the selectors that tie markup to CSS. uitest and
// catalogtest share it. Not API.
package a11y

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var leadingComments = regexp.MustCompile(`^(\s*<!--.*?-->)*\s*`)

// Parse reads a whole page (one that starts with a doctype or <html>), or
// else a fragment as body content.
func Parse(src string) ([]*nethtml.Node, error) {
	head := strings.ToLower(leadingComments.ReplaceAllString(src, ""))
	if strings.HasPrefix(head, "<!doctype") || strings.HasPrefix(head, "<html") {
		doc, err := nethtml.Parse(strings.NewReader(src))
		if err != nil {
			return nil, err
		}
		return []*nethtml.Node{doc}, nil
	}
	body := &nethtml.Node{Type: nethtml.ElementNode, Data: "body", DataAtom: atom.Body}
	return nethtml.ParseFragment(strings.NewReader(src), body)
}

func attr(n *nethtml.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func lowerAttr(n *nethtml.Node, key string) string {
	v, _ := attr(n, key)
	return strings.ToLower(strings.TrimSpace(v))
}

func hasAttr(n *nethtml.Node, key string) bool { _, ok := attr(n, key); return ok }

func hiddenNode(n *nethtml.Node) bool {
	if n.Type != nethtml.ElementNode {
		return false
	}
	return hasAttr(n, "hidden") || lowerAttr(n, "aria-hidden") == "true"
}

// walk visits elements outside hidden subtrees and template contents.
func walk(nodes []*nethtml.Node, fn func(*nethtml.Node)) {
	var visit func(*nethtml.Node)
	visit = func(n *nethtml.Node) {
		if hiddenNode(n) {
			return
		}
		if n.Type == nethtml.ElementNode {
			fn(n)
			if n.DataAtom == atom.Template {
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range nodes {
		visit(n)
	}
}

func isControl(n *nethtml.Node) bool {
	return n.DataAtom == atom.Input || n.DataAtom == atom.Select || n.DataAtom == atom.Textarea
}

// text is n's text as a name from content counts it: text nodes and the
// alt or aria-label of what's inside, outside hidden subtrees. Form
// controls inside it add nothing, so a label's text is its own words, not
// a nested select's options. A node referenced by aria-labelledby counts
// even when it is hidden itself (selfHidden).
func text(n *nethtml.Node, selfHidden bool) string {
	var b strings.Builder
	var visit func(*nethtml.Node, bool)
	visit = func(n *nethtml.Node, root bool) {
		if hiddenNode(n) && !(root && selfHidden) {
			return
		}
		switch n.Type {
		case nethtml.TextNode:
			b.WriteString(n.Data)
		case nethtml.ElementNode:
			if !root && isControl(n) {
				return
			}
			if !root {
				if l, ok := attr(n, "aria-label"); ok && strings.TrimSpace(l) != "" {
					b.WriteString(" " + l + " ")
					return
				}
			}
			if n.DataAtom == atom.Img {
				alt, _ := attr(n, "alt")
				b.WriteString(" " + alt + " ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c, false)
		}
	}
	visit(n, true)
	return strings.Join(strings.Fields(b.String()), " ")
}

// Roles whose name comes from their content, and roles of controls that
// need a label.
var (
	nameFromContent = map[string]bool{"button": true, "link": true, "menuitem": true, "menuitemcheckbox": true, "menuitemradio": true, "tab": true, "option": true, "heading": true}
	controlRoles    = map[string]bool{"checkbox": true, "switch": true, "radio": true, "textbox": true, "searchbox": true, "combobox": true, "listbox": true, "slider": true, "spinbutton": true}
)

// Names lists every element that needs an accessible name and has none
// (design §7), every fieldset without a legend, and landmarks that share
// a name.
func Names(src string) ([]string, error) {
	nodes, err := Parse(src)
	if err != nil {
		return nil, err
	}
	ids := map[string]*nethtml.Node{}
	labelFor := map[string]string{}
	var collect func(*nethtml.Node)
	collect = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			if n.DataAtom == atom.Template {
				return // inert until cloned; its ids and labels name nothing here
			}
			if id, ok := attr(n, "id"); ok {
				ids[id] = n
			}
			if n.DataAtom == atom.Label && !hiddenNode(n) {
				if f, ok := attr(n, "for"); ok {
					labelFor[f] += " " + text(n, false)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	for _, n := range nodes {
		collect(n)
	}
	name := func(n *nethtml.Node) string {
		if ref, ok := attr(n, "aria-labelledby"); ok {
			var parts []string
			for _, id := range strings.Fields(ref) {
				if t, ok := ids[id]; ok {
					parts = append(parts, text(t, true))
				}
			}
			if s := strings.Join(strings.Fields(strings.Join(parts, " ")), " "); s != "" {
				return s
			}
		}
		if l, ok := attr(n, "aria-label"); ok && strings.TrimSpace(l) != "" {
			return strings.Join(strings.Fields(l), " ")
		}
		if isControl(n) || n.DataAtom == atom.Button {
			if id, ok := attr(n, "id"); ok && strings.TrimSpace(labelFor[id]) != "" {
				return strings.Join(strings.Fields(labelFor[id]), " ")
			}
			for p := n.Parent; p != nil; p = p.Parent {
				if p.DataAtom == atom.Label {
					if t := text(p, false); t != "" {
						return t
					}
				}
			}
		}
		// title is the last resort (amendment A4), after content for the
		// elements named by their content.
		if nameFromContent[lowerAttr(n, "role")] || n.DataAtom == atom.Button || n.DataAtom == atom.A {
			if t := text(n, false); t != "" {
				return t
			}
		}
		if t, ok := attr(n, "title"); ok && strings.TrimSpace(t) != "" {
			return strings.Join(strings.Fields(t), " ")
		}
		return ""
	}
	var probs []string
	report := func(n *nethtml.Node, why string) {
		var b strings.Builder
		nethtml.Render(&b, &nethtml.Node{Type: nethtml.ElementNode, Data: n.Data, Attr: n.Attr})
		tag := b.String()
		if i := strings.Index(tag, ">"); i >= 0 {
			tag = tag[:i+1]
		}
		probs = append(probs, fmt.Sprintf("%s: %s", tag, why))
	}
	var navNames []string
	walk(nodes, func(n *nethtml.Node) {
		role := lowerAttr(n, "role")
		typ := lowerAttr(n, "type")
		switch {
		case n.DataAtom == atom.Img:
			if !hasAttr(n, "alt") && name(n) == "" && role != "presentation" && role != "none" {
				report(n, `an image without alt (alt="" marks a decorative one)`)
			}
		case n.DataAtom == atom.Input && typ == "hidden":
		case n.DataAtom == atom.Input && typ == "image":
			if alt, _ := attr(n, "alt"); strings.TrimSpace(alt) == "" && name(n) == "" {
				report(n, "an image button without alt")
			}
		case n.DataAtom == atom.Input && (typ == "submit" || typ == "button" || typ == "reset"):
			// An omitted value gives submit and reset the browser's own label;
			// an empty one, or a button input, gives none.
			v, has := attr(n, "value")
			if strings.TrimSpace(v) == "" && name(n) == "" && (typ == "button" || has) {
				report(n, "a button input without a value or label")
			}
		case isControl(n) || controlRoles[role]:
			if name(n) == "" {
				report(n, "a form control without a label (a placeholder is not one)")
			}
		case n.DataAtom == atom.Button || n.DataAtom == atom.A && hasAttr(n, "href") || nameFromContent[role]:
			if name(n) == "" && text(n, false) == "" {
				report(n, "no accessible name: no text, aria-label or aria-labelledby")
			}
		case n.DataAtom == atom.Dialog || role == "dialog" || role == "alertdialog":
			if name(n) == "" {
				report(n, "a dialog without aria-labelledby or aria-label")
			}
		case n.DataAtom == atom.Fieldset:
			legend := ""
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.DataAtom == atom.Legend {
					legend = text(c, false)
					break
				}
			}
			if legend == "" && name(n) == "" {
				report(n, "a fieldset without a legend that says something")
			}
		case n.DataAtom == atom.Nav || role == "navigation":
			navNames = append(navNames, name(n))
		}
	})
	if len(navNames) > 1 {
		seen := map[string]bool{}
		for _, nn := range navNames {
			if nn == "" || seen[nn] {
				probs = append(probs, fmt.Sprintf("several navigation landmarks need distinct names; got %q", navNames))
				break
			}
			seen[nn] = true
		}
	}
	return probs, nil
}

// VariantAttrs are ui's variant attributes (Q3), selected by value.
// Other data-* attributes on ui's elements are selected by presence;
// data-ui-* attributes are behaviour hooks for the ui script, not styles.
var VariantAttrs = []string{"data-variant", "data-size", "data-tone"}

func selector(class, key, val string) string {
	if slices.Contains(VariantAttrs, key) {
		return fmt.Sprintf("%s[%s=%q]", class, key, val)
	}
	return fmt.Sprintf("%s[%s]", class, key)
}

// Used lists what markup asks of ui's CSS: every ui-* class, and with
// each, the data attributes on the same element, as compound selectors:
// .ui-button, .ui-button[data-variant="outline"], .ui-icon[data-mirror].
func Used(src string, prefixes []string) ([]string, error) {
	covered := func(class string) bool {
		for _, p := range prefixes {
			if p != "" && len(class) > len(p) && strings.HasPrefix(class, p) {
				return true
			}
		}
		return false
	}
	nodes, err := Parse(src)
	if err != nil {
		return nil, err
	}
	var out []string
	var visit func(*nethtml.Node)
	visit = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			cls, _ := attr(n, "class")
			for _, c := range strings.Fields(cls) {
				if !covered(c) {
					continue
				}
				out = append(out, "."+c)
				for _, a := range n.Attr {
					if strings.HasPrefix(a.Key, "data-") && !strings.HasPrefix(a.Key, "data-ui-") {
						out = append(out, selector("."+c, a.Key, a.Val))
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range nodes {
		visit(n)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

var (
	dataAttr = regexp.MustCompile(`\[(data-[a-z0-9-]+)(?:\s*([~|^$*]?)=\s*(?:"([^"]*)"|'([^']*)'|([^\]\s]+))(?:\s+[iIsS])?)?\s*\]`)
)

// Selected lists, in the same form as Used, what ui's CSS selects. It
// resolves nesting (& and descendant rules) and reads each compound
// selector on its own, so .ui-button[data-variant="outline"] needs both in
// one compound, not anywhere in the file.
func Selected(css string, prefixes []string) []string {
	var alts []string
	for _, p := range prefixes {
		if p != "" {
			alts = append(alts, regexp.QuoteMeta(p))
		}
	}
	if len(alts) == 0 {
		return nil
	}
	classRE := regexp.MustCompile(`\.(?:` + strings.Join(alts, "|") + `)[A-Za-z0-9_-]+`)
	css = stripComments(css)
	var out []string
	var stack [][]string // resolved selector lists; nil for at-rule blocks
	var buf strings.Builder
	depth := 0 // parentheses and brackets
	context := func() []string {
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i] != nil {
				return stack[i]
			}
		}
		return nil
	}
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(css) && css[j] != c {
				if css[j] == '\\' {
					j++
				}
				j++
			}
			buf.WriteString(css[i:min(j+1, len(css))])
			i = j
		case c == '(' || c == '[':
			depth++
			buf.WriteByte(c)
		case c == ')' || c == ']':
			depth--
			buf.WriteByte(c)
		case depth > 0:
			buf.WriteByte(c)
		case c == '{':
			prelude := strings.TrimSpace(buf.String())
			buf.Reset()
			if strings.HasPrefix(prelude, "@") {
				stack = append(stack, nil)
				continue
			}
			resolved := resolve(context(), splitTop(prelude, ','))
			stack = append(stack, resolved)
			for _, sel := range resolved {
				out = append(out, compounds(sel, classRE)...)
			}
		case c == ';':
			buf.Reset()
		case c == '}':
			buf.Reset()
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			buf.WriteByte(c)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// resolve joins nested selectors to their parents: & stands for the
// parent; without it, the child is a descendant.
func resolve(parents, children []string) []string {
	if len(parents) == 0 {
		return children
	}
	var out []string
	for _, p := range parents {
		for _, c := range children {
			if strings.Contains(c, "&") {
				out = append(out, strings.ReplaceAll(c, "&", p))
			} else {
				out = append(out, p+" "+c)
			}
		}
	}
	return out
}

// splitTop splits s on sep outside parentheses, brackets and quotes.
func splitTop(s string, sep byte) []string {
	var parts []string
	depth, from := 0, 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[':
			depth++
		case c == ')' || c == ']':
			depth--
		case c == sep && depth == 0:
			parts = append(parts, strings.TrimSpace(s[from:i]))
			from = i + 1
		}
	}
	return append(parts, strings.TrimSpace(s[from:]))
}

// compounds lists, for one full selector, every ui-* class with the data
// attributes of its own compound. What's inside :not() and other
// functions is left out: it excludes or tests, it doesn't select.
func compounds(sel string, classRE *regexp.Regexp) []string {
	var out []string
	for _, part := range splitCompounds(sel) {
		flat := stripFunctions(part)
		classes := classRE.FindAllString(flat, -1)
		attrs := dataAttr.FindAllStringSubmatch(flat, -1)
		for _, cls := range classes {
			out = append(out, cls)
			for _, a := range attrs {
				val := a[3] + a[4] + a[5]
				switch a[2] {
				case "", "~", "|": // selects that value
					out = append(out, selector(cls, a[1], val))
				default: // ^= $= *= select a family of values: count it as presence
					out = append(out, fmt.Sprintf("%s[%s]", cls, a[1]))
				}
			}
		}
	}
	return out
}

// splitCompounds splits a selector at its combinators, outside brackets,
// parentheses and quotes.
func splitCompounds(sel string) []string {
	var parts []string
	var b strings.Builder
	depth := 0
	var quote byte
	flush := func() {
		if s := strings.TrimSpace(b.String()); s != "" {
			parts = append(parts, s)
		}
		b.Reset()
	}
	for i := 0; i < len(sel); i++ {
		c := sel[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			b.WriteByte(c)
		case c == '"' || c == '\'':
			quote = c
			b.WriteByte(c)
		case c == '(' || c == '[':
			depth++
			b.WriteByte(c)
		case c == ')' || c == ']':
			depth--
			b.WriteByte(c)
		case depth == 0 && (c == ' ' || c == '\t' || c == '\n' || c == '>' || c == '+' || c == '~'):
			flush()
		default:
			b.WriteByte(c)
		}
	}
	flush()
	return parts
}

// stripFunctions removes the arguments of :not(), :has(), :dir() and the
// other functions, which exclude or test rather than select. The arguments
// of :is() and :where() select, so they stay. Brackets (attribute
// selectors) stay as they are.
func stripFunctions(s string) string {
	var b strings.Builder
	var keep []bool // per open parenthesis: whether its content is kept
	dropping := func() bool { return slices.Contains(keep, false) }
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '(':
			k := strings.HasSuffix(s[:i], ":is") || strings.HasSuffix(s[:i], ":where")
			keep = append(keep, k)
			if k && !dropping() {
				b.WriteByte(' ')
			}
		case ')':
			if len(keep) > 0 {
				keep = keep[:len(keep)-1]
			}
		default:
			if !dropping() {
				b.WriteByte(c)
			}
		}
	}
	return b.String()
}

// stripComments removes CSS comments, leaving strings alone, so a "/*"
// inside a string can't swallow rules.
func stripComments(css string) string {
	var b strings.Builder
	for i := 0; i < len(css); i++ {
		switch c := css[i]; {
		case c == '"' || c == '\'':
			j := i + 1
			for j < len(css) && css[j] != c {
				if css[j] == '\\' {
					j++
				}
				j++
			}
			b.WriteString(css[i:min(j+1, len(css))])
			i = j
		case c == '/' && i+1 < len(css) && css[i+1] == '*':
			end := strings.Index(css[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			b.WriteByte(' ')
			i += end + 3
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
