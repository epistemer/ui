package elements

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// Variant is a button's look.
type Variant string

const (
	Solid   Variant = "" // the default: the accent's fill
	Outline Variant = "outline"
	Ghost   Variant = "ghost"
)

// Size is a component's size.
type Size string

const (
	Small  Size = "s"
	Medium Size = "" // the default
	Large  Size = "l"
)

// Tone is the accent a component takes its colours from.
type Tone string

const (
	Brand   Tone = "" // the default
	Danger  Tone = "danger"
	Warning Tone = "warning"
	Success Tone = "success"
	Info    Tone = "info"
)

// ButtonSpec is a button being described. Every method returns a changed
// copy, so a spec can be shared and refined:
//
//	var cancel = elements.Button().Variant(elements.Ghost)
//	@cancel.Href(back) { Cancel }
//
// It renders <button type="button">, or <a> when it has an href. A
// disabled link renders without its href, with aria-disabled and
// role="link", because <a> can't be disabled (design §5).
type ButtonSpec struct {
	variant  Variant
	size     Size
	tone     Tone
	href     templ.SafeURL
	submit   bool
	disabled bool
	attrs    templ.Attributes
}

// Button starts a button: solid, medium, in the brand's colours.
func Button() ButtonSpec { return ButtonSpec{} }

// Variant sets the look.
func (b ButtonSpec) Variant(v Variant) ButtonSpec { b.variant = v; return b }

// Size sets the size.
func (b ButtonSpec) Size(s Size) ButtonSpec { b.size = s; return b }

// Tone sets the accent.
func (b ButtonSpec) Tone(t Tone) ButtonSpec { b.tone = t; return b }

// Href makes the button a link.
func (b ButtonSpec) Href(u templ.SafeURL) ButtonSpec { b.href = u; return b }

// Submit makes it submit its form. A link can't submit; Render panics.
func (b ButtonSpec) Submit() ButtonSpec { b.submit = true; return b }

// Disabled disables it when d is true.
func (b ButtonSpec) Disabled(d bool) ButtonSpec { b.disabled = d; return b }

// Attrs adds attributes, merged as Merge does: the button keeps type,
// role, href, disabled, aria-disabled and its variant attributes, which
// only the builder's methods set.
func (b ButtonSpec) Attrs(a templ.Attributes) ButtonSpec {
	b.attrs = Merge(b.attrs, nil, a)
	return b
}

var buttonOwned = []string{"type", "role", "href", "disabled", "aria-disabled", "data-variant", "data-size", "data-tone"}

func (b ButtonSpec) attributes() templ.Attributes {
	own := templ.Attributes{"class": "ui-button"}
	if b.variant != Solid {
		own["data-variant"] = string(b.variant)
	}
	if b.size != Medium {
		own["data-size"] = string(b.size)
	}
	if b.tone != Brand {
		own["data-tone"] = string(b.tone)
	}
	switch {
	case b.href != "" && b.disabled:
		own["role"] = "link"
		own["aria-disabled"] = "true"
	case b.href != "":
		own["href"] = b.href
	case b.submit:
		own["type"] = "submit"
		own["disabled"] = b.disabled
	default:
		own["type"] = "button"
		own["disabled"] = b.disabled
	}
	return Merge(own, buttonOwned, b.attrs)
}

// Render renders the button with the caller's children as its content.
func (b ButtonSpec) Render(ctx context.Context, w io.Writer) error {
	if b.href != "" && b.submit {
		panic(fmt.Sprintf("elements: a button with href %q can't submit", b.href))
	}
	if b.href != "" {
		return linkTag(b.attributes()).Render(ctx, w)
	}
	return buttonTag(b.attributes()).Render(ctx, w)
}
