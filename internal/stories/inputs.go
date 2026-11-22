package stories

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/forms/rule"
)

type toyRecord struct {
	id   string
	name string
}

type toyRecords []toyRecord

func (rs toyRecords) LabelOf(_ context.Context, id string) (string, bool, error) {
	for _, r := range rs {
		if r.id == id {
			return r.name, true, nil
		}
	}
	return "", false, nil
}

func (rs toyRecords) search(q string, limit int) []toyRecord {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []toyRecord
	for _, r := range rs {
		if q == "" || strings.Contains(strings.ToLower(r.name), q) {
			out = append(out, r)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}

var (
	toyCustomers = toyRecords{
		{"1", "Bassem Aziz"},
		{"2", "Carla Ruiz"},
		{"3", "Dina Farouk"},
		{"4", "Elias Berg"},
		{"5", "Fatma Nour"},
		{"6", "Gus Holm"},
		{"7", "Ann Lee"},
		{"8", "Hana Sato"},
		{"9", "Ilse Vos"},
		{"10", "Jamal Odeh"},
		{"11", "Kira Novak"},
		{"12", "Laila Haddad"},
		{"13", "Mona Zaki"},
		{"14", "Nils Lund"},
		{"15", "Omar Saleh"},
		{"16", "Petra Kos"},
		{"17", "Quinn Ash"},
		{"18", "Rana Adel"},
		{"19", "Sami Halabi"},
		{"20", "Tara Quist"},
		{"21", "Uma Rao"},
		{"22", "Victor Hale"},
	}
	toyProducts = toyRecords{
		{"101", "Brick 2x4 red"},
		{"103", "Brick 2x2 yellow"},
		{"105", "Wheel small"},
	}
)

func withGate(c templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(actions.Gate.With(ctx, actions.AllowAll), w)
	})
}

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

// remoteForm shows a remote select with an initial record chosen, and a
// second one blank.
func remoteForm() *forms.Form {
	f := forms.New()
	c := f.Number[int64]("customer_id").Label(i18n.Static("Customer")).
		Help(i18n.Static("Search by name or email.")).
		Required().Remote(ui.URL("/demo/customers/search"), toyCustomers)
	c.Set(7)
	f.Text("referrer_id").Label(i18n.Static("Referred by")).
		Remote(ui.URL("/demo/customers/search"), toyCustomers)
	return f
}

// uploadForm shows an upload field with two stored files (one a PDF, one
// an image with a thumbnail), rules for size and media type, and a file
// field that lost its posted file when the form failed.
func uploadForm() *forms.Form {
	f := forms.New()
	stored := []files.Stored{
		{Key: "k-81", Name: "receipt.pdf", Href: ui.URL("/files/k-81")},
		{Key: "k-82", Name: "box.png", Href: ui.URL("/files/k-82"), Image: true},
	}
	f.Uploads("attachments", stored).Label(i18n.Static("Attachments")).
		Help(i18n.Static("PNG, JPEG or PDF, up to 5 MB.")).
		Required().
		Is(rule.Files(rule.MaxSize(5<<20), rule.ContentType("image/png", "image/jpeg", "application/pdf")))
	f.File("invoice").Label(i18n.Static("Invoice"))
	f.Text("note").Label(i18n.Static("Note")).Required()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("attachments.keep", "k-81")
	mw.WriteField("attachments.keep", "k-82")
	fw, _ := mw.CreateFormFile("invoice", "invoice.pdf")
	fw.Write([]byte("%PDF-1.4"))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/catalog/none", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	f.Submit(context.Background(), r)
	return f
}

// listForm shows a list with MinItems(1), so a fresh form renders one
// blank row besides the template row.
func listForm() *forms.Form {
	f := forms.New()
	f.List("lines", func(g *forms.Form) int {
		g.Text("sku").Label(i18n.Static("SKU")).Required()
		g.Int("qty").Label(i18n.Static("Quantity")).Required().Is(rule.Min(1))
		return 0
	}).Label(i18n.Static("Order lines")).MinItems(1).MaxItems(5)
	return f
}

type nestedLine struct {
	product *forms.Field[int64]
	qty     *forms.Field[int]
	extras  *forms.List[*forms.Field[string]]
}

// nestedListForm shows a list with a nested list inside its rows.
func nestedListForm() *forms.Form {
	f := forms.New()
	lines := f.List("lines", func(g *forms.Form) nestedLine {
		var l nestedLine
		l.product = g.Number[int64]("product_id").Label(i18n.Static("Product")).
			Required().Remote(ui.URL("/demo/products/search"), toyProducts)
		l.qty = g.Int("qty").Label(i18n.Static("Quantity")).Required()
		l.extras = g.List("extras", func(e *forms.Form) *forms.Field[string] {
			return e.Text("note").Label(i18n.Static("Engraving or note")).Required()
		}).Label(i18n.Static("Extras")).MaxItems(3)
		return l
	}).Label(i18n.Static("Lines")).MinItems(1).MaxItems(5)
	row := lines.Add()
	row.product.Set(103)
	row.qty.Set(2)
	extra := row.extras.Add()
	extra.Set("Gift wrap")
	return f
}
