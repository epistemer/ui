package stories

import (
	"context"
	"net/url"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

type choice struct {
	value string
	label i18n.TKer
}

func (c choice) Value() string    { return c.value }
func (c choice) Label() i18n.TKer { return c.label }

var (
	fixedChoice = choice{"fixed", i18n.Key("ui.story.promotion.fixed", "Fixed amount")}
	pctChoice   = choice{"percentage", i18n.Key("ui.story.promotion.percentage", "Percentage")}
)

// promotion is a small form of the toy shop: a type that switches two
// fields on and off, an email that reads left to right, and a checkbox.
type promotion struct {
	*forms.Form
	Type    *forms.Field[string]
	Amount  *forms.Field[string]
	Percent *forms.Field[int]
	Email   *forms.Field[string]
	Agree   *forms.Field[bool]
}

func newPromotion() *promotion {
	p := &promotion{Form: forms.New()}
	p.Type = p.Text("type").Label(i18n.Static("Type")).Options(fixedChoice, pctChoice).Required()
	p.Amount = p.Text("amount").Label(i18n.Static("Amount")).
		Help(i18n.Static("In the store's currency.")).
		When(p.Type.Eq("fixed")).Required()
	p.Percent = p.Int("percentage").Label(i18n.Static("Percentage")).
		When(p.Type.Eq("percentage")).Required().Is(rule.Between(1, 100))
	p.Email = p.Text("email").Label(i18n.Static("Contact email")).Is(rule.Email())
	p.Agree = p.Checkbox("agree").Label(i18n.Static("The terms are agreed")).Required()
	return p
}

func promotionFresh() *forms.Form { return newPromotion().Form }

func promotionFilled() *forms.Form {
	p := newPromotion()
	p.Type.Set("fixed")
	p.Amount.Set("12.50")
	p.Email.Set("shop@example.com")
	p.Agree.Set(true)
	return p.Form
}

func promotionErrors() *forms.Form {
	p := newPromotion()
	p.Bind(forms.FromValues(url.Values{"type": {"percentage"}, "percentage": {"250"}, "email": {"not an email"}}))
	p.Validate(context.Background())
	return p.Form
}

// shipping shows the other kinds: a group with its own condition, a
// three-state toggle, a textarea, files and a hidden field.
type shipping struct {
	*forms.Form
	Gift    *forms.Field[bool]
	Address *address
}

type address struct {
	*forms.Form
	Street *forms.Field[string]
	Note   *forms.Field[string]
}

func newShipping() *shipping {
	s := &shipping{Form: forms.New()}
	s.Text("ref").Hint(forms.KindHidden)
	s.Gift = s.Bool("gift").Label(i18n.Static("Wrapped as a gift?")).Required()
	s.Address = s.Group("address", func(g *forms.Form) *address {
		a := &address{Form: g}
		a.Street = g.Text("street").Label(i18n.Static("Street")).Required()
		a.Note = g.Text("note").Label(i18n.Static("Note for the courier")).Hint(forms.KindMultiline).
			Placeholder(i18n.Static("Ring twice"))
		return a
	})
	s.Address.Label(i18n.Static("Gift address")).When(s.Gift.Eq(true))
	s.Files("photos").Label(i18n.Static("Photos of the toy"))
	return s
}

func shippingFresh() *forms.Form { return newShipping().Form }

func shippingGift() *forms.Form {
	s := newShipping()
	s.Gift.Set(true)
	s.Address.Street.Set("12 Corniche")
	s.Address.Note.Set("\nleave with the doorman")
	return s.Form
}

// signIn shows a password with its reveal button, and a checkbox's help
// above its row.
func newSignIn() *forms.Form {
	f := forms.New()
	f.Text("email").Label(i18n.Static("Email")).Required().Is(rule.Email())
	f.Text("password").Raw().Hint(forms.KindPassword).Label(i18n.Static("Password")).Required().Is(rule.MinLen(12))
	f.Checkbox("remember").Label(i18n.Static("Keep me signed in")).Help(i18n.Static("On this device only."))
	return f
}

func signInFresh() *forms.Form { return newSignIn() }

func signInErrors() *forms.Form {
	f := newSignIn()
	f.Bind(forms.FromValues(url.Values{"email": {"shop@example.com"}, "password": {"short"}}))
	f.Validate(context.Background())
	return f
}
