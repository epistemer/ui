package layout

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/i18ntest"

	"github.com/epistemer/ui"
)

func renderCtx(t *testing.T, ctx context.Context, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	if err := c.Render(ui.WithIDs(ctx, ui.Sequence("t")), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	return renderCtx(t, context.Background(), c)
}

func panicsWith(t *testing.T, what, want string, c templ.Component) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), want) {
			t.Errorf("%s: got panic %v, want one with %q", what, r, want)
		}
	}()
	c.Render(ui.WithIDs(context.Background(), ui.Sequence("t")), &strings.Builder{})
}

func contains(t *testing.T, html string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(html, w) {
			t.Errorf("missing %s in:\n%s", w, html)
		}
	}
}

func TestDocument(t *testing.T) {
	head := templ.Raw(`<link rel="stylesheet" href="/a.css">`)
	more := templ.Raw(`<script src="/b.js"></script>`)
	d := Document(i18n.Static("Orders")).Head(head).Head(more).ColorScheme(DarkScheme).
		BodyAttrs(templ.Attributes{"hx-boost": "true"}).
		Attrs(templ.Attributes{"lang": "fr", "dir": "rtl", "data-color-scheme": "light", "class": "app"})
	html := render(t, d)
	contains(t, html,
		"<!doctype html>",
		`<html class="app" data-color-scheme="dark" dir="ltr" lang="en">`,
		`<meta charset="utf-8">`,
		`<meta name="viewport" content="width=device-width, initial-scale=1">`,
		`<title>Orders</title><link rel="stylesheet" href="/a.css"><script src="/b.js"></script>`,
		`<body hx-boost="true">`,
	)
	if html := render(t, Document(i18n.Static("x"))); strings.Contains(html, "data-color-scheme") {
		t.Errorf("the system scheme renders the attribute: %s", html)
	}
}

func TestDocumentLocale(t *testing.T) {
	tr := i18ntest.New(t, i18n.Locale{Code: "en", NativeName: "English"})
	if err := tr.CreateLocale(i18n.Locale{Code: "ar-EG", NativeName: "العربية", RTL: true}); err != nil {
		t.Fatal(err)
	}
	html := renderCtx(t, i18ntest.Context(t, tr, "ar-EG"), Document(i18n.Static("x")))
	contains(t, html, `dir="rtl" lang="ar-EG"`)
	html = renderCtx(t, i18ntest.Context(t, tr, "en"), Document(i18n.Static("x")))
	contains(t, html, `dir="ltr" lang="en"`)
}

func TestColorSchemeOf(t *testing.T) {
	for value, want := range map[string]ColorScheme{"light": LightScheme, "dark": DarkScheme, "sepia": SystemScheme, "": SystemScheme, "DARK": SystemScheme} {
		r := httptest.NewRequest("GET", "/", nil)
		if value != "" {
			r.AddCookie(&http.Cookie{Name: "ui-color-scheme", Value: value})
		}
		if got := ColorSchemeOf(r); got != want {
			t.Errorf("cookie %q: got %q, want %q", value, got, want)
		}
	}
}

func TestColorSchemeMenu(t *testing.T) {
	html := render(t, ColorSchemeMenu(DarkScheme))
	contains(t, html,
		`aria-checked="false" class="ui-menu-item ui-scheme-item" data-ui-color-scheme="system"`,
		`aria-checked="true" class="ui-menu-item ui-scheme-item" data-ui-color-scheme="dark"`,
		`role="menuitemradio"`, `aria-label="Theme"`)
}

// with renders spec with children as its block, as templ does.
func with(spec, children templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return spec.Render(templ.WithChildren(ctx, children), w)
	})
}

