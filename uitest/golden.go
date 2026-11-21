package uitest

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() {
	if flag.Lookup("update") == nil {
		flag.Bool("update", false, "update golden files")
	}
}

func isUpdate() bool {
	f := flag.Lookup("update")
	return f != nil && f.Value.String() == "true"
}

// Golden normalises html and compares it with testdata/<name>.golden.html
// relative to the test's working directory. With the -update flag, it writes
// the file (creating testdata/) and passes.
func Golden(t testing.TB, name, rawHTML string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden.html")
	goldenAt(t, path, rawHTML, isUpdate())
}

func goldenAt(t testing.TB, path, rawHTML string, update bool) {
	t.Helper()
	normalised := normalise(rawHTML)
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("uitest: %v", err)
			return
		}
		if err := os.WriteFile(path, []byte(normalised), 0o644); err != nil {
			t.Fatalf("uitest: %v", err)
			return
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Fatalf("uitest: golden file %s missing; run go test -update", path)
			return
		}
		t.Fatalf("uitest: %v", err)
		return
	}
	golden := strings.ReplaceAll(string(data), "\r\n", "\n")
	if golden != normalised {
		diff := diffLines(golden, normalised)
		t.Errorf("uitest: golden mismatch in %s:\n%s", path, diff)
	}
}

var voidElements = map[string]bool{
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"keygen": true,
	"link":   true,
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,
}

func isVoidElement(tag string) bool {
	return voidElements[strings.ToLower(tag)]
}

func isHTMLSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func cleanText(s string) string {
	s = strings.TrimFunc(s, isHTMLSpace)
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if isHTMLSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}

func firstElementTag(s string) string {
	z := nethtml.NewTokenizer(strings.NewReader(s))
	for {
		tt := z.Next()
		switch tt {
		case nethtml.ErrorToken:
			return ""
		case nethtml.StartTagToken, nethtml.SelfClosingTagToken:
			name, _ := z.TagName()
			return strings.ToLower(string(name))
		}
	}
}

func fragmentContext(firstTag string) *nethtml.Node {
	var ctxTag string
	switch firstTag {
	case "tr":
		ctxTag = "tbody"
	case "td", "th":
		ctxTag = "tr"
	case "option", "optgroup":
		ctxTag = "select"
	case "thead", "tbody", "tfoot", "caption", "colgroup":
		ctxTag = "table"
	case "col":
		ctxTag = "colgroup"
	case "li":
		ctxTag = "ul"
	case "dt", "dd":
		ctxTag = "dl"
	default:
		ctxTag = "body"
	}
	return &nethtml.Node{
		Type:     nethtml.ElementNode,
		Data:     ctxTag,
		DataAtom: atom.Lookup([]byte(ctxTag)),
	}
}

// parseFragment parses a fragment in the context its first element
// needs; a whole page (starting at <html>, <head> or <body>) is parsed as
// a document, so the page shell and its attributes are kept.
func parseFragment(rawHTML string) ([]*nethtml.Node, error) {
	switch tag := firstElementTag(rawHTML); tag {
	case "html", "head", "body":
		doc, err := nethtml.Parse(strings.NewReader(rawHTML))
		if err != nil {
			return nil, err
		}
		var nodes []*nethtml.Node
		for c := doc.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == nethtml.ElementNode {
				nodes = append(nodes, c)
			}
		}
		return nodes, nil
	default:
		return nethtml.ParseFragment(strings.NewReader(rawHTML), fragmentContext(tag))
	}
}

func attrName(a nethtml.Attribute) string {
	if a.Namespace != "" {
		return a.Namespace + ":" + a.Key
	}
	return a.Key
}

func hasChildren(n *nethtml.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case nethtml.ElementNode:
			return true
		case nethtml.TextNode:
			if cleanText(c.Data) != "" {
				return true
			}
		}
	}
	return false
}

func renderAttrs(attrs []nethtml.Attribute) string {
	if len(attrs) == 0 {
		return ""
	}
	sorted := make([]nethtml.Attribute, len(attrs))
	copy(sorted, attrs)
	slices.SortFunc(sorted, func(a, b nethtml.Attribute) int {
		return cmp.Compare(attrName(a), attrName(b))
	})
	var sb strings.Builder
	for _, a := range sorted {
		sb.WriteByte(' ')
		sb.WriteString(attrName(a))
		sb.WriteString(`="`)
		sb.WriteString(html.EscapeString(a.Val))
		sb.WriteByte('"')
	}
	return sb.String()
}

func normaliseNode(sb *strings.Builder, n *nethtml.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch n.Type {
	case nethtml.TextNode:
		t := cleanText(n.Data)
		if t != "" {
			sb.WriteString(indent)
			sb.WriteString(html.EscapeString(t)) // text "<b>" never reads as an element
			sb.WriteByte('\n')
		}
	case nethtml.ElementNode:
		attrs := renderAttrs(n.Attr)
		tag := n.Data
		if isVoidElement(tag) {
			sb.WriteString(indent)
			sb.WriteByte('<')
			sb.WriteString(tag)
			sb.WriteString(attrs)
			sb.WriteString(">\n")
			return
		}
		if !hasChildren(n) {
			sb.WriteString(indent)
			sb.WriteByte('<')
			sb.WriteString(tag)
			sb.WriteString(attrs)
			sb.WriteString("></")
			sb.WriteString(tag)
			sb.WriteString(">\n")
			return
		}
		sb.WriteString(indent)
		sb.WriteByte('<')
		sb.WriteString(tag)
		sb.WriteString(attrs)
		sb.WriteString(">\n")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			normaliseNode(sb, c, depth+1)
		}
		sb.WriteString(indent)
		sb.WriteString("</")
		sb.WriteString(tag)
		sb.WriteString(">\n")
	}
}

func normalise(rawHTML string) string {
	nodes, err := parseFragment(rawHTML)
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, n := range nodes {
		normaliseNode(&sb, n, 0)
	}
	return sb.String()
}

func splitLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

type diffOp struct {
	op   byte
	num  int
	text string
}

func diffLines(golden, got string) string {
	g := splitLines(golden)
	a := splitLines(got)
	n := len(g)
	m := len(a)

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if g[i-1] == a[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var ops []diffOp
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && g[i-1] == a[j-1] {
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, diffOp{op: '+', num: j, text: a[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i][j-1] < dp[i-1][j]) {
			ops = append(ops, diffOp{op: '-', num: i, text: g[i-1]})
			i--
		}
	}

	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	var sb strings.Builder
	for _, op := range ops {
		sb.WriteString(fmt.Sprintf("%c %d: %s\n", op.op, op.num, op.text))
	}
	return sb.String()
}
