// Package elements is the D4 spike of ui/elements: the Button builder
// (Q2, Q3) and the attribute merge every component uses (design §5).
package elements

import (
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// joined are attributes whose values are space-separated lists: a
// caller's tokens are added to the component's, never replace them.
var joined = map[string]bool{"class": true, "aria-describedby": true, "aria-labelledby": true}

// Merge returns own with the caller's attributes applied (design §5):
//   - keys compare without regard to case, and render in lower case;
//   - class, aria-describedby and aria-labelledby are joined;
//   - a key in owned stays as the component set it, whatever the caller
//     passed, because the component's behaviour needs it;
//   - for any other key, the caller's value wins.
//
// HTML keeps the first of two copies of an attribute, so rendering both
// maps in one tag would let whichever came first win silently.
func Merge(own templ.Attributes, owned []string, caller templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	for k, v := range own {
		out[strings.ToLower(k)] = v
	}
	for k, v := range caller {
		k = strings.ToLower(k)
		switch {
		case slices.Contains(owned, k):
			continue
		case joined[k]:
			out[k] = joinTokens(out[k], v)
		default:
			out[k] = v
		}
	}
	return out
}

func joinTokens(a, b any) any {
	as, aok := a.(string)
	bs, bok := b.(string)
	switch {
	case a == nil:
		return b
	case !aok || !bok:
		return b
	}
	tokens := strings.Fields(as)
	for _, t := range strings.Fields(bs) {
		if !slices.Contains(tokens, t) {
			tokens = append(tokens, t)
		}
	}
	return strings.Join(tokens, " ")
}
