package components

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/icons"
)

// Toasts is the page's region for toasts (§7.5), rendered once by the
// shell, outside what htmx swaps: a manual popover, so it shows above an
// open modal dialog, and a polite live region.
func Toasts() templ.Component { return toastRegion() }

// ToastSpec is a message that arrives later (§7.5): the tone's icon and
// hidden word, the text, and a close button. Success and info toasts time
// out after 6 s, pausing on hover and focus; warning and danger stay until
// closed. It moves itself into the region when the script runs, so a toast
// rendered anywhere in a full page is announced too.
type ToastSpec struct {
	text  i18n.Ter
	tone  elements.Tone
	attrs templ.Attributes
}

// Toast starts an information toast saying text.
func Toast(text i18n.Ter) ToastSpec { return ToastSpec{text: text} }

// Tone sets the level: Info (the default), Success, Warning or Danger. Any
// other tone panics at render.
func (t ToastSpec) Tone(tone elements.Tone) ToastSpec { t.tone = tone; return t }

// Attrs adds attributes to the <ui-toast>, merged as elements.Merge does;
// the toast keeps data-tone and data-ui-timeout.
func (t ToastSpec) Attrs(a templ.Attributes) ToastSpec {
	t.attrs = elements.Merge(t.attrs, nil, a)
	return t
}

func (t ToastSpec) Render(ctx context.Context, w io.Writer) error {
	if t.text == nil || strings.TrimSpace(t.text.T(ctx)) == "" {
		panic("components: Toast with no text")
	}
	var icon templ.Component
	var word i18n.Ter
	own := templ.Attributes{"class": "ui-toast"}
	switch t.tone {
	case "", elements.Info:
		icon, word = icons.Info(), toastInfo
		own["data-ui-timeout"] = "6000"
	case elements.Success:
		icon, word = icons.CircleCheck(), toastSuccess
		own["data-ui-timeout"] = "6000"
		own["data-tone"] = "success"
	case elements.Warning:
		icon, word = icons.Warning(), toastWarning
		own["data-tone"] = "warning"
	case elements.Danger:
		icon, word = icons.CircleX(), toastDanger
		own["data-tone"] = "danger"
	default:
		panic(fmt.Sprintf("components: Toast with tone %q: a message is Info, Success, Warning or Danger", t.tone))
	}
	return toast(elements.Merge(own, []string{"data-tone", "data-ui-timeout"}, t.attrs), icon, word, t.text).Render(ctx, w)
}

// ToastSwap wraps toasts for an htmx response: htmx 2 appends them to the
// region out of band. ADR 0005 allows this one hx-* attribute on a
// response fragment (V18).
func ToastSwap(toasts ...templ.Component) templ.Component { return toastSwap(toasts) }
