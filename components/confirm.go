package components

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

// Confirm returns the attributes that make an action ask first (§7.2): the
// page's ConfirmDialog asks question. Spread it on any button, link, menu
// item, form or element carrying hx-*; the ui script asks, then lets the
// action run (§10.3).
//
//	@elements.Button().Submit().Attrs(components.Confirm(ctx, q)) { Delete }
func Confirm(ctx context.Context, question i18n.Ter) templ.Attributes {
	if question == nil || strings.TrimSpace(question.T(ctx)) == "" {
		panic("components: Confirm with no question")
	}
	return templ.Attributes{"data-ui-confirm": question.T(ctx)}
}

// ConfirmDialog is the page's one confirmation dialog. The app and auth
// shells render it; a page without a shell renders it once. Without it the
// script falls back to the browser's own confirm.
func ConfirmDialog() templ.Component { return confirmDialog() }
