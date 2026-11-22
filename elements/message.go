package elements

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/icons"
)

// BadgeSpec is a badge being described: a short label in its tone's
// subtle colours, neutral by default (§5.2). Children are its text;
// Attrs go to the <span>.
type BadgeSpec struct {
	tone  Tone
	attrs templ.Attributes
}

// Badge starts a neutral badge.
func Badge() BadgeSpec { return BadgeSpec{} }

// Tone sets the colour role.
func (b BadgeSpec) Tone(t Tone) BadgeSpec { b.tone = t; return b }

// Attrs adds attributes to the <span>, merged as Merge does; the badge
// keeps data-tone.
func (b BadgeSpec) Attrs(a templ.Attributes) BadgeSpec { b.attrs = Merge(b.attrs, nil, a); return b }

func (b BadgeSpec) Render(ctx context.Context, w io.Writer) error {
	own := templ.Attributes{"class": "ui-badge"}
	if t := toneAttr(b.tone, Neutral); t != "" {
		own["data-tone"] = t
	}
	return spanTag(Merge(own, []string{"data-tone"}, b.attrs)).Render(ctx, w)
}

// AlertSpec is an inline, static message (§5.2): its tone's icon, a
// visually hidden level word so the level isn't shown by colour alone, an
// optional title, and the children as its body. No live role: a message
// that appears later is a toast.
type AlertSpec struct {
	tone  Tone
	title i18n.Ter
	attrs templ.Attributes
}

// Alert starts an information alert.
func Alert() AlertSpec { return AlertSpec{} }

// Tone sets the level: Info (the default), Success, Warning or Danger.
// Any other tone panics at render: a message's level needs an icon and a
// word.
func (a AlertSpec) Tone(t Tone) AlertSpec { a.tone = t; return a }

// Title adds a title above the body.
func (a AlertSpec) Title(t i18n.Ter) AlertSpec { a.title = t; return a }

// Attrs adds attributes to the outer <div>, merged as Merge does; the
// alert keeps data-tone.
func (a AlertSpec) Attrs(at templ.Attributes) AlertSpec { a.attrs = Merge(a.attrs, nil, at); return a }

// level is a message tone's icon and word.
func level(component string, t Tone) (templ.Component, i18n.Ter) {
	switch t {
	case "", Info:
		return icons.Info(), alertInfo
	case Success:
		return icons.CircleCheck(), alertSuccess
	case Warning:
		return icons.Warning(), alertWarning
	case Danger:
		return icons.CircleX(), alertDanger
	}
	panic(fmt.Sprintf("elements: %s with tone %q: a message is Info, Success, Warning or Danger", component, t))
}

func (a AlertSpec) Render(ctx context.Context, w io.Writer) error {
	icon, word := level("Alert", a.tone)
	own := templ.Attributes{"class": "ui-alert"}
	if t := toneAttr(a.tone, Info); t != "" {
		own["data-tone"] = t
	}
	return alert(Merge(own, []string{"data-tone"}, a.attrs), icon, word, a.title).Render(ctx, w)
}

// Card's slots (§5.2): each renders only when filled.
var (
	CardHeader = ui.NewSlot()
	CardBody   = ui.NewSlot()
	CardFooter = ui.NewSlot()
)

// CardSpec is a card being described: a <section> with a header, a body
// and a footer, each rendered only when its slot is filled.
//
//	@elements.Card() {
//		@elements.CardHeader { <h2>Orders</h2> }
//		@elements.CardBody { … }
//	}
type CardSpec struct{ attrs templ.Attributes }

// Card starts a card.
func Card() CardSpec { return CardSpec{} }

// Attrs adds attributes to the <section>, merged as Merge does.
func (c CardSpec) Attrs(a templ.Attributes) CardSpec { c.attrs = Merge(c.attrs, nil, a); return c }

func (c CardSpec) Render(ctx context.Context, w io.Writer) error {
	return ui.Slotted(card(Merge(templ.Attributes{"class": "ui-card"}, nil, c.attrs)), CardHeader, CardBody, CardFooter).Render(ctx, w)
}
