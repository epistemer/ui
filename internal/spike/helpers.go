package spike

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
)

var (
	itemText   = i18n.Key("ui.inputs.list.item", "Item %d")
	removeItem = i18n.Key("ui.inputs.list.remove", "Remove item %d")
	addItem    = i18n.Key("ui.inputs.list.add", "Add to %s")
	added      = i18n.Key("ui.inputs.list.added", "Item %d added")
	removed    = i18n.Key("ui.inputs.list.removed", "Item %d removed")
	atMost     = i18n.Key("ui.inputs.list.at_most", "Maximum: %d")
	atLeast    = i18n.Key("ui.inputs.list.at_least", "Minimum: %d")
	clearText  = i18n.Key("ui.inputs.combobox.clear", "Clear %s")
	results    = i18n.Key("ui.inputs.combobox.results", "Results: %d")
	none       = i18n.Key("ui.inputs.combobox.none", "No matches")
	failedText = i18n.Key("ui.inputs.combobox.failed", "Couldn't search. Try again.")
	more       = i18n.Key("ui.inputs.combobox.more", "Keep typing to narrow these down")
	keepText   = i18n.Key("ui.inputs.upload.keep", "Keep")
	removeFile = i18n.Key("ui.inputs.upload.remove", "Remove %s")
	undo       = i18n.Key("ui.inputs.upload.undo", "Undo")
	chosenText = i18n.Key("ui.inputs.upload.chosen", "%s chosen")
	again      = i18n.Key("ui.inputs.upload.again", "Choose the files again")
	tooLarge   = i18n.Key("ui.inputs.upload.too_large", "%s is larger than %s")
	wrongType  = i18n.Key("ui.inputs.upload.wrong_type", "%s isn't a type this field takes")
	bodyLarge  = i18n.Key("ui.inputs.upload.body_too_large", "These files are too large to send together. Send fewer at a time.")
)

func orNothing(c templ.Component) templ.Component {
	if c == nil {
		return templ.NopComponent
	}
	return c
}

func fail(err error) templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error { return err })
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func count(c forms.Control) int {
	n := 0
	for range c.Children() {
		n++
	}
	return n
}

func errText(ctx context.Context, err error) string { return errors.T(ctx, err) }

// texts renders the script's texts, translated, with their verbs kept for
// the script to fill (§3.2, ADR 0005).
func texts(ctx context.Context, m map[string]i18n.Ter) string {
	out := make(map[string]string, len(m))
	for k, t := range m {
		out[k] = t.T(ctx)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func listTexts(ctx context.Context) string {
	return texts(ctx, map[string]i18n.Ter{"item": itemText, "remove": removeItem, "added": added, "removed": removed, "atMost": atMost, "atLeast": atLeast})
}

func uploadTexts(ctx context.Context) string {
	return texts(ctx, map[string]i18n.Ter{"remove": removeFile, "undo": undo, "chosen": chosenText, "tooLarge": tooLarge, "wrongType": wrongType, "bodyTooLarge": bodyLarge})
}

// chosen is the name shown for the field's id (§3.2): looked up only for
// text that converts; a miss shows the id; an error fails the render.
func chosen(ctx context.Context, c forms.Control, r remote, text string) (string, error) {
	if text == "" {
		return "", nil
	}
	if _, err := strconv.ParseInt(text, 10, 64); err != nil {
		return "", nil // Control.Remote would give no id
	}
	name, ok, err := labelOf(ctx, r.l, text)
	if err != nil {
		return "", err
	}
	if !ok {
		return text, nil
	}
	return name, nil
}

func comboAttrs(ctx context.Context, c forms.Control, r remote) templ.Attributes {
	a := templ.Attributes{"data-ui-texts": texts(ctx, map[string]i18n.Ter{
		"clear":   i18n.Static(strings.ReplaceAll(clearText.T(ctx, c.Label()), "%", "%%")),
		"results": results, "none": none, "failed": failedText, "more": more,
	})}
	if actions.Allowed(ctx, r.search) {
		a["data-ui-search"] = r.search.URL()
	}
	return a
}

func inputAttrs(ctx context.Context, c forms.Control, text string) templ.Attributes {
	described := []string{fieldID.Part(ctx, "chosen")}
	a := templ.Attributes{
		"id": fieldID.Get(ctx), "class": "ui-input", "type": "text", "name": c.Path(),
		"value": text, "inputmode": "numeric", "autocomplete": "off",
	}
	if c.Err() != nil {
		described = append(described, fieldID.Part(ctx, "error"))
		a["aria-invalid"] = "true"
	}
	if c.Required() {
		a["required"] = true
	}
	a["aria-describedby"] = strings.Join(described, " ")
	return a
}

func describedBy(ctx context.Context, lost bool) string {
	d := fieldID.Part(ctx, "help")
	if lost {
		d += " " + fieldID.Part(ctx, "lost")
	}
	return d
}

// resultsFor is the filter's bound value, for the stand-in results.
func resultsFor(f *forms.Form) string {
	for c := range f.Controls() {
		for g := range c.Children() {
			return first(g.Text())
		}
	}
	return ""
}
