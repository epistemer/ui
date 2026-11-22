// Package catalogtest runs every check on every variant of a catalogue's
// stories (visual-interface.md §9.1): a golden file, id references,
// accessible names, and the selectors the markup asks of ui's CSS and the
// app's own.
//
//	func TestStories(t *testing.T) {
//		catalogtest.Check(t, stories.All(), catalogtest.CSS(appCSS), catalogtest.Prefix("shop-"))
//	}
package catalogtest

import (
	"context"
	"strings"
	"testing"

	"github.com/epistemer/ui/catalog"
	"github.com/epistemer/ui/internal/a11y"
	"github.com/epistemer/ui/internal/themehook"
	_ "github.com/epistemer/ui/theme" // sets themehook.Stylesheet
	"github.com/epistemer/ui/uitest"
)

// Option changes what Check covers.
type Option func(*options)

type options struct {
	css      []string
	prefixes []string
	external []string
}

// CSS adds the app's own stylesheets, checked with ui's.
func CSS(stylesheets ...string) Option {
	return func(o *options) { o.css = append(o.css, stylesheets...) }
}

// Prefix adds class prefixes the selector checks cover; "ui-" always is.
func Prefix(prefixes ...string) Option {
	return func(o *options) { o.prefixes = append(o.prefixes, prefixes...) }
}

// External lists ids a variant may reference outside itself.
func External(ids ...string) Option {
	return func(o *options) { o.external = append(o.external, ids...) }
}

// Check runs, as a subtest per variant named <package>-<name>-<variant>:
// uitest.Render, uitest.Golden at testdata/<package>-<name>-<variant>.golden.html,
// uitest.CheckIDs, uitest.CheckNames and uitest.CheckSelectors against ui's
// CSS and the CSS options. Then it fails for each rule of the CSS options
// that selects a covered class no story renders.
func Check(t *testing.T, stories []catalog.Story, opts ...Option) {
	t.Helper()
	o := options{prefixes: []string{"ui-"}}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	uiCSS := string(themehook.Stylesheet())
	all := uiCSS + "\n" + strings.Join(o.css, "\n")
	used := map[string]bool{}
	for _, s := range stories {
		for _, v := range s.Variants {
			name := strings.ReplaceAll(s.Name, "/", "-") + "-" + v.Name
			t.Run(name, func(t *testing.T) {
				html := uitest.Render(t, v.Render(context.Background()))
				uitest.Golden(t, name, html)
				uitest.CheckIDs(t, html, uitest.External(o.external...))
				uitest.CheckNames(t, html)
				uitest.CheckSelectors(t, html, all, uitest.Prefix(o.prefixes...))
				u, err := a11y.Used(html, o.prefixes)
				if err != nil {
					t.Fatal(err)
				}
				for _, s := range u {
					used[s] = true
				}
			})
		}
	}
	reverse := func(css, whose string) {
		for _, s := range a11y.Selected(css, o.prefixes) {
			if !used[s] {
				t.Errorf("catalogtest: %s selects %s, which no story renders", whose, s)
			}
		}
	}
	for _, css := range o.css {
		reverse(css, "the app's CSS")
	}
	if themehook.ReverseCheck {
		reverse(uiCSS, "ui's CSS")
	}
}
