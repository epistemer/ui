package ui_test

import (
	"context"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
)

var validID = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func decodeKeyParts(idStr, prefix string) []string {
	body := strings.TrimPrefix(idStr, prefix+"-")
	tokens := strings.Split(body, "-")
	parts := make([]string, len(tokens))
	for i, tok := range tokens {
		parts[i] = decodePart(tok)
	}
	return parts
}

func decodePart(tok string) string {
	if tok == "_" {
		return ""
	}
	var b strings.Builder
	for j := 0; j < len(tok); {
		if tok[j] == '_' && j+2 < len(tok) && isLowerHex(tok[j+1]) && isLowerHex(tok[j+2]) {
			b.WriteByte((fromHex(tok[j+1]) << 4) | fromHex(tok[j+2]))
			j += 3
		} else {
			b.WriteByte(tok[j])
			j++
		}
	}
	return b.String()
}

func isLowerHex(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f')
}

func fromHex(c byte) byte {
	if '0' <= c && c <= '9' {
		return c - '0'
	}
	return c - 'a' + 10
}

func FuzzKey(f *testing.F) {
	for _, seed := range []string{"", "1042", "A/7", "a-b", "_", "-", "é", "\x00", "a b"} {
		f.Add(seed)
	}
	row := ui.NewID("row")
	f.Fuzz(func(t *testing.T, p string) {
		k1 := row.Key(p)
		if !validID.MatchString(k1) {
			t.Fatalf("Key(%q) = %q does not match identifier syntax", p, k1)
		}
		if strings.Contains(k1, "--") {
			t.Fatalf("Key(%q) = %q contains --", p, k1)
		}
		decoded1 := decodeKeyParts(k1, "row")
		if len(decoded1) != 1 || decoded1[0] != p {
			t.Fatalf("Key(%q) decoded to %v, want [%q]", p, decoded1, p)
		}
		if k1 == row.Key(p+"x") {
			t.Fatalf("Key(%q) == Key(%q)", p, p+"x")
		}
		if k1 == row.Key("x"+p) {
			t.Fatalf("Key(%q) == Key(%q)", p, "x"+p)
		}

		k2 := row.Key(p, p+"x")
		if !validID.MatchString(k2) {
			t.Fatalf("Key(%q, %q) = %q does not match identifier syntax", p, p+"x", k2)
		}
		if strings.Contains(k2, "--") {
			t.Fatalf("Key(%q, %q) = %q contains --", p, p+"x", k2)
		}
		decoded2 := decodeKeyParts(k2, "row")
		if len(decoded2) != 2 || decoded2[0] != p || decoded2[1] != p+"x" {
			t.Fatalf("Key(%q, %q) decoded to %v, want [%q, %q]", p, p+"x", decoded2, p, p+"x")
		}
	})
}

func FuzzPart(f *testing.F) {
	for _, seed := range []string{"", "1042", "A/7", "a-b", "_", "-", "é", "\x00", "a b"} {
		f.Add(seed)
	}
	id := ui.NewID("row")
	f.Fuzz(func(t *testing.T, p string) {
		ctx := ui.WithIDs(context.Background(), ui.Sequence("t"))
		var get, part string
		c := ui.Scope(id)
		child := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
			get = id.Get(ctx)
			part = id.Part(ctx, p)
			return nil
		})
		if err := c.Render(templ.WithChildren(ctx, child), io.Discard); err != nil {
			t.Fatalf("Scope render: %v", err)
		}
		if !strings.Contains(part, "--") {
			t.Fatalf("Part(%q) = %q, want it to contain --", p, part)
		}
		if !validID.MatchString(part) {
			t.Fatalf("Part(%q) = %q does not match identifier syntax", p, part)
		}
		prefix := get + "-"
		if !strings.HasPrefix(part, prefix) {
			t.Fatalf("Part(%q) = %q does not have prefix %q", p, part, prefix)
		}
		decoded := decodePart(strings.TrimPrefix(part, prefix))
		if decoded != p {
			t.Fatalf("Part(%q) decoded to %q, want %q", p, decoded, p)
		}
	})
}

func TestKeyedNeverEqualsGenerated(t *testing.T) {
	id := ui.NewID("row")
	const count = 1000
	generated := make([]string, 0, count)
	c := ui.Scope(id)
	grab := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error {
		generated = append(generated, id.Get(ctx))
		return nil
	})
	ctx := templ.WithChildren(context.Background(), grab)
	for range count {
		if err := c.Render(ctx, io.Discard); err != nil {
			t.Fatalf("Scope render: %v", err)
		}
	}
	if len(generated) != count {
		t.Fatalf("got %d generated ids, want %d", len(generated), count)
	}
	seen := make(map[string]bool, count)
	for _, gen := range generated {
		if !strings.Contains(gen, "--") {
			t.Fatalf("generated id %q does not contain --", gen)
		}
		seen[gen] = true
	}
	for _, gen := range generated {
		parts := strings.SplitN(gen, "--", 2)
		if len(parts) != 2 {
			t.Fatalf("generated id %q missing -- delimiter", gen)
		}
		tok := parts[1]
		keyed := id.Key(tok)
		if strings.Contains(keyed, "--") {
			t.Fatalf("keyed id %q contains --", keyed)
		}
		if seen[keyed] {
			t.Fatalf("keyed id %q equals a generated id", keyed)
		}
	}
}
