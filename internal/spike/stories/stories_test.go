package stories

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/epistemer/ui/internal/spike/a11y"
	"github.com/epistemer/ui/internal/themehook"
	_ "github.com/epistemer/ui/theme" // sets the hook
	"github.com/epistemer/ui/uitest"
)

// TestStories runs every check on every variant of every story (design
// §8): a golden file, id references, accessible names, and selectors that
// exist in ui's CSS. Then it checks the other way: every component rule in
// ui's CSS is used by some story.
func TestStories(t *testing.T) {
	selected := a11y.Selected(string(themehook.Stylesheet()))
	used := map[string]bool{}
	for _, s := range All() {
		for _, v := range s.Variants {
			name := strings.ReplaceAll(s.Name, "/", "-") + "-" + v.Name
			t.Run(name, func(t *testing.T) {
				html := uitest.Render(t, v.Render(context.Background()))
				uitest.Golden(t, name, html)
				uitest.CheckIDs(t, html)
				probs, err := a11y.Names(html)
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range probs {
					t.Error(p)
				}
				sel, err := a11y.Used(html)
				if err != nil {
					t.Fatal(err)
				}
				for _, s := range sel {
					used[s] = true
					if !slices.Contains(selected, s) {
						t.Errorf("markup uses %s, which ui's CSS never selects", s)
					}
				}
			})
		}
	}
	for _, s := range selected {
		if !used[s] && !notYetStoried[s] {
			t.Errorf("ui's CSS selects %s, which no story renders", s)
		}
	}
}

// notYetStoried lists selectors that exist before their stories: base
// classes used by components still to come in the slice.
var notYetStoried = map[string]bool{}

func TestNamesCheckCatchesMistakes(t *testing.T) {
	bad := []string{
		`<button type="button"><svg aria-hidden="true"></svg></button>`,
		`<a href="/x"></a>`,
		`<input type="text" name="q" placeholder="Search">`,
		`<img src="/x.png">`,
		`<dialog open><p>Hi</p></dialog>`,
		`<fieldset><input type="checkbox" aria-label="x"></fieldset>`,
		`<nav></nav><nav></nav>`,
		`<button type="button" aria-labelledby="missing"></button>`,
	}
	for _, h := range bad {
		probs, err := a11y.Names(h)
		if err != nil {
			t.Fatal(err)
		}
		if len(probs) == 0 {
			t.Errorf("%s passed the names check", h)
		}
	}
	good := []string{
		`<button type="button" aria-label="Close"><svg aria-hidden="true"></svg></button>`,
		`<label for="q">Search</label><input id="q" type="search">`,
		`<label>Email <input type="email"></label>`,
		`<img src="/x.png" alt="">`,
		`<input type="hidden" name="csrf">`,
		`<dialog aria-labelledby="t"><h2 id="t">Delete?</h2></dialog>`,
		`<fieldset><legend>Size</legend><label><input type="radio" name="s"> S</label></fieldset>`,
		`<nav aria-label="Main"></nav><nav aria-label="Breadcrumb"></nav>`,
		`<div hidden><button type="button"></button></div>`,
		`<template><button type="button"></button></template>`,
	}
	for _, h := range good {
		probs, err := a11y.Names(h)
		if err != nil {
			t.Fatal(err)
		}
		if len(probs) != 0 {
			t.Errorf("%s: %v", h, probs)
		}
	}
}
