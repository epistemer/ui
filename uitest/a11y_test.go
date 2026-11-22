package uitest_test

import (
	"testing"

	"github.com/epistemer/ui/uitest"
)

// recorder captures failures, so a test can assert that a check fails.
type recorder struct {
	testing.TB
	failed bool
}

func (r *recorder) Helper()               {}
func (r *recorder) Errorf(string, ...any) { r.failed = true }
func (r *recorder) Fatalf(string, ...any) { r.failed = true }

func TestCheckNames(t *testing.T) {
	uitest.CheckNames(t, `<label for="q">Search</label><input id="q" type="search"><button type="button" aria-label="Close"><svg aria-hidden="true"></svg></button>`)
	r := &recorder{TB: t}
	uitest.CheckNames(r, `<button type="button"><svg aria-hidden="true"></svg></button>`)
	if !r.failed {
		t.Error("an icon button without a name passed")
	}
}

func TestCheckSelectors(t *testing.T) {
	css := `.ui-button { &[data-variant="outline"] { color: inherit; } } .shop-card { color: inherit; }`
	uitest.CheckSelectors(t, `<button class="ui-button" data-variant="outline" data-ui-hook="x">Go</button>`, css)
	uitest.CheckSelectors(t, `<div class="shop-card"></div>`, css, uitest.Prefix("shop-"))
	// Prefix replaces the default: ui's classes aren't checked against an
	// app's own CSS unless asked.
	uitest.CheckSelectors(t, `<div class="shop-card ui-unknown"></div>`, `.shop-card { color: red; }`, uitest.Prefix("shop-"))
	for _, html := range []string{
		`<button class="ui-button" data-variant="ghost">Go</button>`,
		`<span class="ui-badge"></span>`,
	} {
		r := &recorder{TB: t}
		uitest.CheckSelectors(r, html, css)
		if !r.failed {
			t.Errorf("%s passed", html)
		}
	}
	r := &recorder{TB: t}
	uitest.CheckSelectors(r, `<div class="shop-panel"></div>`, css, uitest.Prefix("shop-"))
	if !r.failed {
		t.Error("an app class the CSS doesn't select passed")
	}
}
