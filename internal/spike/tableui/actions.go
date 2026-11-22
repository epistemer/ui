// Package tableui is the spike of what ui/components gains
// (server-state-design.md §7): the table's view, pagination, the action
// placements, the action form and dialog, and Reply.
package tableui

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/spike/actions"
)

// The action form and dialog's ids (§5.4).
const (
	ActionFormID       = "ui-action-form"
	ActionDialogID     = "ui-action-dialog"
	ActionDialogBodyID = "ui-action-dialog-body"
	ActionDialogTitle  = "ui-action-dialog-title"
)

var (
	actionsLabel = i18n.Key("ui.components.actions.label", "Actions")
	closeLabel   = i18n.Key("ui.components.dialog.close", "Close")
	dialogLabel  = i18n.Key("ui.components.actions.dialog", "Action")
)

// InActionDialog reports whether r loads into the action dialog, so the
// handler renders its form alone.
func InActionDialog(r *http.Request) bool {
	return r.Header.Get("HX-Target") == ActionDialogBodyID
}

// htmxAttrs are the request attributes of an item, by its kind.
func htmxAttrs(ctx context.Context, it actions.Item) templ.Attributes {
	a := templ.Attributes{}
	switch it.Kind {
	case actions.Request:
		a["hx-"+strings.ToLower(it.Method)] = string(it.URL)
		a["hx-swap"] = "none"
	case actions.Open:
		a["hx-get"] = string(it.URL)
		a["hx-target"] = "#" + ActionDialogBodyID
		a["hx-swap"] = "innerHTML"
		a["hx-push-url"] = "false"
	}
	if it.Question != nil {
		for k, v := range components.Confirm(ctx, it.Question) {
			a[k] = v
		}
	}
	for k, v := range it.Attrs {
		a[k] = v
	}
	return a
}

// requestAttrs make a Request item a submit button of the action form,
// so it posts with script off.
func requestAttrs(it actions.Item, form string) templ.Attributes {
	a := templ.Attributes{"form": form, "formaction": string(it.URL)}
	if it.Method != "POST" {
		// No form in HTML can send another method: it needs htmx.
		a["formmethod"] = "post"
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

// split puts danger items last.
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

var reasonID = ui.NewID("action-reason")

func buttonSpec(it actions.Item) elements.ButtonSpec {
	b := elements.Button().Variant(elements.Outline).Tone(elements.Neutral)
	switch {
	case it.Primary:
		b = elements.Button()
	case it.Danger:
		b = b.Tone(elements.Danger)
	}
	return b
}

// buttonClass and buttonData render a disabled item like Button would,
// which can't yet render aria-disabled on a button (S21).
func buttonClass(it actions.Item) string { return "ui-button" }

func buttonData(it actions.Item) templ.Attributes {
	a := templ.Attributes{}
	switch {
	case it.Primary:
	case it.Danger:
		a["data-variant"], a["data-tone"] = "outline", "danger"
	default:
		a["data-variant"], a["data-tone"] = "outline", "neutral"
	}
	return a
}

// labelledBy names a row's menu trigger by its own text and the row's
// label cell.
func labelledBy(trigger templ.Attributes, rowLabel string) templ.Attributes {
	if rowLabel == "" {
		return nil
	}
	id, _ := trigger["id"].(string)
	return templ.Attributes{"aria-labelledby": id + " " + rowLabel}
}
