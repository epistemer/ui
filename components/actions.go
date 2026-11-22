package components

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/elements"
)

// The action form and dialog's ids (server-state-interface.md §4.3).
const (
	actionFormID       = "ui-action-form"
	actionDialogID     = "ui-action-dialog"
	actionDialogBodyID = "ui-action-dialog-body"
	actionDialogTitle  = "ui-action-dialog-title"
)

// reasonTer is an unavailable item's reason for people.
type reasonTer struct{ err error }

func (r reasonTer) T(ctx context.Context, _ ...any) string { return errors.T(ctx, r.err) }

// requestAttrs make a Request item a submit button of the action form, so
// it posts with script off; another method than POST needs htmx.
func requestAttrs(it actions.Item) templ.Attributes {
	a := templ.Attributes{"formaction": string(it.URL)}
	if it.Method != "POST" {
		a["formmethod"] = "post"
	}
	return a
}

// htmxAttrs are an item's request attributes, by its kind, and its
// confirmation.
func htmxAttrs(ctx context.Context, it actions.Item) templ.Attributes {
	a := templ.Attributes{}
	switch it.Kind {
	case actions.Request:
		a["hx-"+strings.ToLower(it.Method)] = string(it.URL)
		a["hx-swap"] = "none"
	case actions.Open:
		a["hx-get"] = string(it.URL)
		a["hx-target"] = "#" + actionDialogBodyID
		a["hx-swap"] = "innerHTML"
		a["hx-push-url"] = "false"
	}
	if it.Question != nil && it.Kind != actions.Open {
		for k, v := range Confirm(ctx, it.Question) {
			a[k] = v
		}
	}
	return a
}

func merge(as ...templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	for _, a := range as {
		for k, v := range a {
			out[k] = v
		}
	}
	return out
}

// ActionButtonsSpec is a row of buttons for actions (§4.2).
type ActionButtonsSpec struct {
	items []actions.Item
	attrs templ.Attributes
}

// ActionButtons renders items as buttons: the primary one solid, a danger
// one outlined in the danger tone, the rest outlined neutral. It renders
// nothing for no items.
func ActionButtons(items []actions.Item) ActionButtonsSpec { return ActionButtonsSpec{items: items} }

// Attrs adds attributes to the row's <div>, merged as elements.Merge does.
func (b ActionButtonsSpec) Attrs(a templ.Attributes) ActionButtonsSpec {
	b.attrs = elements.Merge(b.attrs, nil, a)
	return b
}

func (b ActionButtonsSpec) Render(ctx context.Context, w io.Writer) error {
	if len(b.items) == 0 {
		return nil
	}
	return actionButtons(b).Render(ctx, w)
}

// button is an item's Button, with what its kind needs.
func button(ctx context.Context, it actions.Item) elements.ButtonSpec {
	bt := elements.Button().Variant(elements.Outline).Tone(elements.Neutral)
	switch {
	case it.Primary:
		bt = elements.Button()
	case it.Danger:
		bt = bt.Tone(elements.Danger)
	}
	bt = bt.Attrs(it.Attrs)
	switch {
	case it.Reason != nil:
		return bt.Unavailable(reasonTer{it.Reason})
	case it.Kind == actions.Request:
		return bt.Submit().Attrs(merge(templ.Attributes{"form": actionFormID}, requestAttrs(it), htmxAttrs(ctx, it)))
	default:
		return bt.Href(ui.URL(string(it.URL))).Attrs(htmxAttrs(ctx, it))
	}
}

// ActionMenuSpec is a menu of actions (§4.2): an icon trigger, an item
// each, danger items last after a separator.
type ActionMenuSpec struct {
	label    i18n.Ter
	items    []actions.Item
	id       string
	labelled string
	attrs    templ.Attributes
}

// ActionMenu starts a menu of items labelled label. It renders nothing for
// no items.
func ActionMenu(label i18n.Ter, items []actions.Item) ActionMenuSpec {
	return ActionMenuSpec{label: label, items: items}
}

// ID gives the trigger a keyed id (Menu's ID), so focus can find it again
// after a swap replaced it.
func (m ActionMenuSpec) ID(id string) ActionMenuSpec { m.id = id; return m }

// LabelledBy adds the element with this id, a row's label cell, to the
// trigger's name: "Actions omar42@example.com".
func (m ActionMenuSpec) LabelledBy(id string) ActionMenuSpec { m.labelled = id; return m }

// Attrs adds attributes to the <ui-menu>, as Menu's Attrs does.
func (m ActionMenuSpec) Attrs(a templ.Attributes) ActionMenuSpec {
	m.attrs = elements.Merge(m.attrs, nil, a)
	return m
}

func (m ActionMenuSpec) Render(ctx context.Context, w io.Writer) error {
	if len(m.items) == 0 {
		return nil
	}
	if m.label == nil || strings.TrimSpace(m.label.T(ctx)) == "" {
		panic("components: ActionMenu with no label: a menu needs its name")
	}
	return actionMenu(m).Render(ctx, w)
}

func (m ActionMenuSpec) menu() MenuSpec {
	mn := Menu(m.label).Attrs(m.attrs)
	if m.id != "" {
		mn = mn.ID(m.id)
	}
	return mn
}

// triggerAttrs name the trigger by its own text and, with LabelledBy, the
// row's label.
func (m ActionMenuSpec) triggerAttrs(trigger templ.Attributes) templ.Attributes {
	if m.labelled == "" {
		return trigger
	}
	id, _ := trigger["id"].(string)
	return merge(trigger, templ.Attributes{"aria-labelledby": id + " " + m.labelled})
}

func split(items []actions.Item) (plain, danger []actions.Item) {
	for _, it := range items {
		if it.Danger {
			danger = append(danger, it)
		} else {
			plain = append(plain, it)
		}
	}
	return plain, danger
}

// menuItem is an item's MenuItem, with what its kind needs.
func menuItem(ctx context.Context, it actions.Item) MenuItemSpec {
	mi := MenuItem().Attrs(it.Attrs)
	if it.Danger {
		mi = mi.Danger()
	}
	switch {
	case it.Reason != nil:
		return mi.Unavailable(reasonTer{it.Reason})
	case it.Kind == actions.Request:
		return mi.Submit(actionFormID).Attrs(merge(requestAttrs(it), htmxAttrs(ctx, it)))
	default:
		return mi.Href(ui.URL(string(it.URL))).Attrs(htmxAttrs(ctx, it))
	}
}

// InActionDialog reports whether r loads into the action dialog, so its
// handler renders the form alone, with its ActionTitle (§4.3).
func InActionDialog(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == actionDialogBodyID
}
