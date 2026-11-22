package layout

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
)

// The auth shell's slots (§8.3). AuthBody is required; the brand and the
// footer render only when filled.
var (
	AuthBrand  = ui.NewSlot()
	AuthBody   = ui.NewSlot()
	AuthFooter = ui.NewSlot()
)

// AuthSpec is the auth shell being described: a centred card for sign-in
// and its siblings, with the brand, a title, the body (a form of
// inputs.Field) and a footer of other links. The toast region and the
// confirmation dialog follow, outside #ui-main.
type AuthSpec struct {
	title i18n.Ter
	attrs templ.Attributes
}

// Auth starts the auth shell titled title (its <h1>).
func Auth(title i18n.Ter) AuthSpec { return AuthSpec{title: title} }

// Attrs adds attributes to the outer <div class="ui-auth">, merged as
// elements.Merge does.
func (a AuthSpec) Attrs(at templ.Attributes) AuthSpec {
	a.attrs = elements.Merge(a.attrs, nil, at)
	return a
}

func (a AuthSpec) Render(ctx context.Context, w io.Writer) error {
	if a.title == nil || strings.TrimSpace(a.title.T(ctx)) == "" {
		panic("layout: Auth with no title: a page needs its heading")
	}
	return ui.Slotted(auth(a), AuthBrand, AuthBody, AuthFooter).Render(ctx, w)
}

func requireAuthBody(ctx context.Context) error {
	if !AuthBody.Filled(ctx) {
		panic("layout: Auth without AuthBody: fill it with the page's form")
	}
	return nil
}
