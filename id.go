package ui

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/a-h/templ"
)

// ID is a handle for element ids that wire elements together: a label's
// for and its input's id, aria-describedby and the hint it names, an
// hx-target and the element it swaps.
//
// It gives two kinds of id. Generated ids (Get, Part) are fresh for each
// Scope and wire elements inside one render. Keyed ids (Key) are built
// from data, the same on every render, for elements another request
// targets. A generated id always contains "--" and a keyed id never does,
// so the two never collide. Every id is a valid HTML id and CSS
// identifier.
//
// Declare an ID once, in a package-level variable. Copies refer to the
// same handle.
type ID struct{ h *idHandle }

type idHandle struct {
	name string
}

var idName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// NewID declares an id handle. It panics unless name is lower-case letters
// and digits in hyphen-separated words, starting with a letter, such as
// "email" or "cart-dialog".
func NewID(name string) ID {
	if !idName.MatchString(name) {
		panic(fmt.Sprintf("ui: NewID(%q): an id name is lower-case words of letters and digits joined by single hyphens, starting with a letter", name))
	}
	return ID{&idHandle{name: name}}
}

func (id ID) handle(method string) *idHandle {
	if id.h == nil {
		panic("ui: " + method + " on a zero ID; declare it with ui.NewID")
	}
	return id.h
}

// Scope renders its block with a fresh generated value for each id:
//
//	@ui.Scope(EmailID) {
//		<label for={ EmailID.Get(ctx) }>Email</label>
//		<input id={ EmailID.Get(ctx) }/>
//	}
//
// Tokens come from the context's IDSource (WithIDs), else from the
// process-wide source. Scope returns an error for an id listed twice or a
// token that is not lower-case words joined by single hyphens.
func Scope(ids ...ID) templ.Component {
	hs := make([]*idHandle, len(ids))
	for i, id := range ids {
		hs[i] = id.handle("Scope")
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children, ctx := block(ctx)
		src := sourceOf(ctx)
		for i, h := range hs {
			for _, prev := range hs[:i] {
				if prev == h {
					return fmt.Errorf("ui: Scope lists the id %q twice", h.name)
				}
			}
			tok := src.Next()
			if !validToken(tok) {
				return fmt.Errorf("ui: the id source %T returned the token %q for the id %q; a token is lower-case letters and digits in words joined by single hyphens", src, tok, h.name)
			}
			ctx = context.WithValue(ctx, h, h.name+"--"+tok)
		}
		return children.Render(ctx, w)
	})
}

// Get returns the id's value in the nearest enclosing Scope of it. It
// panics outside one: a reference that silently got a fresh id is the bug
// scopes exist to prevent.
func (id ID) Get(ctx context.Context) string {
	h := id.handle("Get")
	if v, ok := ctx.Value(h).(string); ok {
		return v
	}
	panic(fmt.Sprintf("ui: %[1]s.Get outside ui.Scope(%[1]s): open @ui.Scope(...) around every element that uses the id", h.name))
}

// Part returns Get(ctx) followed by each part, escaped, after a hyphen:
// EmailID.Part(ctx, "hint") is "email--k3x9a2-1f-hint".
func (id ID) Part(ctx context.Context, parts ...string) string {
	id.handle("Part")
	return join(id.Get(ctx), parts)
}

// Key returns a keyed id built from data, the same on every render and
// needing no context: OrderRow.Key("1042") is "order-row-1042". With no
// parts it is the id's name.
func (id ID) Key(parts ...string) string {
	return join(id.handle("Key").name, parts)
}

func join(base string, parts []string) string {
	if len(parts) == 0 {
		return base
	}
	var b strings.Builder
	b.WriteString(base)
	for _, p := range parts {
		b.WriteByte('-')
		escape(&b, p)
	}
	return b.String()
}

// escape writes p with every byte outside [A-Za-z0-9] as "_" and two
// lower-case hex digits, and an empty p as "_". The result never contains
// "-", so parts stay apart, and different parts never give the same text.
func escape(b *strings.Builder, p string) {
	if p == "" {
		b.WriteByte('_')
		return
	}
	const hex = "0123456789abcdef"
	for i := 0; i < len(p); i++ {
		c := p[i]
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('_')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&0xf])
	}
}
