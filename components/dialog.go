package components

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
)

// Dialog's slots (§7.1). The trigger's fill receives the attributes that
// open the dialog; the body is required; the actions render only when
// filled.
var (
	DialogTrigger = ui.NewSlotOf[templ.Attributes]()
	DialogBody    = ui.NewSlot()
	DialogActions = ui.NewSlot()
)

var dialogID = ui.NewID("dialog")

// DialogSpec is a modal dialog being described: the APG dialog (modal)
// pattern. Invoker commands open it (the ui script stands in where they're
// missing); Esc and its close button close it, and the browser returns
// focus to the trigger.
//
//	@components.Dialog(title) {
//		@components.DialogTrigger {
//			@elements.Button().Attrs(components.DialogTrigger.Value(ctx)) { Edit }
//		}
//		@components.DialogBody { … }
//		@components.DialogActions {
//			@elements.Button().Attrs(components.DialogClose(ctx, "save")) { Save }
//		}
//	}
type DialogSpec struct {
	title i18n.Ter
	attrs templ.Attributes
}

// Dialog starts a modal dialog titled title. Unfilled, its trigger is an
// outline Button reading the title.
func Dialog(title i18n.Ter) DialogSpec { return DialogSpec{title: title} }

// Attrs adds attributes to the <dialog>, merged as elements.Merge does; the
// dialog keeps id and aria-labelledby.
func (d DialogSpec) Attrs(a templ.Attributes) DialogSpec {
	d.attrs = elements.Merge(d.attrs, nil, a)
	return d
}

func (d DialogSpec) Render(ctx context.Context, w io.Writer) error {
	if d.title == nil {
		panic("components: Dialog(nil): a dialog needs its title")
	}
	return ui.Slotted(dialog(d), DialogTrigger, DialogBody, DialogActions).Render(ctx, w)
}

func (d DialogSpec) dialogAttrs(ctx context.Context) templ.Attributes {
	own := templ.Attributes{"class": "ui-dialog", "id": dialogID.Get(ctx), "aria-labelledby": dialogID.Part(ctx, "title")}
	return elements.Merge(own, []string{"id", "aria-labelledby"}, d.attrs)
}

func triggerAttrs(ctx context.Context) templ.Attributes {
	return templ.Attributes{"commandfor": dialogID.Get(ctx), "command": "show-modal", "aria-haspopup": "dialog"}
}

func requireBody(ctx context.Context) error {
	if !DialogBody.Filled(ctx) {
		panic("components: Dialog without DialogBody: fill it with the dialog's content")
	}
	return nil
}

// DialogClose returns the attributes of a button that closes the dialog
// around it, with value as its returnValue. Fills see the placement's
// context, so inside DialogBody or DialogActions it names that dialog.
// Outside a dialog the kernel's Get panics.
func DialogClose(ctx context.Context, value string) templ.Attributes {
	return templ.Attributes{"commandfor": dialogID.Get(ctx), "command": "close", "value": value}
}
