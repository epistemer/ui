package uitest

import (
	"slices"
	"testing"

	"github.com/epistemer/ui/internal/a11y"
)

// CheckNames fails for each element in rawHTML that needs an accessible
// name and has none (amendment A4): buttons, links with href, form
// controls other than hidden inputs, images without alt (an empty alt
// passes), dialogs, and elements with a control role. A name comes from
// text, aria-label, aria-labelledby (its targets count even when hidden),
// a <label>, alt or title. It also fails a <fieldset> without a <legend>
// that says something, and navigation landmarks that share a name or have
// none. Hidden and aria-hidden subtrees and <template> contents are
// skipped. rawHTML is a fragment or a whole page, as Golden reads it.
func CheckNames(t testing.TB, rawHTML string) {
	t.Helper()
	probs, err := a11y.Names(rawHTML)
	if err != nil {
		t.Fatalf("uitest: parse: %v", err)
		return
	}
	for _, p := range probs {
		t.Errorf("uitest: %s", p)
	}
}

// SelectorOption changes what CheckSelectors covers.
type SelectorOption func(*selectorOptions)

type selectorOptions struct{ prefixes []string }

// Prefix adds class prefixes CheckSelectors covers; "ui-" is always
// covered.
func Prefix(prefixes ...string) SelectorOption {
	return func(o *selectorOptions) { o.prefixes = append(o.prefixes, prefixes...) }
}

// CheckSelectors fails for each compound selector the markup asks of css
// that css never selects (amendment A4): every class of a covered prefix,
// alone, with each variant attribute on the same element by value
// (.ui-button[data-variant="outline"]), and with its other data-*
// attributes by presence (.ui-icon[data-mirror]). data-ui-* hooks for the
// ui script are skipped. css is read with nesting resolved.
func CheckSelectors(t testing.TB, rawHTML, css string, opts ...SelectorOption) {
	t.Helper()
	o := selectorOptions{prefixes: []string{"ui-"}}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	used, err := a11y.Used(rawHTML, o.prefixes)
	if err != nil {
		t.Fatalf("uitest: parse: %v", err)
		return
	}
	selected := a11y.Selected(css, o.prefixes)
	for _, u := range used {
		if !slices.Contains(selected, u) {
			t.Errorf("uitest: the markup asks for %s, which the CSS never selects", u)
		}
	}
}
