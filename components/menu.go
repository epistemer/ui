package components

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/links"
)

// Menu's slots (§7.3): the trigger's fill receives the attributes that
// open the menu; the items are required.
var (
	MenuTrigger = ui.NewSlotOf[templ.Attributes]()
	MenuItems   = ui.NewSlot()
)

var menuID = ui.NewID("menu")

// MenuSpec is a menu button being described: the APG menu button pattern.
// popovertarget opens the popover natively (light dismiss, Esc, focus back
// to the trigger) and anchors it to the trigger; <ui-menu> moves focus into
// it, handles the keys, keeps aria-expanded in step, and places the
// popover where anchor positioning is missing.
type MenuSpec struct {
	label i18n.Ter
	attrs templ.Attributes
}

// Menu starts a menu whose default trigger is an outline Button reading
// label.
func Menu(label i18n.Ter) MenuSpec { return MenuSpec{label: label} }

// Attrs adds attributes to the <ui-menu> element, merged as elements.Merge
// does.
func (m MenuSpec) Attrs(a templ.Attributes) MenuSpec {
	m.attrs = elements.Merge(m.attrs, nil, a)
	return m
}

func (m MenuSpec) Render(ctx context.Context, w io.Writer) error {
	if m.label == nil || strings.TrimSpace(m.label.T(ctx)) == "" {
		panic("components: Menu with no label: a menu needs its name")
	}
	return ui.Slotted(menu(m), MenuTrigger, MenuItems).Render(ctx, w)
}

func menuTriggerAttrs(ctx context.Context) templ.Attributes {
	return templ.Attributes{
		"id":            menuID.Part(ctx, "trigger"),
		"popovertarget": menuID.Get(ctx),
		"aria-haspopup": "menu",
		"aria-controls": menuID.Get(ctx),
	}
}

func requireItems(ctx context.Context) error {
	if !MenuItems.Filled(ctx) {
		panic("components: Menu without MenuItems: fill it with MenuItem components")
	}
	return nil
}

// MenuItemSpec is one item of a menu: a button, or a link with Href.
// Children are its text and icon.
type MenuItemSpec struct {
	href     templ.SafeURL
	danger   bool
	disabled bool
	form     string
	attrs    templ.Attributes
}

// MenuItem starts an item.
func MenuItem() MenuItemSpec { return MenuItemSpec{} }

// Href makes the item a link to l. No address leaves it a button.
func (m MenuItemSpec) Href(l ui.Link) MenuItemSpec { m.href = links.Href(l); return m }

// Disabled disables the item when d is true: arrow keys skip it, the
// script stops its clicks, and a link loses its href.
func (m MenuItemSpec) Disabled(d bool) MenuItemSpec { m.disabled = d; return m }

// Danger marks a destructive item.
func (m MenuItemSpec) Danger() MenuItemSpec { m.danger = true; return m }

// Submit makes the item submit the form with id form, which may sit
// outside the menu. With Href, Render panics.
func (m MenuItemSpec) Submit(form string) MenuItemSpec { m.form = form; return m }

// Attrs adds attributes to the button or link, merged as elements.Merge
// does. The item keeps type, form, role, tabindex, href, aria-disabled and
// data-tone.
func (m MenuItemSpec) Attrs(a templ.Attributes) MenuItemSpec {
	m.attrs = elements.Merge(m.attrs, nil, a)
	return m
}

var menuItemOwned = []string{"type", "form", "role", "tabindex", "href", "aria-disabled", "data-tone"}

func (m MenuItemSpec) Render(ctx context.Context, w io.Writer) error {
	if m.href != "" && m.form != "" {
		panic(fmt.Sprintf("components: MenuItem with href %q can't submit", m.href))
	}
	own := templ.Attributes{"class": "ui-menu-item", "role": "menuitem", "tabindex": "-1"}
	if m.danger {
		own["data-tone"] = "danger"
	}
	if m.disabled {
		own["aria-disabled"] = "true"
	}
	if m.href != "" {
		if !m.disabled {
			own["href"] = m.href
		}
		return menuLink(elements.Merge(own, menuItemOwned, m.attrs)).Render(ctx, w)
	}
	own["type"] = "button"
	if m.form != "" {
		own["type"] = "submit"
		own["form"] = m.form
	}
	return menuButton(elements.Merge(own, menuItemOwned, m.attrs)).Render(ctx, w)
}
