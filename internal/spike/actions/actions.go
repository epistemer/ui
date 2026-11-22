// Package actions is the spike of ui/actions (server-state-design.md §5):
// declare, bind, resolve and describe what can be done to a record. It
// renders nothing.
package actions

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/links"
)

// Kind is how an item runs, taken from its link.
type Kind int

const (
	Navigate Kind = iota // a GET link
	Request              // any other method
	Open                 // a GET whose answer shows in the action dialog
)

// Method is what a link that knows its method says. handlers' links will
// (design §10); ui.URL's don't, and are GETs.
type Method interface{ Method() string }

// Action is declared once.
type Action struct {
	id       *byte // its identity: copies made by builders share it
	label    i18n.Ter
	icon     templ.Component
	danger   bool
	primary  bool
	question i18n.Ter
	dialog   bool
	attrs    templ.Attributes
}

// New declares an action labelled label. A blank label panics.
func New(label i18n.Ter) Action {
	if label == nil || strings.TrimSpace(label.T(context.Background())) == "" {
		panic("actions: New with no label")
	}
	return Action{id: new(byte), label: label}
}

func (a Action) Icon(c templ.Component) Action { a.icon = c; return a }
func (a Action) Danger() Action                { a.danger = true; return a }
func (a Action) Primary() Action               { a.primary = true; return a }

// Confirm asks question first. With Dialog it panics.
func (a Action) Confirm(q i18n.Ter) Action {
	if a.dialog {
		panic("actions: Confirm on a Dialog action: the dialog is the question")
	}
	a.question = q
	return a
}

// Dialog makes it an Open action: its answer shows in the action dialog.
func (a Action) Dialog() Action {
	if a.question != nil {
		panic("actions: Dialog on an action with a question")
	}
	a.dialog = true
	return a
}

func (a Action) Attrs(at templ.Attributes) Action { a.attrs = at; return a }

// To binds the action to a link.
func (a Action) To(l ui.Link) Bound { return Bound{action: a, link: l, shown: true} }

// Bound is an action with its link and its state rule.
type Bound struct {
	action Action
	link   ui.Link
	reason error
	shown  bool
}

// Unless disables it, with err as the reason, when err isn't nil.
func (b Bound) Unless(err error) Bound { b.reason = err; return b }

// If leaves it out when ok is false.
func (b Bound) If(ok bool) Bound { b.shown = b.shown && ok; return b }

// Item is what a placement reads.
type Item struct {
	Label    i18n.Ter
	Icon     templ.Component // nil for none
	Kind     Kind
	Method   string // GET, POST, …
	URL      templ.SafeURL
	Question i18n.Ter // nil for none
	Danger   bool
	Primary  bool
	Disabled bool
	Reason   error // why it's disabled
	Attrs    templ.Attributes
	action   Action
}

// Is reports whether the item comes from a.
func (it Item) Is(a Action) bool {
	return it.action.label == a.label && it.action.dialog == a.dialog && it.action.danger == a.danger
}

// Gate is set once per request and read by Resolve.
var Gate = ui.NewProvider[func(ui.Link) bool](nil)

// AllowAll admits every link.
func AllowAll(ui.Link) bool { return true }

// Allowed asks the request's gate about l. No gate panics.
func Allowed(ctx context.Context, l ui.Link) bool {
	g := Gate.Value(ctx)
	if g == nil {
		panic("actions: no actions.Gate in the context: set it in a middleware, or to actions.AllowAll")
	}
	return g(l)
}

// Resolve asks the gate about each link and describes what it admits, in
// order. An action with no address, or If(false), is absent.
func Resolve(ctx context.Context, bound ...Bound) []Item {
	items := []Item{}
	for _, b := range bound {
		u := links.Href(b.link)
		if !b.shown || u == "" || !Allowed(ctx, b.link) {
			continue
		}
		method := "GET"
		if m, ok := b.link.(Method); ok && m.Method() != "" {
			method = strings.ToUpper(m.Method())
		}
		kind := Navigate
		switch {
		case b.action.dialog:
			kind = Open
		case method != "GET":
			kind = Request
		}
		items = append(items, Item{
			Label: b.action.label, Icon: b.action.icon, Kind: kind, Method: method, URL: u,
			Question: b.action.question, Danger: b.action.danger, Primary: b.action.primary,
			Disabled: b.reason != nil, Reason: b.reason, Attrs: b.action.attrs, action: b.action,
		})
	}
	return items
}

// ReasonText is the disabled item's reason for people.
func (it Item) ReasonText(ctx context.Context) string {
	if it.Reason == nil {
		return ""
	}
	return errors.T(ctx, it.Reason)
}
