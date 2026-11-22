package elements

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// joined are attributes whose values are space-separated lists: a
// caller's tokens are added to the component's, never replace them.
var joined = map[string]bool{"class": true, "aria-describedby": true, "aria-labelledby": true, "aria-controls": true}

// Merge returns own with the caller's attributes applied (design §5):
//   - keys compare without regard to case, and render in lower case;
//   - class, aria-describedby, aria-labelledby and aria-controls are
//     space-separated lists: the caller's tokens are added after the
//     component's, without repeats; either side must be a string;
//   - a key in owned stays as the component set it, whatever the caller
//     passed, because the component's behaviour needs it;
//   - for any other key, the caller's value wins.
//
// HTML keeps the first of two copies of an attribute, so rendering both
// maps in one tag would let whichever came first win silently.
func Merge(own templ.Attributes, owned []string, caller templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	put := func(k string, v any) {
		if joined[k] {
			out[k] = joinTokens(out[k], v)
		} else {
			out[k] = v
		}
	}
	for k, v := range own {
		put(strings.ToLower(k), v)
	}
	for k, v := range caller {
		k = strings.ToLower(k)
		if slices.ContainsFunc(owned, func(o string) bool { return strings.EqualFold(o, k) }) {
			continue
		}
		put(k, v)
	}
	return out
}

// joinTokens joins space-separated lists without repeats. Both sides must
// be strings: a caller's class given as anything else would otherwise
// replace the component's own class, and its styles with it.
func joinTokens(a, b any) any {
	var tokens []string
	for _, side := range []any{a, b} {
		if side == nil {
			continue
		}
		s, ok := side.(string)
		if !ok {
			panic(fmt.Sprintf("elements: a class or id list must be a string, not %T", side))
		}
		for _, t := range strings.Fields(s) {
			if !slices.Contains(tokens, t) {
				tokens = append(tokens, t)
			}
		}
	}
	return strings.Join(tokens, " ")
}
