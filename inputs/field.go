// Package inputs renders forms' controls as fields (visual-interface.md
// §6): one widget per kind, with its label, help and error wired by
// generated ids, and its condition handed to the ui script, which shows,
// hides and disables it as the server would.
//
//	for c := range f.Controls() {
//		@inputs.Field(c)
//	}
package inputs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/internal/links"
)

var fieldID = ui.NewID("field")

// ui's own text (§2.5, §11; rich-inputs-interface.md §7).
var (
	// choose is a select's empty first option. Without one, a browser shows
	// the first option chosen when nothing is, and the page and the server
	// disagree about the value (design finding 11).
	choose       = i18n.Key("ui.inputs.field.choose", "Choose…")
	yes          = i18n.Key("ui.inputs.field.yes", "Yes")
	no           = i18n.Key("ui.inputs.field.no", "No")
	errorWord    = i18n.Key("ui.inputs.field.error", "Error:")
	showPassword = i18n.Key("ui.inputs.field.show_password", "Show password")

	comboboxClear   = i18n.Key("ui.inputs.combobox.clear", "Clear %s")
	comboboxResults = i18n.Key("ui.inputs.combobox.results", "Results: %d")
	comboboxNone    = i18n.Key("ui.inputs.combobox.none", "No matches")
	comboboxFailed  = i18n.Key("ui.inputs.combobox.failed", "Couldn't search. Try again.")
	comboboxMore    = i18n.Key("ui.inputs.combobox.more", "Keep typing to narrow these down")

	uploadKeep         = i18n.Key("ui.inputs.upload.keep", "Keep %s")
	uploadRemove       = i18n.Key("ui.inputs.upload.remove", "Remove %s")
	uploadUndo         = i18n.Key("ui.inputs.upload.undo", "Undo")
	uploadChosen       = i18n.Key("ui.inputs.upload.chosen", "%s chosen")
	uploadAgain        = i18n.Key("ui.inputs.upload.again", "Choose the files again")
	uploadTooLarge     = i18n.Key("ui.inputs.upload.too_large", "%s is larger than %s")
	uploadWrongType    = i18n.Key("ui.inputs.upload.wrong_type", "%s isn't a type this field takes")
	uploadBodyTooLarge = i18n.Key("ui.inputs.upload.body_too_large", "These files are too large to send together. Send fewer at a time.")

	listItem    = i18n.Key("ui.inputs.list.item", "Item %d")
	listAdd     = i18n.Key("ui.inputs.list.add", "Add to %s")
	listRemove  = i18n.Key("ui.inputs.list.remove", "Remove item %d")
	listAdded   = i18n.Key("ui.inputs.list.added", "Item %d added")
	listRemoved = i18n.Key("ui.inputs.list.removed", "Item %d removed")
	listAtMost  = i18n.Key("ui.inputs.list.at_most", "Maximum: %d")
	listAtLeast = i18n.Key("ui.inputs.list.at_least", "Minimum: %d")
)

type labelKey struct {
	l  listing.Labeler
	id string
}

type labelVal struct {
	name  string
	found bool
}

type labelCache struct {
	mu sync.Mutex
	m  map[labelKey]labelVal
}

var labelCaches = ui.NewProvider[*labelCache](nil)

// WithLabelCache returns ctx with a cache of LabelOf answers by Labeler
// and id, for the request (R10). Without one, each field calls once. A
// Labeler whose dynamic type isn't comparable (a func adapter) is never
// cached.
func WithLabelCache(ctx context.Context) context.Context {
	if labelCaches.Value(ctx) != nil {
		return ctx
	}
	return labelCaches.With(ctx, &labelCache{m: map[labelKey]labelVal{}})
}

func labelOf(ctx context.Context, l listing.Labeler, id string) (string, bool, error) {
	c := labelCaches.Value(ctx)
	if c == nil || l == nil || !reflect.ValueOf(l).Comparable() {
		return l.LabelOf(ctx, id)
	}
	k := labelKey{l, id}
	c.mu.Lock()
	v, ok := c.m[k]
	c.mu.Unlock()
	if ok {
		return v.name, v.found, nil
	}
	name, found, err := l.LabelOf(ctx, id)
	if err != nil {
		return name, found, err
	}
	c.mu.Lock()
	c.m[k] = labelVal{name, found}
	c.mu.Unlock()
	return name, found, nil
}

// FieldSpec is a form field being described: a widget for its control.
type FieldSpec struct {
	c        forms.Control
	attrs    templ.Attributes
	nested   bool // inside a group: the group's fieldset hides and disables it
	previews bool
}

// Field starts a widget for c. Its kind picks the element:
//   - <input> of the right type; a checkbox before its label;
//   - <textarea>, <select> (multiple for a multiselect);
//   - a Yes/No select for a three-state toggle, which a checkbox can't
//     express (it has no "no answer");
//   - <ui-combobox> for a remote select;
//   - the upload shape for File, Files, Upload and Uploads;
//   - a <fieldset> for a group, holding its children;
//   - a <fieldset> with <ui-repeat> for a list;
//   - <input type="hidden"> for a hidden field.
func Field(c forms.Control) FieldSpec { return FieldSpec{c: c} }

