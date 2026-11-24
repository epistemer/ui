package elements

import (
	"github.com/a-h/templ"

	"github.com/epistemer/ui/internal/attrs"
)

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
	return attrs.Merge("elements", own, owned, caller)
}