func TestPageHeader(t *testing.T) {
	blank := templ.Raw("  \n ")
	if html := render(t, with(PageHeader(i18n.Static("Settings")).Description(i18n.Static(" ")), blank)); strings.Contains(html, "ui-page-actions") || strings.Contains(html, "ui-page-description") {
		t.Errorf("blank actions or description render their box: %s", html)
	}
	html := render(t, with(PageHeader(i18n.Static("Orders")).Breadcrumbs(
		Crumb{Href: ui.URL("/"), Text: i18n.Static("Home")},
		Crumb{Href: ui.URL("/orders"), Text: i18n.Static("Orders")},
	), templ.Raw("<button>Export</button>")))
	for _, last := range []ui.Link{nil, ui.URL(""), ui.URL("javascript:x")} {
		if html := render(t, PageHeader(i18n.Static("Orders")).Breadcrumbs(Crumb{Href: ui.URL("/"), Text: i18n.Static("Home")}, Crumb{Href: last, Text: i18n.Static("Orders")})); !strings.Contains(html, `<span aria-current="page">Orders</span>`) {
			t.Errorf("a last crumb with no address (%v) isn't the current page as text: %s", last, html)
		}
	}
	contains(t, html,
		`<a href="/">Home</a>`,
		`<a href="/orders" aria-current="page">Orders</a>`,
		`<div class="ui-page-actions"><button>Export</button></div>`)
	if strings.Count(html, "data-mirror") != 1 {
		t.Errorf("want one separator between two crumbs: %s", html)
	}
}

func TestOwned(t *testing.T) {
	html := render(t, with(NavItem(ui.URL("/a")).Current(true).Attrs(templ.Attributes{"href": "/b", "aria-current": "false", "class": "mine"}), templ.Raw("A")))
	contains(t, html, `href="/a"`, `aria-current="page"`, `class="ui-nav-item mine"`)
	html = render(t, with(NavSection(i18n.Static("Catalogue")).Attrs(templ.Attributes{"role": "list"}), templ.Raw("")))
	contains(t, html, `role="group"`, `aria-labelledby="nav-section--t-1-label"`, `id="nav-section--t-1-label"`)
}

func TestPanics(t *testing.T) {
	panicsWith(t, "a document without a title", "no title", Document(nil))
	panicsWith(t, "a document with a blank title", "no title", Document(i18n.Static(" ")))
	panicsWith(t, "a document in sepia", "colour scheme", Document(i18n.Static("x")).ColorScheme("sepia"))
	panicsWith(t, "a document with a nil head", "nil Head", Document(i18n.Static("x")).Head(nil))
	panicsWith(t, "an app without main", "without AppMain", App())
	panicsWith(t, "auth without a title", "no title", Auth(nil))
	panicsWith(t, "auth without a body", "without AuthBody", Auth(i18n.Static("Sign in")))
	panicsWith(t, "a page header without a title", "no title", PageHeader(i18n.Static("")))
	panicsWith(t, "a crumb without text", "crumb", PageHeader(i18n.Static("x")).Breadcrumbs(Crumb{Href: ui.URL("/")}))
	panicsWith(t, "a nav item without an address", "no address", NavItem(nil))
	panicsWith(t, "a nav item with a refused address", "no address", NavItem(ui.URL("javascript:alert(1)")))
	panicsWith(t, "a nav item with an empty address", "no address", NavItem(ui.URL("")))
	panicsWith(t, "a crumb above the page with a refused address", "no address", PageHeader(i18n.Static("x")).Breadcrumbs(Crumb{Href: ui.URL("javascript:x"), Text: i18n.Static("Home")}, Crumb{Text: i18n.Static("Here")}))
	panicsWith(t, "a nav section without a label", "no label", NavSection(i18n.Static(" ")))
	panicsWith(t, "a scheme menu in sepia", "colour scheme", ColorSchemeMenu("sepia"))
	panicsWith(t, "a crumb above the page without an address", "no address", PageHeader(i18n.Static("x")).Breadcrumbs(Crumb{Text: i18n.Static("Home")}, Crumb{Text: i18n.Static("Here")}))
}
