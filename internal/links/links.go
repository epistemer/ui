// Package links turns a ui.Link into the address the visual packages
// render (visual-interface.md §2.7).
package links

import (
	"reflect"
	"strings"

	"github.com/a-h/templ"

	"github.com/epistemer/ui"
)

// Href is l's address, sanitised by templ.URL, or "" when l is no address:
// nil (a typed nil pointer too, whose URL would panic), an empty URL, or one
// templ.URL refuses, such as javascript: or data:.
func Href(l ui.Link) templ.SafeURL {
	if l == nil {
		return ""
	}
	if v := reflect.ValueOf(l); v.Kind() == reflect.Pointer && v.IsNil() {
		return ""
	}
	s := strings.TrimSpace(l.URL())
	if s == "" {
		return ""
	}
	u := templ.URL(s)
	if u == templ.FailedSanitizationURL {
		return ""
	}
	return u
}