// Previews shows thumbnails of chosen images (KindUpload, KindUploads,
// KindFile, KindFiles), marking the field data-ui-previews; the app's CSP
// must allow img-src blob: (R17). On any other kind it panics.
func (f FieldSpec) Previews() FieldSpec {
	if !f.isUpload() {
		panic(fmt.Sprintf("inputs: Previews on %q (%s): only file and upload fields show previews", f.c.Path(), f.c.Kind()))
	}
	f.previews = true
	return f
}

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
	case forms.KindFile, forms.KindFiles, forms.KindUpload, forms.KindUploads:
		return "file"
	}
	return "text"
}

func (f FieldSpec) isUpload() bool {
	switch f.c.Kind() {
	case forms.KindFile, forms.KindFiles, forms.KindUpload, forms.KindUploads:
		return true
	}
	return false
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
	var ownClass = "ui-input"
	if f.isUpload() {
		ownClass = "ui-input ui-upload-input"
	}
	own := templ.Attributes{
		"class": ownClass,
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
			own["multiple"] = f.c.Kind() == forms.KindFiles || f.c.Kind() == forms.KindUploads
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
	if f.c.Kind() == forms.KindRemote {
		described = append(described, fieldID.Part(ctx, "chosen"))
	}
	if f.c.Help() != nil {
		described = append(described, fieldID.Part(ctx, "help"))
	}
	if f.c.Lost() {
		described = append(described, fieldID.Part(ctx, "lost"))
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
			{Value: "true", Label: yes, Selected: set && b},
			{Value: "false", Label: no, Selected: set && !b},
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

// hiddenAttrs are a hidden field's: its value, its state, then the
// control's own attributes and the caller's, which can't change them.
func (f FieldSpec) hiddenAttrs() templ.Attributes {
	own := templ.Attributes{"type": "hidden", "name": f.c.Path(), "value": f.value()}
	if f.hiddenWrapper() {
		own["disabled"] = true
		own["data-ui-when-off"] = ""
	}
	fromControl := templ.Attributes{}
	for k, v := range f.c.Attrs() {
		fromControl[k] = v
	}
	return elements.Merge(elements.Merge(own, fieldOwned, fromControl), fieldOwned, f.attrs)
}

// textareaValue keeps a value's leading newline: the HTML parser drops the
// first newline after <textarea>.
func (f FieldSpec) textareaValue() string { return "\n" + f.value() }

func marshalTexts(m map[string]string) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (f FieldSpec) labelText(ctx context.Context) string {
	if l := f.c.Label(); l != nil {
		return l.T(ctx)
	}
	return ""
}

func (f FieldSpec) comboboxAttrs(ctx context.Context) templ.Attributes {
	a := templ.Attributes{
		"class": "ui-combobox",
		"id":    fieldID.Part(ctx, "combobox"),
		"data-ui-texts": marshalTexts(map[string]string{
			"clear":   comboboxClear.T(ctx, f.labelText(ctx)),
			"results": comboboxResults.T(ctx),
			"none":    comboboxNone.T(ctx),
			"failed":  comboboxFailed.T(ctx),
			"more":    comboboxMore.T(ctx),
		}),
	}
	search, _, _, _ := f.c.Remote()
	if actions.Allowed(ctx, search) {
		if u := links.Href(search); u != "" {
			s := string(u)
			a["data-ui-search"] = s
			a["hx-get"] = s
			a["hx-trigger"] = "ui-search"
			a["hx-params"] = "none"
			a["hx-target"] = "find .ui-combobox-options"
			a["hx-select"] = "[data-ui-suggestions] > li"
			a["hx-swap"] = "innerHTML"
			a["hx-sync"] = "this:replace"
		}
	}
	return a
}

type storedFile struct {
	file files.Stored
	kept bool
	href templ.SafeURL
}

func (f FieldSpec) storedFiles(ctx context.Context) []storedFile {
	var out []storedFile
	for s, kept := range f.c.Stored() {
		sf := storedFile{file: s, kept: kept}
		if s.Href != nil && actions.Allowed(ctx, s.Href) {
			sf.href = links.Href(s.Href)
		}
		out = append(out, sf)
	}
	return out
}

func (f FieldSpec) uploadTexts(ctx context.Context) string {
	size := ""
	for k, v := range f.c.Attrs() {
		if k == "data-ui-max-size" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				size = formatFileSize(n)
			}
		}
	}
	return marshalTexts(map[string]string{
		"remove":       uploadRemove.T(ctx),
		"undo":         uploadUndo.T(ctx),
		"chosen":       uploadChosen.T(ctx),
		"tooLarge":     fmt.Sprintf(uploadTooLarge.T(ctx), "%s", size),
		"wrongType":    uploadWrongType.T(ctx),
		"bodyTooLarge": uploadBodyTooLarge.T(ctx),
	})
}

func formatFileSize(b int64) string {
	const (
		gb = 1024 * 1024 * 1024
		mb = 1024 * 1024
		kb = 1024
	)
	switch {
	case b > 0 && b%gb == 0:
		return fmt.Sprintf("%d GB", b/gb)
	case b > 0 && b%mb == 0:
		return fmt.Sprintf("%d MB", b/mb)
	case b > 0 && b%kb == 0:
		return fmt.Sprintf("%d KB", b/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func (f FieldSpec) listTexts(ctx context.Context) string {
	return marshalTexts(map[string]string{
		"item":    listItem.T(ctx),
		"remove":  listRemove.T(ctx),
		"added":   listAdded.T(ctx),
		"removed": listRemoved.T(ctx),
		"atMost":  listAtMost.T(ctx),
		"atLeast": listAtLeast.T(ctx),
	})
}

type listItemData struct {
	n        int
	children []FieldSpec
}

func itemChildren(c forms.Control) []FieldSpec {
	var out []FieldSpec
	for ch := range c.Children() {
		out = append(out, FieldSpec{c: ch, nested: true})
	}
	return out
}

// Render renders the field.
func (f FieldSpec) Render(ctx context.Context, w io.Writer) error {
	switch f.c.Kind() {
	case forms.KindList:
		wrapper, err := f.wrapperAttrs("ui-list")
		if err != nil {
			return err
		}
		if f.hiddenWrapper() {
			wrapper["disabled"] = true
			wrapper["data-ui-when-off"] = ""
		}
		var items []listItemData
		n := 0
		for it := range f.c.Children() {
			n++
			items = append(items, listItemData{n: n, children: itemChildren(it)})
		}
		var tpl []FieldSpec
		if t, ok := f.c.Template(); ok {
			tpl = itemChildren(t)
		}
		return list(f, items, tpl, func(ctx context.Context) templ.Attributes {
			own := wrapper
			var described []string
			if f.c.Help() != nil {
				described = append(described, fieldID.Part(ctx, "help"))
			}
			if f.c.Err() != nil {
				described = append(described, fieldID.Part(ctx, "error"))
			}
			if len(described) > 0 {
				own = elements.Merge(own, nil, templ.Attributes{"aria-describedby": strings.Join(described, " ")})
			}
			return elements.Merge(own, []string{"disabled", "hidden", "data-ui-when", "data-ui-when-off"}, f.attrs)
		}).Render(ctx, w)
	case forms.KindGroup:
		wrapper, err := f.wrapperAttrs("ui-group")
		if err != nil {
			return err
		}
		if f.hiddenWrapper() {
			wrapper["disabled"] = true // a fieldset disables what it holds
			wrapper["data-ui-when-off"] = ""
		}
		children := itemChildren(f.c)
		// Attrs go to the group's fieldset, after its own; it keeps what
		// carries its state.
		return group(f, children, func(ctx context.Context) templ.Attributes {
			own := wrapper
			var described []string
			if f.c.Help() != nil {
				described = append(described, fieldID.Part(ctx, "help"))
			}
			if f.c.Err() != nil {
				described = append(described, fieldID.Part(ctx, "error"))
			}
			if len(described) > 0 {
				own = elements.Merge(own, nil, templ.Attributes{"aria-describedby": strings.Join(described, " ")})
			}
			return elements.Merge(own, []string{"disabled", "hidden", "data-ui-when", "data-ui-when-off"}, f.attrs)
		}).Render(ctx, w)
	case forms.KindHidden:
		wrapper, err := f.wrapperAttrs("")
		if err != nil {
			return err
		}
		return hidden(f, wrapper).Render(ctx, w)
	case forms.KindRemote:
		_, l, id, _ := f.c.Remote()
		var chosen string
		if id != "" {
			name, ok, err := labelOf(ctx, l, id)
			if err != nil {
				return err
			}
			if ok {
				chosen = name
			} else {
				chosen = id
			}
		}
		wrapper, err := f.wrapperAttrs("ui-field")
		if err != nil {
			return err
		}
		return remoteField(f, wrapper, chosen).Render(ctx, w)
	case forms.KindFile, forms.KindFiles, forms.KindUpload, forms.KindUploads:
		wrapper, err := f.wrapperAttrs("ui-field ui-upload")
		if err != nil {
			return err
		}
		wrapper["data-ui-texts"] = f.uploadTexts(ctx)
		if f.previews {
			wrapper["data-ui-previews"] = ""
		}
		return uploadField(f, wrapper, f.storedFiles(ctx)).Render(ctx, w)
	}
	wrapper, err := f.wrapperAttrs("ui-field")
	if err != nil {
		return err
	}
	return field(f, wrapper).Render(ctx, w)
}
