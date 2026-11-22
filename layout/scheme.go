package layout

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

// ColorSchemeMenu is the colour scheme switch for the app bar (§8.2,
// V13): a menu labelled "Theme" with three items, system, light and dark,
// each a menuitemradio checked from current. Pass ColorSchemeOf(r). The ui
// script sets <html data-color-scheme> and the cookie ui-color-scheme when
// someone picks one, and keeps the items' aria-checked in step.
func ColorSchemeMenu(current ColorScheme) templ.Component {
	switch current {
	case SystemScheme, LightScheme, DarkScheme:
	default:
		panic("layout: ColorSchemeMenu with colour scheme " + string(current) + ": use SystemScheme, LightScheme or DarkScheme")
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return schemeMenu(current).Render(ctx, w)
	})
}

type schemeChoice struct {
	value string // data-ui-color-scheme
	text  i18n.Ter
	is    ColorScheme
}

var schemeChoices = []schemeChoice{
	{"system", schemeSystem, SystemScheme},
	{"light", schemeLight, LightScheme},
	{"dark", schemeDark, DarkScheme},
}

func schemeItemAttrs(c schemeChoice, current ColorScheme) templ.Attributes {
	checked := "false"
	if c.is == current {
		checked = "true"
	}
	return templ.Attributes{
		"type":                 "button",
		"class":                "ui-menu-item ui-scheme-item",
		"role":                 "menuitemradio",
		"tabindex":             "-1",
		"aria-checked":         checked,
		"data-ui-color-scheme": c.value,
	}
}
