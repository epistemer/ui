package ui_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
)

var validID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// decodeParts strictly decodes what follows "<base>-" in an id: parts
// joined by "-", each "_" alone for an empty part, else letters, digits
// and "_hh" escapes in lower-case hex. Anything else is an error.
func decodeParts(id, base string) ([]string, error) {
	body, ok := strings.CutPrefix(id, base+"-")
	if !ok {
		return nil, fmt.Errorf("%q does not start with %q", id, base+"-")
	}
	var parts []string
	for _, tok := range strings.Split(body, "-") {
		p, err := decodePart(tok)
		if err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

func decodePart(tok string) (string, error) {
	if tok == "_" {
		return "", nil
	}
	if tok == "" {
		return "", fmt.Errorf("an empty encoded part")
	}
	var b strings.Builder
	for j := 0; j < len(tok); {
		c := tok[j]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
			b.WriteByte(c)
			j++
		case c == '_' && j+2 < len(tok) && isLowerHex(tok[j+1]) && isLowerHex(tok[j+2]):
			b.WriteByte(fromHex(tok[j+1])<<4 | fromHex(tok[j+2]))
			j += 3
		default:
			return "", fmt.Errorf("%q: byte %q at %d is not letter, digit or _hh", tok, c, j)
		}
	}
	return b.String(), nil
}

func isLowerHex(c byte) bool { return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' }

func fromHex(c byte) byte {
	if c <= '9' {
		return c - '0'
	}
	return c - 'a' + 10
}

var seeds = []string{"", "1042", "A/7", "a-b", "_", "-", "--", "_2d", "é", "\x00", "a b"}

func FuzzKey(f *testing.F) {
	for _, a := range seeds {
		f.Add(a, "x")
	}
	row := ui.NewID("order-row")
	f.Fuzz(func(t *testing.T, p1, p2 string) {
		for _, parts := range [][]string{{p1}, {p1, p2}} {
			k := row.Key(parts...)
			if !validID.MatchString(k) || strings.Contains(k, "--") {
				t.Fatalf("Key(%q) = %q: not an identifier without --", parts, k)
			}
			got, err := decodeParts(k, "order-row")
			if err != nil {
				t.Fatalf("Key(%q) = %q: %v", parts, k, err)
			}
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", parts) {
				t.Fatalf("Key(%q) = %q decodes to %q", parts, k, got)
			}
		}
		if p1 != p2 && row.Key(p1) == row.Key(p2) {
			t.Fatalf("Key(%q) == Key(%q)", p1, p2)
		}
		if row.Key(p1, p2) == row.Key(p1+"-"+p2) {
			t.Fatalf("two parts collide with one joined by -")
		}
	})
}

func FuzzPart(f *testing.F) {
	for _, a := range seeds {
		f.Add(a, "hint")
	}
	id := ui.NewID("row")
	f.Fuzz(func(t *testing.T, p1, p2 string) {
		for _, parts := range [][]string{{p1}, {p1, p2}} {
			var get, part string
			child := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
				get, part = id.Get(ctx), id.Part(ctx, parts...)
				return nil
			})
			ctx := templ.WithChildren(ui.WithIDs(context.Background(), ui.Sequence("t")), child)
			if err := ui.Scope(id).Render(ctx, io.Discard); err != nil {
				t.Fatal(err)
			}
			if !validID.MatchString(part) || !strings.Contains(part, "--") {
				t.Fatalf("Part(%q) = %q: not a generated identifier", parts, part)
			}
			got, err := decodeParts(part, get)
			if err != nil {
				t.Fatalf("Part(%q) = %q: %v", parts, part, err)
			}
			if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", parts) {
				t.Fatalf("Part(%q) = %q decodes to %q", parts, part, got)
			}
		}
	})
}

func TestKeyedNeverEqualsGenerated(t *testing.T) {
	id := ui.NewID("row")
	var generated []string
	grab := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
		generated = append(generated, id.Get(ctx))
		return nil
	})
	ctx := templ.WithChildren(context.Background(), grab)
	for range 1000 {
		if err := ui.Scope(id).Render(ctx, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	seen := make(map[string]bool, len(generated))
	for _, g := range generated {
		seen[g] = true
	}
	for _, g := range generated {
		_, tok, _ := strings.Cut(g, "--")
		for _, k := range []string{id.Key(tok), id.Key("-" + tok), id.Key("", tok), id.Key(strings.Split(tok, "-")...)} {
			if strings.Contains(k, "--") || seen[k] {
				t.Fatalf("keyed id %q looks generated or equals one", k)
			}
		}
	}
}
