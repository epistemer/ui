package a11y

import (
	"slices"
	"testing"
)

func TestSelectedResolvesNestingAndCompounds(t *testing.T) {
	css := `
@layer ui.elements {
.ui-button {
	color: red;
	&:hover:not(:disabled, [aria-disabled="true"], .ui-ignored) { color: blue; }
	&[data-variant="outline"],
	&[data-variant='ghost'] {
		&:active { color: green; }
	}
	.ui-icon { inline-size: 1em; }
}
.ui-icon {
	&[data-mirror]:dir(rtl) { transform: scaleX(-1); }
}
.ui-badge > .ui-dot[data-tone=danger] { color: red; }
:root[data-color-scheme="dark"] { color-scheme: dark; }
/* .ui-commented { } */
}`
	got := Selected(css, []string{"ui-"})
	want := []string{
		".ui-badge",
		".ui-button",
		`.ui-button[data-variant="ghost"]`,
		`.ui-button[data-variant="outline"]`,
		".ui-dot",
		`.ui-dot[data-tone="danger"]`,
		".ui-icon",
		".ui-icon[data-mirror]",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Selected:\n got %q\nwant %q", got, want)
	}
}

func TestUsedPairsAttributesWithTheirClass(t *testing.T) {
	got, err := Used(`<span class="ui-badge" data-variant="outline"><svg class="ui-icon" data-mirror data-ui-hook="x"></svg></span><button class="other" data-variant="x"></button>`, []string{"ui-"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".ui-badge", `.ui-badge[data-variant="outline"]`, ".ui-icon", ".ui-icon[data-mirror]"}
	if !slices.Equal(got, want) {
		t.Errorf("Used:\n got %q\nwant %q", got, want)
	}
}

func TestNamesEdgeCases(t *testing.T) {
	bad := []string{
		`<label><select><option>One</option></select></label>`,
		`<label><textarea>Default</textarea></label>`,
		`<fieldset><legend> </legend><input type="checkbox" aria-label="x"></fieldset>`,
		`<input type="IMAGE" src="/pay.png">`,
		`<div role="checkbox" aria-checked="false"></div>`,
		`<div role="navigation"></div><nav></nav>`,
		`<template><label for="q">Search</label></template><input id="q" type="search">`,
	}
	for _, h := range bad {
		if probs, _ := Names(h); len(probs) == 0 {
			t.Errorf("%s passed", h)
		}
	}
	good := []string{
		`<label>Count <select><option>One</option></select></label>`,
		`<input type="image" src="/pay.png" alt="Pay">`,
		`<input type="SUBMIT">`,
		`<div role="checkbox" aria-checked="false" aria-label="Agree"></div>`,
		`<span id="h" hidden>Hidden name</span><button type="button" aria-labelledby="h"></button>`,
		`<nav aria-label="Main"></nav><div role="navigation" aria-label=" Footer "></div>`,
		`<html lang="en"><body><button type="button">Go</button></body></html>`,
	}
	for _, h := range good {
		if probs, _ := Names(h); len(probs) != 0 {
			t.Errorf("%s: %v", h, probs)
		}
	}
}
func TestNamesBasics(t *testing.T) {
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
		probs, err := Names(h)
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
		probs, err := Names(h)
		if err != nil {
			t.Fatal(err)
		}
		if len(probs) != 0 {
			t.Errorf("%s: %v", h, probs)
		}
	}
}

func TestSelectedEdges(t *testing.T) {
	css := `/* a comment with .ui-gone */
.ui-a { content: "/* not a comment"; }
.ui-b:is(.ui-c, .ui-d) { color: red; }
.ui-e:not(.ui-f):has(.ui-g) { color: red; }
.ui-h[data-variant~="x"], .ui-i[data-tone="y" i], .ui-j[data-size^="s"] { color: red; }`
	got := Selected(css, []string{"ui-", ""})
	for _, want := range []string{".ui-a", ".ui-b", ".ui-c", ".ui-d", ".ui-e", `.ui-h[data-variant="x"]`, `.ui-i[data-tone="y"]`, ".ui-j[data-size]"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s in %q", want, got)
		}
	}
	for _, bad := range []string{".ui-gone", ".ui-f", ".ui-g"} {
		if slices.Contains(got, bad) {
			t.Errorf("%s selected in %q", bad, got)
		}
	}
	if Selected(css, []string{""}) != nil {
		t.Error("an empty prefix covered every class")
	}
}

func TestNamesSources(t *testing.T) {
	good := []string{
		`<button type="button" title="Close"><svg aria-hidden="true"></svg></button>`,
		`<label for="b">Clear</label><button id="b" type="button"><svg aria-hidden="true"></svg></button>`,
		`<input type="submit">`,
		`<img src="/x.png" role="presentation">`,
	}
	for _, h := range good {
		if probs, _ := Names(h); len(probs) != 0 {
			t.Errorf("%s: %v", h, probs)
		}
	}
	for _, h := range []string{`<input type="submit" value="">`, `<input type="reset" value="  ">`} {
		if probs, _ := Names(h); len(probs) == 0 {
			t.Errorf("%s passed", h)
		}
	}
}
