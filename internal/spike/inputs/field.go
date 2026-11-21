// Package inputs is the D4 spike of ui/inputs: one widget rendered from a
// forms.Control (design §10), with its label, help and error wired by
// generated ids, and its condition handed to the ui script.
package inputs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"

	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/spike/elements"
)

var fieldID = ui.NewID("field")

// ChooseText is the empty first option of a select. A required select
// without one shows its first option chosen when nothing is, so the page
// and the server would disagree about the value (spike finding); with it,
// required makes the browser ask for a choice.
var ChooseText = i18n.Key("ui.inputs.choose", "Choose…")

func anySelected(c forms.Control) bool {
	for o := range c.Options() {
		if o.Selected {
			return true
		}
	}
	return false
}

// FieldSpec is a form field being described: a widget for its control.
type FieldSpec struct {
	c     forms.Control
	attrs templ.Attributes
}

// Field starts a widget for c. Its kind picks the element: <input> of the
// right type, <textarea>, <select>, or a checkbox before its label.
func Field(c forms.Control) FieldSpec { return FieldSpec{c: c} }

// Attrs adds attributes to the control element itself (not its wrapper),
// merged as elements.Merge does. The field keeps id, name, type, required,
// disabled, aria-invalid and the ids aria-describedby lists.
func (f FieldSpec) Attrs(a templ.Attributes) FieldSpec {
	f.attrs = elements.Merge(f.attrs, nil, a)
	return f
}

var fieldOwned = []string{"id", "name", "type", "required", "disabled", "aria-invalid"}

// ltrTypes hold text that reads left to right in any locale (ADR 0004).
var ltrTypes = map[string]bool{"email": true, "url": true, "tel": true}

// textTypes take a placeholder.
var textTypes = map[string]bool{"text": true, "email": true, "url": true, "tel": true, "search": true, "password": true, "number": true}

func (f FieldSpec) inputType() string {
	for k, v := range f.c.Attrs() {
		if k == "type" {
			return v
		}
	}
	switch f.c.Kind() {
	case forms.KindPassword:
		return "password"
	case forms.KindNumber:
		return "number"
	case forms.KindDate:
		return "date"
	case forms.KindDateTime:
		return "datetime-local"
	case forms.KindCheckbox, forms.KindToggle:
		return "checkbox"
	case forms.KindHidden:
		return "hidden"
	case forms.KindFile, forms.KindFiles:
		return "file"
	}
	return "text"
}

func (f FieldSpec) wrapperAttrs() (templ.Attributes, error) {
	a := templ.Attributes{"class": "ui-field"}
	if cond, ok := f.c.When(); ok {
		b, err := json.Marshal(cond)
		if err != nil {
			return nil, err
		}
		a["data-ui-when"] = string(b)
	}
	if !f.c.Active() {
		a["hidden"] = true
	}
	return a, nil
}

func (f FieldSpec) controlAttrs(ctx context.Context) templ.Attributes {
	typ := f.inputType()
	own := templ.Attributes{
		"class": "ui-input",
		"id":    fieldID.Get(ctx),
		"name":  f.c.Path(),
		// A required checkbox must be ticked, which is what the attribute
		// means; a three-state toggle rendered as a checkbox isn't.
		"required": f.c.Required() && f.c.Kind() != forms.KindToggle,
		"disabled": !f.c.Active(),
	}
	for k, v := range f.c.Attrs() {
		if k != "type" {
			own[k] = v
		}
	}
	switch f.c.Kind() {
	case forms.KindSelect, forms.KindMultiSelect, forms.KindMultiline:
		if f.c.Kind() == forms.KindMultiSelect {
			own["multiple"] = true
		}
	default:
		own["type"] = typ
		if ltrTypes[typ] {
			own["dir"] = "ltr"
		}
		switch typ {
		case "checkbox":
			own["value"] = "true"
			own["checked"] = len(f.c.Text()) > 0 && f.c.Text()[0] == "true"
		case "password", "file":
		default:
			if t := f.c.Text(); len(t) > 0 {
				own["value"] = t[0]
			}
		}
	}
	// forms' Placeholder falls back to the label. ui's fields always show
	// their label, so a placeholder repeating it would be noise for a
	// screen reader: render one only when it says something else, and
	// only where text is typed (spike finding).
	if textTypes[typ] && f.c.Kind() != forms.KindSelect && f.c.Kind() != forms.KindMultiSelect {
		if p, l := f.c.Placeholder(), f.c.Label(); p != nil && (l == nil || p.T(ctx) != l.T(ctx)) {
			own["placeholder"] = p.T(ctx)
		}
	}
	var described []string
	if f.c.Help() != nil {
		described = append(described, fieldID.Part(ctx, "help"))
	}
	if f.c.Err() != nil {
		own["aria-invalid"] = "true"
		described = append(described, fieldID.Part(ctx, "error"))
	}
	if len(described) > 0 {
		own["aria-describedby"] = strings.Join(described, " ")
	}
	return elements.Merge(own, fieldOwned, f.attrs)
}

// Render renders the field.
func (f FieldSpec) Render(ctx context.Context, w io.Writer) error {
	switch f.c.Kind() {
	case forms.KindGroup, forms.KindList:
		return fmt.Errorf("inputs: Field(%q): a %s renders through its children", f.c.Path(), f.c.Kind())
	case forms.KindHidden:
		return hidden(f.c.Path(), f.c.Text()).Render(ctx, w)
	}
	wrapper, err := f.wrapperAttrs()
	if err != nil {
		return err
	}
	return field(f, wrapper).Render(ctx, w)
}
