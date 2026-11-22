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
	id    string
	attrs templ.Attributes
}

// Menu starts a menu whose default trigger is an outline Button reading
// label.
func Menu(label i18n.Ter) MenuSpec { return MenuSpec{label: label} }

// ID gives the trigger a keyed id, and the popover "<id>-menu", for a menu
// whose trigger focus must find again after a swap (amendment A6). Without
// it, both ids are generated.
func (m MenuSpec) ID(id string) MenuSpec { m.id = id; return m }

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
	if m.id != "" && (strings.ContainsAny(m.id, " \t\n#") || strings.Contains(m.id, "--")) {
		panic(fmt.Sprintf("components: Menu.ID(%q): a keyed id has no white space, no # and no --", m.id))
	}
	return ui.Slotted(menu(m), MenuTrigger, MenuItems).Render(ctx, w)
}

// ids are the trigger's and the popover's: keyed with ID, else generated.
func (m MenuSpec) ids(ctx context.Context) (trigger, popover string) {
	if m.id != "" {
		return m.id, m.id + "-menu"
	}
	return menuID.Part(ctx, "trigger"), menuID.Get(ctx)
}

func (m MenuSpec) triggerAttrs(ctx context.Context) templ.Attributes {
	trigger, popover := m.ids(ctx)
	return templ.Attributes{
		"id":            trigger,
		"popovertarget": popover,
		"aria-haspopup": "menu",
		"aria-controls": popover,
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
	reason   i18n.Ter // Unavailable's
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

// Unavailable renders an item that can be read but not used: an action
// the record's state forbids (amendment A6, S21). It is aria-disabled and
// inert, named by its label alone, with reason as a second line that
// describes it; the menu's arrow keys stop on it, as they don't on a
// Disabled item. A blank reason panics at render.
func (m MenuItemSpec) Unavailable(reason i18n.Ter) MenuItemSpec { m.reason = reason; return m }

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
	if m.reason != nil {
		if strings.TrimSpace(m.reason.T(ctx)) == "" {
			panic("components: MenuItem with a blank Unavailable reason")
		}
		return unavailableItem(m).Render(ctx, w)
	}
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

var menuItemID = ui.NewID("menu-item")

func (m MenuItemSpec) unavailableAttrs(ctx context.Context) templ.Attributes {
	own := templ.Attributes{
		"class": "ui-menu-item", "role": "menuitem", "tabindex": "-1", "type": "button",
		"aria-disabled": "true", "data-ui-unavailable": true,
		"aria-labelledby":  menuItemID.Part(ctx, "label"),
		"aria-describedby": menuItemID.Part(ctx, "reason"),
	}
	if m.danger {
		own["data-tone"] = "danger"
	}
	rest := templ.Attributes{}
	for k, v := range m.attrs {
		if !inert(k) {
			rest[k] = v
		}
	}
	return elements.Merge(own, append(menuItemOwned[:len(menuItemOwned):len(menuItemOwned)], "aria-labelledby"), rest)
}

// inert reports whether an unavailable item drops the attribute k: its
// address, its form, its requests (htmx's data- aliases too), its
// confirmation and its commands.
func inert(k string) bool {
	switch k := strings.ToLower(k); {
	case k == "href", strings.HasPrefix(k, "form"), strings.HasPrefix(k, "hx-"), strings.HasPrefix(k, "data-hx-"),
		k == "data-ui-confirm", k == "command", k == "commandfor", k == "popovertarget":
		return true
	}
	return false
}
