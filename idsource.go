package ui

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
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
	b := make([]byte, 6)
	rand.Read(b) // never fails (crypto/rand, Go 1.24 and later)
	for i, c := range b {
		b[i] = digits[c%36]
	}
	return string(b)
}

var tokenSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// validToken reports whether t can follow "<name>--" in an id: words of
// lower-case letters and digits joined by single hyphens.
func validToken(t string) bool { return tokenSyntax.MatchString(t) }
