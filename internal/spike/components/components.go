// Package components is the D4 spike of ui/components: Dialog, the shared
// confirmation, and Menu (design §6, §10).
package components

import (
	"context"
	"io"

	"github.com/a-h/templ"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui"
)

// ui's own text (Q5): keys under ui.<package>, English by default.
var (
	CloseText    = i18n.Key("ui.dialog.close", "Close")
	ConfirmTitle = i18n.Key("ui.confirm.title", "Are you sure?")
	ConfirmYes   = i18n.Key("ui.confirm.yes", "Confirm")
	ConfirmNo    = i18n.Key("ui.confirm.no", "Cancel")
)

// Dialog's slots. The trigger receives the attributes that open the
// dialog; the body and the actions are the caller's content.
var (
	DialogTrigger = ui.NewSlotOf[templ.Attributes]()
	DialogBody    = ui.NewSlot()
	DialogActions = ui.NewSlot()
)

var dialogID = ui.NewID("dialog")

// DialogSpec is a modal dialog being described.
type DialogSpec struct{ title i18n.Ter }

// Dialog starts a modal dialog titled title. It opens with commandfor and
// command="show-modal" (the ui script stands in where the browser lacks
// invoker commands), closes with Esc or its close button, and returns
// focus to its trigger, all natively:
//
//	@components.Dialog(title) {
//		@components.DialogTrigger {
//			@elements.Button().Attrs(components.DialogTrigger.Value(ctx)) { Edit }
//		}
//		@components.DialogBody { … }
//	}
func Dialog(title i18n.Ter) DialogSpec { return DialogSpec{title: title} }

func (d DialogSpec) Render(ctx context.Context, w io.Writer) error {
	return ui.Slotted(dialog(d.title), DialogTrigger, DialogBody, DialogActions).Render(ctx, w)
}

func triggerAttrs(ctx context.Context) templ.Attributes {
	return templ.Attributes{
		"commandfor":    dialogID.Get(ctx),
		"command":       "show-modal",
		"aria-haspopup": "dialog",
	}
}

// DialogClose returns the attributes of a button that closes the dialog
// around it. Fills see the placement's context, so inside DialogBody or
// DialogActions it names that dialog; value becomes the dialog's
// returnValue.
func DialogClose(ctx context.Context, value string) templ.Attributes {
	return templ.Attributes{"commandfor": dialogID.Get(ctx), "command": "close", "value": value}
}

// Confirm returns the attributes that make an action ask first (Q6): the
// shared confirmation dialog, rendered once by ConfirmDialog, asks msg.
// For an htmx action the ui script answers htmx:confirm; for a link or a
// submit button it answers the click.
func Confirm(ctx context.Context, msg i18n.Ter) templ.Attributes {
	return templ.Attributes{"data-ui-confirm": msg.T(ctx)}
}

// ConfirmDialog is the page's one confirmation dialog. The layout renders
// it once; actions that Confirm use it.
func ConfirmDialog() templ.Component { return confirmDialog() }

// Menu's slots: the trigger receives the attributes that open the menu;
// the items are MenuItem components.
var (
	MenuTrigger = ui.NewSlotOf[templ.Attributes]()
	MenuItems   = ui.NewSlot()
)

var menuID = ui.NewID("menu")

// MenuSpec is a menu button being described: the APG menu button pattern.
type MenuSpec struct{ label i18n.Ter }

// Menu starts a menu whose default trigger reads label. It opens a
// popover with popovertarget (Baseline 2024: light dismiss, Esc, focus
// back to the trigger); <ui-menu> moves focus to the first item, handles
// the arrow keys, Home and End, keeps aria-expanded in step, and places
// the popover where anchor positioning is missing.
func Menu(label i18n.Ter) MenuSpec { return MenuSpec{label: label} }

func (m MenuSpec) Render(ctx context.Context, w io.Writer) error {
	return ui.Slotted(menu(m.label), MenuTrigger, MenuItems).Render(ctx, w)
}

func menuTriggerAttrs(ctx context.Context) templ.Attributes {
	return templ.Attributes{
		"id":            menuID.Part(ctx, "trigger"),
		"popovertarget": menuID.Get(ctx),
		"aria-haspopup": "menu",
		"aria-controls": menuID.Get(ctx),
	}
}

// MenuItemSpec is one item of a menu: a button, or a link with Href.
type MenuItemSpec struct {
	href  templ.SafeURL
	tone  string
	attrs templ.Attributes
}

// MenuItem starts an item; its children are its text.
func MenuItem() MenuItemSpec { return MenuItemSpec{} }

// Href makes the item a link.
func (m MenuItemSpec) Href(u templ.SafeURL) MenuItemSpec { m.href = u; return m }

// Danger marks a destructive item.
func (m MenuItemSpec) Danger() MenuItemSpec { m.tone = "danger"; return m }

// Attrs adds attributes, such as Confirm's or hx-*.
func (m MenuItemSpec) Attrs(a templ.Attributes) MenuItemSpec {
	if m.attrs == nil {
		m.attrs = templ.Attributes{}
	} else {
		c := templ.Attributes{}
		for k, v := range m.attrs {
			c[k] = v
		}
		m.attrs = c
	}
	for k, v := range a {
		m.attrs[k] = v
	}
	return m
}

func (m MenuItemSpec) Render(ctx context.Context, w io.Writer) error {
	own := templ.Attributes{"class": "ui-menu-item", "role": "menuitem", "tabindex": "-1"}
	if m.tone != "" {
		own["data-tone"] = m.tone
	}
	for k, v := range m.attrs {
		if _, taken := own[k]; !taken {
			own[k] = v
		}
	}
	if m.href != "" {
		own["href"] = m.href
		return menuLink(own).Render(ctx, w)
	}
	own["type"] = "button"
	return menuButton(own).Render(ctx, w)
}
