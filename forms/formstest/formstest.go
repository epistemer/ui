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

// AssertLabelled fails t once for every field, named group and list, at
// any depth, whose label is still the default built from its name. A
// list's members are checked once, on its template row, whether or not it
// has items. The root and list items have no label of their own. Hidden
// fields (KindHidden) aren't checked: nobody reads their label (A9).
func AssertLabelled(t testing.TB, f *forms.Form) {
	t.Helper()
	if f == nil {
		t.Fatal("formstest: AssertLabelled(nil)")
	}
	var walk func(c forms.Control)
	walk = func(c forms.Control) {
		t.Helper()
		if c.Kind() == forms.KindHidden {
			return
		}
		if c.Path() != "" && !hook.LabelSet(c) {
			t.Errorf("formstest: %s has no label of its own", c.Path())
		}
		if c.Kind() == forms.KindList {
			if tpl, ok := c.Template(); ok {
				for child := range tpl.Children() {
					walk(child)
				}
			}
			return
		}
		for child := range c.Children() {
			walk(child)
		}
	}
	walk(f.Control())
}
