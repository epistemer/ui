// Package inputs is the D4 spike of ui/inputs: widgets rendered from a
// forms.Control (design §10), with label, help and error wired by
// generated ids, and conditions handed to the ui script.
package inputs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/internal/spike/elements"
)

var fieldID = ui.NewID("field")

// ui's own text (Q5).
var (
	// ChooseText is a select's empty first option. Without one, a browser
	// shows the first option chosen when nothing is, and the page and the
	// server disagree about the value (spike finding 11).
	ChooseText = i18n.Key("ui.inputs.choose", "Choose…")
	YesText    = i18n.Key("ui.inputs.yes", "Yes")
	NoText     = i18n.Key("ui.inputs.no", "No")
)

// FieldSpec is a form field being described: a widget for its control.
type FieldSpec struct {
	c      forms.Control
	attrs  templ.Attributes
	nested bool // inside a group: the group's fieldset hides and disables it
}

// Field starts a widget for c. Its kind picks the element:
//   - <input> of the right type; a checkbox before its label;
//   - <textarea>, <select> (multiple for a multiselect);
//   - a Yes/No select for a three-state toggle, which a checkbox can't
//     express (it has no "no answer");
//   - a <fieldset> for a group, holding its children;
//   - <input type="hidden"> for a hidden field.
func Field(c forms.Control) FieldSpec { return FieldSpec{c: c} }

// Attrs adds attributes to the control element itself (not its wrapper),
// merged as elements.Merge does. The field keeps the attributes that
// carry its value and state.
func (f FieldSpec) Attrs(a templ.Attributes) FieldSpec {
	f.attrs = elements.Merge(f.attrs, nil, a)
	return f
}

var fieldOwned = []string{"id", "name", "type", "required", "disabled", "aria-invalid", "value", "checked", "multiple", "data-ui-when-off"}

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
	case forms.KindCheckbox:
		return "checkbox"
	case forms.KindHidden:
		return "hidden"
	case forms.KindFile, forms.KindFiles:
		return "file"
	}
	return "text"
}

// isSelect reports whether the field renders as a <select>.
func (f FieldSpec) isSelect() bool {
	switch f.c.Kind() {
	case forms.KindSelect, forms.KindMultiSelect, forms.KindToggle:
		return true
	}
	return false
}

// value is the field's text: forms keeps one, the last non-blank one, for
// a field of one value.
func (f FieldSpec) value() string {
	if t := f.c.Text(); len(t) > 0 {
		return t[len(t)-1]
	}
	return ""
}

// ticked is the server's rule for a checkbox: a value that isn't one
// strconv.ParseBool reads as false.
func ticked(text string) bool {
	if text == "" {
		return false
	}
	b, err := strconv.ParseBool(text)
	return err != nil || b
}

func (f FieldSpec) hiddenWrapper() bool {
	// A field inside a group without a condition of its own follows its
	// group, whose fieldset hides and disables it.
	_, own := f.c.When()
	return !f.c.Active() && (!f.nested || own)
}

func (f FieldSpec) wrapperAttrs(class string) (templ.Attributes, error) {
	a := templ.Attributes{}
	if class != "" {
		a["class"] = class
	}
	if cond, ok := f.c.When(); ok {
		b, err := json.Marshal(cond)
		if err != nil {
			return nil, err
		}
		a["data-ui-when"] = string(b)
	}
	if f.hiddenWrapper() {
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
		// means.
		"required": f.c.Required(),
	}
	if f.hiddenWrapper() {
		// Disabled for its condition: the marker tells the ui script it may
		// enable it again, which it never does for a control the app
		// disabled itself.
		own["disabled"] = true
		own["data-ui-when-off"] = ""
	}
	switch {
	case f.isSelect():
		if f.c.Kind() == forms.KindMultiSelect {
			own["multiple"] = true
		}
	case f.c.Kind() == forms.KindMultiline:
	default:
		own["type"] = typ
		if ltrTypes[typ] {
			own["dir"] = "ltr"
		}
		switch typ {
		case "checkbox":
			own["value"] = "true"
			own["checked"] = ticked(f.value())
		case "file":
			own["multiple"] = f.c.Kind() == forms.KindFiles
		case "password":
		default:
			if v := f.value(); v != "" {
				own["value"] = v
			}
		}
	}
	// forms' Placeholder falls back to the label. ui's fields always show
	// their label, so render one only on text inputs, and only when it says
	// something else (spike finding 12).
	if textTypes[typ] && !f.isSelect() || f.c.Kind() == forms.KindMultiline {
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
	// The control's own attributes (maxlength, min, pattern …), then the
	// caller's; neither may change what carries the value and state.
	fromControl := templ.Attributes{}
	for k, v := range f.c.Attrs() {
		fromControl[k] = v
	}
	return elements.Merge(elements.Merge(own, fieldOwned, fromControl), fieldOwned, f.attrs)
}

// choices are the options of a select, a toggle's Yes and No included,
// and whether an empty first option is needed.
func (f FieldSpec) choices(ctx context.Context) (opts []forms.Choice, empty bool) {
	if f.c.Kind() == forms.KindToggle {
		v := f.value()
		b, err := strconv.ParseBool(v)
		set := v != "" && err == nil
		if v == "on" {
			b, set = true, true
		}
		opts = []forms.Choice{
			{Value: "true", Label: YesText, Selected: set && b},
			{Value: "false", Label: NoText, Selected: set && !b},
		}
		return opts, true
	}
	for o := range f.c.Options() {
		opts = append(opts, o)
	}
	// Always the same first option, fresh or re-rendered, unless the
	// options have an empty value of their own; required makes the browser
	// ask for another.
	empty = f.c.Kind() == forms.KindSelect && !slices.ContainsFunc(opts, func(o forms.Choice) bool { return o.Value == "" })
	return opts, empty
}

// textareaValue keeps a value's leading newline: the HTML parser drops the
// first newline after <textarea>.
func (f FieldSpec) textareaValue() string { return "\n" + f.value() }

// Render renders the field.
func (f FieldSpec) Render(ctx context.Context, w io.Writer) error {
	switch f.c.Kind() {
	case forms.KindList:
		return fmt.Errorf("inputs: Field(%q): lists render with a repeat (design §6), not in the spike", f.c.Path())
	case forms.KindGroup:
		wrapper, err := f.wrapperAttrs("ui-group")
		if err != nil {
			return err
		}
		if f.hiddenWrapper() {
			wrapper["disabled"] = true // a fieldset disables what it holds
			wrapper["data-ui-when-off"] = ""
		}
		var children []FieldSpec
		for c := range f.c.Children() {
			children = append(children, FieldSpec{c: c, nested: true})
		}
		return group(f, wrapper, children).Render(ctx, w)
	case forms.KindHidden:
		wrapper, err := f.wrapperAttrs("")
		if err != nil {
			return err
		}
		return hidden(f, wrapper).Render(ctx, w)
	}
	wrapper, err := f.wrapperAttrs("ui-field")
	if err != nil {
		return err
	}
	return field(f, wrapper).Render(ctx, w)
}
