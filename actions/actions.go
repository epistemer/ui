// Package actions declares, binds, resolves and describes what can be done
// to a record (server-state-interface.md §2, ADR 0009). It renders nothing:
// ui/components renders the items, and an app can render them itself.
//
//	var Refund = actions.New(L10nRefund).Danger().Confirm(L10nRefundQuestion)
//
//	items := actions.Resolve(ctx,
//		View.To(adminroutes.Order(o.ID)),
//		Refund.To(adminroutes.OrderRefund(o.ID)).Unless(shop.CanRefund(o)),
//	)
//
// An action is shown only when the request's gate admits its link, and is
// unavailable, with a reason, when the record's state forbids it.
package actions

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/links"
)

// Kind is how an item runs, taken from its link, never chosen by a caller.
type Kind int

const (
	Navigate Kind = iota // a GET link
	Request              // any other method: a form post, or htmx's verb
	Open                 // a GET whose answer shows in the action dialog
)

// Action is declared once, at package level. Every builder returns a new
// action, so two actions built from one base are two for Item.Is.
type Action struct {
	id       *byte
	label    i18n.Ter
	icon     templ.Component
	danger   bool
	primary  bool
	question i18n.Ter
	dialog   bool
	attrs    templ.Attributes
}

func blank(t i18n.Ter) bool {
	return t == nil || strings.TrimSpace(t.T(context.Background())) == ""
}

// New declares an action labelled label. A blank label panics.
func New(label i18n.Ter) Action {
	if blank(label) {
		panic("actions: New with no label")
	}
	return Action{id: new(byte), label: label}
}

func (a Action) next() Action { a.id = new(byte); return a }

// Icon sets the icon shown before the label.
func (a Action) Icon(c templ.Component) Action { a = a.next(); a.icon = c; return a }

// Danger marks a destructive action. With Primary it panics.
func (a Action) Danger() Action {
	if a.primary {
		panic("actions: Danger on a Primary action")
	}
	a = a.next()
	a.danger = true
	return a
}

// Primary marks the main action of a set. With Danger it panics.
func (a Action) Primary() Action {
	if a.danger {
		panic("actions: Primary on a Danger action")
	}
	a = a.next()
	a.primary = true
	return a
}

// Confirm asks question first. A blank question, or a Dialog action,
// panics: the dialog is its own question.
func (a Action) Confirm(question i18n.Ter) Action {
	switch {
	case blank(question):
		panic("actions: Confirm with no question")
	case a.dialog:
		panic("actions: Confirm on a Dialog action: the dialog is the question")
	}
	a = a.next()
	a.question = question
	return a
}

// Dialog makes it an Open action, whose answer shows in the action
// dialog. On an action with a question it panics.
func (a Action) Dialog() Action {
	if a.question != nil {
		panic("actions: Dialog on an action with a question")
	}
	a = a.next()
	a.dialog = true
	return a
}

// Attrs adds attributes to the control a placement renders.
func (a Action) Attrs(at templ.Attributes) Action {
	a = a.next()
	merged := templ.Attributes{}
	for k, v := range a.attrs {
		merged[k] = v
	}
	for k, v := range at {
		merged[k] = v
	}
	a.attrs = merged
	return a
}

// To binds the action to a link, per record, in a request.
func (a Action) To(l ui.Link) Bound { return Bound{action: a, link: l, shown: true} }

// Bound is an action with its link and its state rule.
type Bound struct {
	action Action
	link   ui.Link
	reason error
	shown  bool
}

// Unless makes the item unavailable, with err as the reason, when err
// isn't nil. err is what the handler's own check returns.
func (b Bound) Unless(err error) Bound { b.reason = err; return b }

// If leaves the item out when ok is false.
func (b Bound) If(ok bool) Bound { b.shown = b.shown && ok; return b }

// Item is what a placement reads, as forms.Control is for a field.
type Item struct {
	Label    i18n.Ter
	Icon     templ.Component // nil for none
	Kind     Kind
	Method   string        // GET, POST, PUT, PATCH, DELETE
	URL      templ.SafeURL // sanitised (visual-interface.md §2.7)
	Question i18n.Ter      // nil for none
	Danger   bool
	Primary  bool
	Reason   error // non-nil: the item is unavailable, and this says why
	Attrs    templ.Attributes
	id       *byte
}

// Is reports whether the item was bound from a.
func (it Item) Is(a Action) bool { return it.id != nil && it.id == a.id }

// Gate is set once per request; Resolve reads it. With handlers:
// actions.Gate.With(ctx, handlers.Gate(admit)).
var Gate = ui.NewProvider[func(ui.Link) bool](nil)

// AllowAll admits every link, for an app with no permissions.
func AllowAll(ui.Link) bool { return true }

// Allowed asks the request's gate about l: for a nav item, a tab, a link
// in a cell. No gate panics.
func Allowed(ctx context.Context, l ui.Link) bool {
	g := Gate.Value(ctx)
	if g == nil {
		panic("actions: no actions.Gate in the context: set it in a middleware, or to actions.AllowAll")
	}
	return g(l)
}

// method is a link that knows its method, as handlers' Link does.
type method interface{ Method() string }

// local reports whether an address is this site's: one leading slash,
// no backslash, white space or control character.
func local(u string) bool {
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") || strings.ContainsRune(u, '\\') {
		return false
	}
	for _, c := range u {
		if c <= ' ' || c == 0x7f {
			return false
		}
	}
	return true
}

// Resolve asks the gate about each link and describes what it admits, in
// order. An empty result is an empty, non-nil slice. An action with no
// address, If(false), or a Request or Open item whose address isn't
// local, is absent. A Dialog action bound to a link that isn't a GET
// panics, naming it.
func Resolve(ctx context.Context, bound ...Bound) []Item {
	if Gate.Value(ctx) == nil {
		panic("actions: no actions.Gate in the context: set it in a middleware, or to actions.AllowAll")
	}
	items := []Item{}
	for _, b := range bound {
		u := links.Href(b.link)
		if !b.shown || u == "" || !Allowed(ctx, b.link) {
			continue
		}
		m := "GET"
		if l, ok := b.link.(method); ok && l.Method() != "" {
			m = strings.ToUpper(l.Method())
		}
		kind := Navigate
		switch {
		case b.action.dialog && m != "GET":
			panic("actions: the Dialog action " + b.action.label.T(ctx) + " is bound to a " + m + " link; an Open action's link is a GET")
		case b.action.dialog:
			kind = Open
		case m != "GET":
			kind = Request
		}
		if kind != Navigate && !local(string(u)) {
			continue
		}
		items = append(items, Item{
			Label: b.action.label, Icon: b.action.icon, Kind: kind, Method: m, URL: u,
			Question: b.action.question, Danger: b.action.danger, Primary: b.action.primary,
			Reason: b.reason, Attrs: b.action.attrs, id: b.action.id,
		})
	}
	return items
}
