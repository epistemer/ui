package elements

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/links"
)

// ButtonSpec is a button being described (§5.1). Every method returns a
// changed copy, so a spec can be shared and refined:
//
//	var cancel = elements.Button().Variant(elements.Ghost).Tone(elements.Neutral)
//	@cancel.Href(back) { Cancel }
//
// It renders <button type="button">, type="submit" with Submit, or <a
// href> with Href. A disabled link renders without its href, with
// aria-disabled and role="link", because <a> can't be disabled; the ui
// script keeps it inert. Children are the content.
type ButtonSpec struct {
	variant  Variant
	size     Size
	tone     Tone
	href     templ.SafeURL
	submit   bool
	disabled bool
	label    i18n.Ter
	reason   i18n.Ter // Unavailable's
	attrs    templ.Attributes
}

// Button starts a button: solid, medium, in the brand's colours.
func Button() ButtonSpec { return ButtonSpec{} }

// Variant sets the look.
func (b ButtonSpec) Variant(v Variant) ButtonSpec { b.variant = v; return b }

// Size sets the size.
func (b ButtonSpec) Size(s Size) ButtonSpec { b.size = s; return b }

// Tone sets the colour role; the default is Brand, and Neutral suits a
// Cancel beside a brand action.
func (b ButtonSpec) Tone(t Tone) ButtonSpec { b.tone = t; return b }

// Href makes the button a link to l. No address (a nil link, an empty
// one, or one templ.URL refuses) leaves it a button.
func (b ButtonSpec) Href(l ui.Link) ButtonSpec { b.href = links.Href(l); return b }

// Submit makes it submit its form. A link can't submit; Render panics.
func (b ButtonSpec) Submit() ButtonSpec { b.submit = true; return b }

// Disabled disables it when d is true.
func (b ButtonSpec) Disabled(d bool) ButtonSpec { b.disabled = d; return b }

// Unavailable renders a button that can be read but not used: an action
// the record's state forbids (server-state-interface.md §5, S21). It is
// aria-disabled and focusable, does nothing with script or without (no
// href, type="button", no form, formaction or request attribute), and
// reason, in a Tooltip around it, describes it. Apart from Disabled, which
// stays native. A blank reason panics at render.
func (b ButtonSpec) Unavailable(reason i18n.Ter) ButtonSpec { b.reason = reason; return b }

// Label names an icon-only button (aria-label).
func (b ButtonSpec) Label(t i18n.Ter) ButtonSpec { b.label = t; return b }

// Attrs adds attributes to the <button> or <a>, merged as Merge does. The
// button keeps type, role, href, disabled, aria-disabled, its variant
// attributes, and aria-label when Label is set.
func (b ButtonSpec) Attrs(a templ.Attributes) ButtonSpec {
	b.attrs = Merge(b.attrs, nil, a)
	return b
}

var buttonOwned = []string{"type", "role", "href", "disabled", "aria-disabled", "data-variant", "data-size", "data-tone"}

func (b ButtonSpec) attributes(ctx context.Context) templ.Attributes {
	own := templ.Attributes{"class": "ui-button"}
	if b.variant != Solid {
		own["data-variant"] = string(b.variant)
	}
	if b.size != Medium {
		own["data-size"] = string(b.size)
	}
	if t := toneAttr(b.tone, Brand); t != "" {
		own["data-tone"] = t
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
	owned := buttonOwned
	if b.label != nil {
		own["aria-label"] = b.label.T(ctx)
		owned = append(owned[:len(owned):len(owned)], "aria-label")
	}
	return Merge(own, owned, b.attrs)
}

// Render renders the button with the caller's children as its content.
func (b ButtonSpec) Render(ctx context.Context, w io.Writer) error {
	if b.href != "" && b.submit && b.reason == nil {
		panic(fmt.Sprintf("elements: Button with href %q can't submit", b.href))
	}
	checkVariant("Button", b.variant)
	checkSize("Button", b.size)
	checkTone("Button", b.tone)
	if b.label != nil && strings.TrimSpace(b.label.T(ctx)) == "" {
		panic("elements: Button with a blank Label: an icon-only button needs a name")
	}
	if b.reason != nil {
		if strings.TrimSpace(b.reason.T(ctx)) == "" {
			panic("elements: Button with a blank Unavailable reason")
		}
		return unavailableTag(b.unavailableAttributes(ctx), b.reason).Render(ctx, w)
	}
	if b.href != "" {
		return linkTag(b.attributes(ctx)).Render(ctx, w)
	}
	return buttonTag(b.attributes(ctx)).Render(ctx, w)
}

// inert lists the attributes an unavailable control drops, so it does
// nothing without script or with htmx.
func inert(k string) bool {
	switch k {
	case "href", "form", "formaction", "formmethod", "formenctype", "formnovalidate", "formtarget", "data-ui-confirm", "commandfor", "command", "popovertarget":
		return true
	}
	return strings.HasPrefix(k, "hx-") || strings.HasPrefix(k, "data-hx-")
}

func (b ButtonSpec) unavailableAttributes(ctx context.Context) templ.Attributes {
	own := templ.Attributes{"class": "ui-button", "type": "button", "aria-disabled": "true"}
	if b.variant != Solid {
		own["data-variant"] = string(b.variant)
	}
	if b.size != Medium {
		own["data-size"] = string(b.size)
	}
	if t := toneAttr(b.tone, Brand); t != "" {
		own["data-tone"] = t
	}
	owned := buttonOwned
	if b.label != nil {
		own["aria-label"] = b.label.T(ctx)
		owned = append(owned[:len(owned):len(owned)], "aria-label")
	}
	rest := templ.Attributes{}
	for k, v := range b.attrs {
		if !inert(strings.ToLower(k)) {
			rest[k] = v
		}
	}
	return Merge(own, owned, rest)
}
