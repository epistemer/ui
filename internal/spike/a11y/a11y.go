// Package a11y is the D4 spike of two uitest checks (visual-design.md §7,
// §8): accessible names, and the selectors that tie markup to ui's CSS.
// It is internal: adding them to uitest is an interface amendment.
package a11y

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Parse reads a fragment as body content, or a whole page.
func Parse(src string) ([]*nethtml.Node, error) {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(src)), "<!doctype") {
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

func hiddenNode(n *nethtml.Node) bool {
	if n.Type != nethtml.ElementNode {
		return false
	}
	if _, ok := attr(n, "hidden"); ok {
		return true
	}
	v, _ := attr(n, "aria-hidden")
	return v == "true"
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

// text is n's visible text, as a name from content counts it: text nodes
// and the alt of images, outside hidden subtrees.
func text(n *nethtml.Node) string {
	var b strings.Builder
	var visit func(*nethtml.Node)
	visit = func(n *nethtml.Node) {
		if hiddenNode(n) {
			return
		}
		switch n.Type {
		case nethtml.TextNode:
			b.WriteString(n.Data)
		case nethtml.ElementNode:
			if n.DataAtom == atom.Img {
				alt, _ := attr(n, "alt")
				b.WriteString(" " + alt + " ")
			}
			if l, ok := attr(n, "aria-label"); ok && strings.TrimSpace(l) != "" {
				b.WriteString(" " + l + " ")
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

var nameFromContent = map[string]bool{"button": true, "link": true, "menuitem": true, "tab": true, "option": true, "heading": true}

// Names lists every element that needs an accessible name and has none
// (design §7), and every fieldset without a legend, and landmarks that
// share a name.
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
			if id, ok := attr(n, "id"); ok {
				ids[id] = n
			}
			if n.DataAtom == atom.Label {
				if f, ok := attr(n, "for"); ok {
					labelFor[f] += " " + text(n)
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
					parts = append(parts, text(t))
				}
			}
			if s := strings.TrimSpace(strings.Join(parts, " ")); s != "" {
				return s
			}
		}
		if l, ok := attr(n, "aria-label"); ok && strings.TrimSpace(l) != "" {
			return l
		}
		if id, ok := attr(n, "id"); ok && strings.TrimSpace(labelFor[id]) != "" {
			return labelFor[id]
		}
		for p := n.Parent; p != nil; p = p.Parent {
			if p.DataAtom == atom.Label {
				return text(p)
			}
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
		role, _ := attr(n, "role")
		typ, _ := attr(n, "type")
		switch {
		case n.DataAtom == atom.Img:
			if _, ok := attr(n, "alt"); !ok && name(n) == "" {
				report(n, "an image without alt (alt=\"\" marks a decorative one)")
			}
		case n.DataAtom == atom.Input && typ == "hidden":
		case n.DataAtom == atom.Input && (typ == "submit" || typ == "button" || typ == "reset"):
			if v, _ := attr(n, "value"); strings.TrimSpace(v) == "" && name(n) == "" {
				report(n, "a button input without a value or label")
			}
		case n.DataAtom == atom.Input || n.DataAtom == atom.Select || n.DataAtom == atom.Textarea:
			if name(n) == "" {
				report(n, "a form control without a label (a placeholder is not one)")
			}
		case n.DataAtom == atom.Button || n.DataAtom == atom.A && hasAttr(n, "href") || nameFromContent[role]:
			if name(n) == "" && text(n) == "" {
				report(n, "no accessible name: no text, aria-label or aria-labelledby")
			}
		case n.DataAtom == atom.Dialog || role == "dialog" || role == "alertdialog":
			if name(n) == "" {
				report(n, "a dialog without aria-labelledby or aria-label")
			}
		case n.DataAtom == atom.Fieldset:
			legend := false
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.DataAtom == atom.Legend {
					legend = true
				}
			}
			if !legend && name(n) == "" {
				report(n, "a fieldset without a legend")
			}
		case n.DataAtom == atom.Nav:
			navNames = append(navNames, name(n))
		}
	})
	if len(navNames) > 1 {
		seen := map[string]bool{}
		for _, nn := range navNames {
			if nn == "" || seen[nn] {
				probs = append(probs, fmt.Sprintf("several <nav> landmarks need distinct names; got %q", navNames))
				break
			}
			seen[nn] = true
		}
	}
	return probs, nil
}

func hasAttr(n *nethtml.Node, key string) bool { _, ok := attr(n, key); return ok }

// Used lists what markup asks of ui's CSS: every ui-* class, and every
// value of ui's variant attributes, as CSS would select them.
func Used(src string) ([]string, error) {
	nodes, err := Parse(src)
	if err != nil {
		return nil, err
	}
	var out []string
	var visit func(*nethtml.Node)
	visit = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			cls, _ := attr(n, "class")
			ui := false
			for _, c := range strings.Fields(cls) {
				if strings.HasPrefix(c, "ui-") {
					out = append(out, "."+c)
					ui = true
				}
			}
			if ui {
				for _, a := range n.Attr {
					if slices.Contains(VariantAttrs, a.Key) {
						out = append(out, fmt.Sprintf("[%s=%q]", a.Key, a.Val))
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

// VariantAttrs are ui's variant attributes (Q3).
var VariantAttrs = []string{"data-variant", "data-size", "data-tone"}

var (
	cssClass   = regexp.MustCompile(`\.ui-[a-z0-9-]+`)
	cssVariant = regexp.MustCompile(`\[(data-variant|data-size|data-tone)="[^"]*"\]`)
	cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// Selected lists the ui-* classes and variant values ui's CSS selects.
func Selected(css string) []string {
	css = cssComment.ReplaceAllString(css, "")
	out := cssClass.FindAllString(css, -1)
	out = append(out, cssVariant.FindAllString(css, -1)...)
	slices.Sort(out)
	return slices.Compact(out)
}
