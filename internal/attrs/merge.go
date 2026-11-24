// Package attrs provides attribute merging shared by elements and icons.
// Not API.
package attrs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/a-h/templ"
)

// joined are attributes whose values are space-separated lists: a
// caller's tokens are added to the component's, never replace them.
var joined = map[string]bool{
	"class":            true,
	"aria-describedby": true,
	"aria-labelledby":  true,
	"aria-controls":    true,
}

// Merge returns own with caller's attributes applied:
//   - keys compare without regard to case, and render in lower case;
//   - class, aria-describedby, aria-labelledby and aria-controls are
//     space-separated lists: caller's tokens are added after own's,
//     without repeats; either side must be a string;
//   - a key in owned stays as own set it (or absent), whatever caller
//     passed;
//   - for any other key, caller's value wins.
//
// pkg names the calling package in panics.
func Merge(pkg string, own templ.Attributes, owned []string, caller templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	put := func(k string, v any) {
		if joined[k] {
			out[k] = joinTokens(pkg, out[k], v)
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

func joinTokens(pkg string, a, b any) any {
	var tokens []string
	for _, side := range []any{a, b} {
		if side == nil {
			continue
		}
		s, ok := side.(string)
		if !ok {
			panic(fmt.Sprintf("%s: a class or id list must be a string, not %T", pkg, side))
		}
		for _, t := range strings.Fields(s) {
			if !slices.Contains(tokens, t) {
				tokens = append(tokens, t)
			}
		}
	}
	return strings.Join(tokens, " ")
}
