// Package formstest helps test forms without HTTP.
package formstest

import (
	"context"
	"net/url"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/internal/hook"
)

// Submit binds values, a query string such as "name=Gold&type=fixed", and
// validates. A malformed query string fails t.
func Submit(t testing.TB, ctx context.Context, f *forms.Form, values string) bool {
	t.Helper()
	v, err := url.ParseQuery(values)
	if err != nil {
		t.Fatalf("formstest: Submit: %v", err)
	}
	f.Bind(forms.FromValues(v))
	return f.Validate(ctx)
}

// Errors lists the form's errors in Err's order, one "<path>: <key>" per
// failing node: <path> is "<form>" for the form's own error, <key> the
// i18n key of the error people see, or "-" when none is public.
func Errors(f *forms.Form) []string {
	var out []string
	for fe := range f.Errors() {
		path := fe.Path
		if path == "" {
			path = "<form>"
		}
		key := "-"
		for p := range errors.All(fe.Err) {
			key = p.K()
			break
		}
		out = append(out, path+": "+key)
	}
	return out
}

// AssertLabelled fails t for every field, named group and list, at any
// depth, whose label is still the default built from its name. The root
// and list items have no label of their own and are skipped.
func AssertLabelled(t testing.TB, f *forms.Form) {
	t.Helper()
	var walk func(c forms.Control, item bool)
	walk = func(c forms.Control, item bool) {
		if c.Path() != "" && !item && !hook.LabelSet(c) {
			t.Errorf("formstest: %s has no label of its own", c.Path())
		}
		for child := range c.Children() {
			walk(child, c.Kind() == forms.KindList)
		}
	}
	walk(f.Control(), false)
}
