package ui

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"sync/atomic"
)

// IDSource gives the tokens of generated ids. Each token must be unique
// among those of its source, and be lower-case letters and digits in words
// joined by single hyphens. A source is used concurrently.
type IDSource interface{ Next() string }

// WithIDs returns a context whose Scopes draw tokens from s. Tests use it
// with Sequence for byte-stable output; a handler may give each separately
// rendered fragment its own prefix. It panics on a nil s.
func WithIDs(ctx context.Context, s IDSource) context.Context {
	if s == nil {
		panic("ui: WithIDs(ctx, nil)")
	}
	return context.WithValue(ctx, idSourceKey{}, s)
}

type idSourceKey struct{}

func sourceOf(ctx context.Context) IDSource {
	if s, ok := ctx.Value(idSourceKey{}).(IDSource); ok {
		return s
	}
	return processIDs
}

// Sequence returns a source of prefix-1, prefix-2, …, safe for concurrent
// use. It panics unless prefix is a valid id name (NewID).
func Sequence(prefix string) IDSource {
	if !idName.MatchString(prefix) {
		panic(fmt.Sprintf("ui: Sequence(%q): the prefix must be a valid id name", prefix))
	}
	return &sequence{prefix: prefix + "-", base: 10}
}

type sequence struct {
	prefix string
	base   int // 10 for Sequence, 36 for the process source
	n      atomic.Uint64
}

func (s *sequence) Next() string {
	return s.prefix + strconv.FormatUint(s.n.Add(1), s.base)
}

// processIDs is the default source: one per process, so fragments rendered
// by separate requests never share a generated id. Its tokens are a random
// prefix of 6 base-36 characters fixed at start-up and a counter in base
// 36, such as "k3x9a2-1f".
var processIDs IDSource = &sequence{prefix: randomPrefix() + "-", base: 36}

func randomPrefix() string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	out := make([]byte, 0, 6)
	var buf [16]byte
	for len(out) < 6 {
		rand.Read(buf[:]) // never fails (crypto/rand, Go 1.24 and later)
		for _, c := range buf {
			if c < 252 && len(out) < 6 { // 252 = 7×36: no modulo bias
				out = append(out, digits[c%36])
			}
		}
	}
	return string(out)
}

// validToken reports whether t can follow "<name>--" in an id: words of
// lower-case letters and digits joined by single hyphens.
func validToken(t string) bool {
	if t == "" || t[0] == '-' || t[len(t)-1] == '-' {
		return false
	}
	for i := 0; i < len(t); i++ {
		switch c := t[i]; {
		case 'a' <= c && c <= 'z', '0' <= c && c <= '9':
		case c == '-' && t[i-1] != '-':
		default:
			return false
		}
	}
	return true
}
