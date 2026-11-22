package elements

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
)

// AvatarSpec is a person's picture or initials (§5.3).
type AvatarSpec struct {
	name       i18n.Ter
	src        templ.SafeURL
	size       Size
	decorative bool
	attrs      templ.Attributes
}

// Avatar starts an avatar for the person named name: their picture with
// Src, else the first letters of the name's first two words.
func Avatar(name i18n.Ter) AvatarSpec { return AvatarSpec{name: name} }

// Src shows a picture.
func (a AvatarSpec) Src(u templ.SafeURL) AvatarSpec { a.src = u; return a }

// Size sets the size.
func (a AvatarSpec) Size(s Size) AvatarSpec { a.size = s; return a }

// Decorative hides it from assistive technology, for an avatar beside the
// name it shows.
func (a AvatarSpec) Decorative() AvatarSpec { a.decorative = true; return a }

// Attrs adds attributes to the <img> or <span>, merged as Merge does. The
// avatar keeps alt, role, aria-label, aria-hidden, src and data-size.
func (a AvatarSpec) Attrs(at templ.Attributes) AvatarSpec {
	a.attrs = Merge(a.attrs, nil, at)
	return a
}

var avatarOwned = []string{"alt", "role", "aria-label", "aria-hidden", "src", "data-size"}

// initials are the first letters of a name's first two words.
func initials(name string) string {
	var b strings.Builder
	for _, w := range strings.Fields(name) {
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(unicode.ToUpper(r))
				break
			}
		}
		if b.Len() > 0 && len([]rune(b.String())) == 2 {
			break
		}
	}
	return b.String()
}

func (a AvatarSpec) Render(ctx context.Context, w io.Writer) error {
	if a.name == nil {
		panic("elements: Avatar(nil): an avatar needs the person's name")
	}
	name := a.name.T(ctx)
	own := templ.Attributes{"class": "ui-avatar"}
	if a.size != Medium {
		own["data-size"] = string(a.size)
	}
	if a.src != "" {
		own["src"] = a.src
		own["alt"] = name
		if a.decorative {
			own["alt"] = ""
		}
		return imgTag(Merge(own, avatarOwned, a.attrs)).Render(ctx, w)
	}
	if a.decorative {
		own["aria-hidden"] = "true"
	} else {
		own["role"] = "img"
		own["aria-label"] = name
	}
	return textSpan(Merge(own, avatarOwned, a.attrs), initials(name)).Render(ctx, w)
}

// SpinnerSpec is a loading indicator (§5.3): role="status" with a
// visually hidden label. Its animation reads the duration tokens, so it
// stops under reduced motion. It fits htmx's .htmx-indicator.
type SpinnerSpec struct {
	size  Size
	label i18n.Ter
	attrs templ.Attributes
}

// Spinner starts a spinner labelled "Loading".
func Spinner() SpinnerSpec { return SpinnerSpec{} }

// Size sets the size.
func (s SpinnerSpec) Size(sz Size) SpinnerSpec { s.size = sz; return s }

// Label replaces "Loading" with what is loading.
func (s SpinnerSpec) Label(t i18n.Ter) SpinnerSpec { s.label = t; return s }

// Attrs adds attributes to the outer <span>, merged as Merge does; the
// spinner keeps role and data-size.
func (s SpinnerSpec) Attrs(a templ.Attributes) SpinnerSpec {
	s.attrs = Merge(s.attrs, nil, a)
	return s
}

func (s SpinnerSpec) Render(ctx context.Context, w io.Writer) error {
	own := templ.Attributes{"class": "ui-spinner", "role": "status"}
	if s.size != Medium {
		own["data-size"] = string(s.size)
	}
	label := i18n.Ter(spinnerLoading)
	if s.label != nil {
		label = s.label
	}
	return spinner(Merge(own, []string{"role", "data-size"}, s.attrs), label).Render(ctx, w)
}

// TooltipTrigger is the control a tooltip describes. Its fill receives
// aria-describedby naming the tooltip's text; it's required.
var TooltipTrigger = ui.NewSlotOf[templ.Attributes]()

var tooltipID = ui.NewID("tooltip")

// TooltipSpec is a tooltip being described (§5.3): a description shown on
// hover and on focus within, hidden with Esc by the ui script (WCAG
// 1.4.13). It only adds a description: the trigger needs a name of its
// own.
//
//	@elements.Tooltip(help) {
//		@elements.TooltipTrigger {
//			@elements.Button().Label(info).Attrs(elements.TooltipTrigger.Value(ctx)) { @icons.Info() }
//		}
//	}
type TooltipSpec struct {
	text  i18n.Ter
	attrs templ.Attributes
}

// Tooltip starts a tooltip showing text.
func Tooltip(text i18n.Ter) TooltipSpec { return TooltipSpec{text: text} }

// Attrs adds attributes to the outer <span>, merged as Merge does.
func (t TooltipSpec) Attrs(a templ.Attributes) TooltipSpec {
	t.attrs = Merge(t.attrs, nil, a)
	return t
}

func (t TooltipSpec) Render(ctx context.Context, w io.Writer) error {
	if t.text == nil {
		panic("elements: Tooltip(nil): a tooltip needs its text")
	}
	return ui.Slotted(tooltip(Merge(templ.Attributes{"class": "ui-tooltip"}, nil, t.attrs), t.text), TooltipTrigger).Render(ctx, w)
}

func requireTrigger(ctx context.Context) error {
	if !TooltipTrigger.Filled(ctx) {
		panic("elements: Tooltip without TooltipTrigger: fill it with the control the tooltip describes")
	}
	return nil
}

// Separator is a horizontal rule.
func Separator() templ.Component { return separator() }

// VerticalSeparator separates items in a row.
func VerticalSeparator() templ.Component { return verticalSeparator() }

// HeadingSpec is a heading being described (§5.3).
type HeadingSpec struct {
	level int
	size  Size
	attrs templ.Attributes
}

// Heading starts a heading of level 1 to 6. Children are its text.
func Heading(level int) HeadingSpec { return HeadingSpec{level: level} }

// Size sets the look apart from the level.
func (h HeadingSpec) Size(s Size) HeadingSpec { h.size = s; return h }

// Attrs adds attributes to the <hN>, merged as Merge does; the heading
// keeps data-size.
func (h HeadingSpec) Attrs(a templ.Attributes) HeadingSpec {
	h.attrs = Merge(h.attrs, nil, a)
	return h
}

func (h HeadingSpec) Render(ctx context.Context, w io.Writer) error {
	if h.level < 1 || h.level > 6 {
		panic(fmt.Sprintf("elements: Heading(%d): a heading's level is 1 to 6", h.level))
	}
	own := templ.Attributes{"class": "ui-heading"}
	if h.size != Medium {
		own["data-size"] = string(h.size)
	}
	return heading(fmt.Sprintf("h%d", h.level), Merge(own, []string{"data-size"}, h.attrs)).Render(ctx, w)
}
