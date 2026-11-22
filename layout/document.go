package layout

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"
)

// ColorScheme is the colour scheme a page renders in.
type ColorScheme string

const (
	SystemScheme ColorScheme = ""
	LightScheme  ColorScheme = "light"
	DarkScheme   ColorScheme = "dark"
)

// schemeCookie is the cookie the ui script writes when someone picks a
// scheme in ColorSchemeMenu (§10.2).
const schemeCookie = "ui-color-scheme"

// ColorSchemeOf reads the scheme someone picked, from the cookie
// ui-color-scheme; anything but light or dark is the system's. Render it
// with Document's ColorScheme, so no page flashes the wrong scheme.
func ColorSchemeOf(r *http.Request) ColorScheme {
	c, err := r.Cookie(schemeCookie)
	if err != nil {
		return SystemScheme
	}
	switch s := ColorScheme(c.Value); s {
	case LightScheme, DarkScheme:
		return s
	}
	return SystemScheme
}

// DocumentSpec is an HTML document being described (§8.1): <!doctype
// html>, <html lang dir>, a <head> and a <body> holding the children.
type DocumentSpec struct {
	title  i18n.Ter
	head   []templ.Component
	scheme ColorScheme
	body   templ.Attributes
	attrs  templ.Attributes
}

// Document starts a document titled title. lang and dir come from the
// request's locale (i18n.Code): its code, and rtl when the locale is
// written right to left. Without a locale in the context, lang is "en",
// the language of ui's own text.
func Document(title i18n.Ter) DocumentSpec { return DocumentSpec{title: title} }

// Head adds components to the <head>, after charset, viewport and title,
// in order: an app passes its theme's Head and anything of its own.
// Repeated calls append.
func (d DocumentSpec) Head(c ...templ.Component) DocumentSpec {
	d.head = append(slices.Clip(d.head), c...)
	return d
}

// ColorScheme sets <html data-color-scheme>; SystemScheme leaves it out,
// so the system's scheme applies.
func (d DocumentSpec) ColorScheme(s ColorScheme) DocumentSpec { d.scheme = s; return d }

// BodyAttrs adds attributes to <body>, merged as elements.Merge does: an
// htmx app puts hx-boost or hx-headers here.
func (d DocumentSpec) BodyAttrs(a templ.Attributes) DocumentSpec {
	d.body = elements.Merge(d.body, nil, a)
	return d
}

// Attrs adds attributes to <html>, merged as elements.Merge does. The
// document keeps lang, dir and data-color-scheme.
func (d DocumentSpec) Attrs(a templ.Attributes) DocumentSpec {
	d.attrs = elements.Merge(d.attrs, nil, a)
	return d
}

var documentOwned = []string{"lang", "dir", "data-color-scheme"}

func (d DocumentSpec) Render(ctx context.Context, w io.Writer) error {
	if d.title == nil || strings.TrimSpace(d.title.T(ctx)) == "" {
		panic("layout: Document with no title: a page needs its title")
	}
	switch d.scheme {
	case SystemScheme, LightScheme, DarkScheme:
	default:
		panic("layout: Document with colour scheme " + string(d.scheme) + ": use SystemScheme, LightScheme or DarkScheme")
	}
	for _, h := range d.head {
		if h == nil {
			panic("layout: Document with a nil Head component")
		}
	}
	loc := i18n.Code(ctx)
	own := templ.Attributes{"lang": "en", "dir": "ltr"}
	if loc.Code != "" {
		own["lang"] = loc.Code
	}
	if loc.RTL {
		own["dir"] = "rtl"
	}
	if d.scheme != SystemScheme {
		own["data-color-scheme"] = string(d.scheme)
	}
	return document(d, elements.Merge(own, documentOwned, d.attrs)).Render(ctx, w)
}
