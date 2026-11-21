package uitest

import (
	"strings"
	"testing"

	nethtml "golang.org/x/net/html"
)

// RefOption changes how CheckIDs checks references.
type RefOption func(*refOptions)

type refOptions struct {
	external map[string]bool
}

// External lists ids a fragment may reference outside itself.
func External(ids ...string) RefOption {
	return func(o *refOptions) {
		if o.external == nil {
			o.external = make(map[string]bool)
		}
		for _, id := range ids {
			o.external[id] = true
		}
	}
}

// CheckIDs checks that every id in rawHTML is unique and non-empty,
// and that every id reference resolves to an id in the document or in External.
func CheckIDs(t testing.TB, rawHTML string, opts ...RefOption) {
	t.Helper()
	var o refOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	nodes, err := parseFragment(rawHTML)
	if err != nil {
		t.Fatalf("uitest: parse: %v", err)
		return
	}
	checkDocument(t, nodes, o.external)
}

func checkDocument(t testing.TB, rootNodes []*nethtml.Node, external map[string]bool) {
	t.Helper()
	seenIDs := make(map[string]bool)

	// Pass 1: collect IDs in this document scope (excluding <template> children).
	var collectIDs func(n *nethtml.Node)
	collectIDs = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			for _, a := range n.Attr {
				if a.Key == "id" {
					if a.Val == "" {
						t.Errorf("uitest: <%s> attribute %q: empty id %q", n.Data, a.Key, "")
					} else if strings.ContainsAny(a.Val, " \t\n\f\r") {
						t.Errorf("uitest: <%s> attribute %q: id %q contains white space", n.Data, a.Key, a.Val)
					} else if seenIDs[a.Val] {
						t.Errorf("uitest: <%s> attribute %q: duplicate id %q", n.Data, a.Key, a.Val)
					} else {
						seenIDs[a.Val] = true
					}
				}
			}
			if n.Data == "template" {
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			collectIDs(c)
		}
	}

	for _, n := range rootNodes {
		collectIDs(n)
	}

	// Pass 2: check references in this document scope (excluding <template> children).
	var checkRefs func(n *nethtml.Node)
	checkRefs = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			for _, a := range n.Attr {
				checkAttrRef(t, n.Data, a, seenIDs, external)
			}
			if n.Data == "template" {
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkRefs(c)
		}
	}

	for _, n := range rootNodes {
		checkRefs(n)
	}

	// Pass 3: recursively check each <template> element's content as its own document.
	var checkTemplates func(n *nethtml.Node)
	checkTemplates = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "template" {
			var tmplChildren []*nethtml.Node
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				tmplChildren = append(tmplChildren, c)
			}
			checkDocument(t, tmplChildren, external)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			checkTemplates(c)
		}
	}

	for _, n := range rootNodes {
		checkTemplates(n)
	}
}

func checkAttrRef(t testing.TB, tag string, a nethtml.Attribute, seenIDs, external map[string]bool) {
	switch a.Key {
	case "for", "form", "list", "popovertarget", "commandfor", "aria-activedescendant", "aria-errormessage":
		if a.Val == "" {
			t.Errorf("uitest: <%s> attribute %q: empty reference to id %q", tag, a.Key, "")
			return
		}
		if !seenIDs[a.Val] && !external[a.Val] {
			t.Errorf("uitest: <%s> attribute %q references missing id %q", tag, a.Key, a.Val)
		}
	case "headers", "aria-controls", "aria-labelledby", "aria-describedby", "aria-owns", "aria-flowto", "aria-details":
		parts := strings.FieldsFunc(a.Val, isHTMLSpace)
		if len(parts) == 0 {
			t.Errorf("uitest: <%s> attribute %q: empty id list %q", tag, a.Key, "")
			return
		}
		for _, id := range parts {
			if !seenIDs[id] && !external[id] {
				t.Errorf("uitest: <%s> attribute %q references missing id %q", tag, a.Key, id)
			}
		}
	case "href":
		if strings.HasPrefix(a.Val, "#") && a.Val != "#" {
			id := a.Val[1:]
			if !seenIDs[id] && !external[id] {
				t.Errorf("uitest: <%s> attribute %q references missing id %q", tag, "href", id)
			}
		}
	case "hx-target", "hx-include", "hx-indicator", "hx-select", "hx-disabled-elt":
		if id, ok := isIDSelector(a.Val); ok {
			if !seenIDs[id] && !external[id] {
				t.Errorf("uitest: <%s> attribute %q references missing id %q", tag, a.Key, id)
			}
		}
	}
}

func isIDSelector(s string) (string, bool) {
	if !strings.HasPrefix(s, "#") || len(s) == 1 {
		return "", false
	}
	id := s[1:]
	for _, r := range id {
		if isHTMLSpace(r) || strings.ContainsRune(".#>:[],+~=", r) {
			return "", false
		}
	}
	return id, true
}
