package forms

import (
	"iter"
	"slices"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/listing"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/forms/internal/hook"
)

// Control is a read-only view of one node of a form (a field, a group, a
// list or one of its items) with everything a widget needs and nothing
// typed by T. ui/inputs renders forms through it.
type Control struct {
	fd *field
	f  *Form
	l  *list
}

// Control returns the field's view.
func (h *Field[T]) Control() Control { return Control{fd: h.use("Control")} }

// Control returns the list's view.
func (h *List[G]) Control() Control { return Control{l: h.l} }

// Control returns the form's own view: a group, a list item, or the root.
func (f *Form) Control() Control { return Control{f: f} }

// Controls yields the form's direct children, in declaration order.
func (f *Form) Controls() iter.Seq[Control] { return f.Control().Children() }

// Kind is the node's kind. A group, a list item and the root are
// KindGroup; a list is KindList.
func (c Control) Kind() Kind {
	switch {
	case c.fd != nil:
		return c.fd.kind
	case c.l != nil:
		return KindList
	case c.f != nil:
		return KindGroup
	}
	return 0
}

// Path is the node's name attribute: "name", "percentage.value",
// "constraints[2].type"; "" for the root.
func (c Control) Path() string {
	switch {
	case c.fd != nil:
		return c.fd.path()
	case c.l != nil:
		return c.l.path()
	case c.f != nil:
		return c.f.path()
	}
	return ""
}

// Label is the node's label: what was set, else its name; a list item's is
// its list's.
func (c Control) Label() i18n.Ter {
	switch {
	case c.fd != nil:
		return c.fd.label
	case c.l != nil:
		return c.l.label
	case c.f == nil || c.f.parent == nil:
		return i18n.Static("")
	}
	return c.f.groupLabel()
}

func (c Control) labelSet() bool {
	switch {
	case c.fd != nil:
		return c.fd.labelSet
	case c.l != nil:
		return c.l.labelSet
	case c.f != nil:
		return c.f.labelSet || c.f.item != nil
	}
	return false
}

// Placeholder is a field's placeholder, else its label.
func (c Control) Placeholder() i18n.Ter {
	if c.fd != nil && c.fd.placeholder != nil {
		return c.fd.placeholder
	}
	return c.Label()
}

// Help is a field's help text, or nil.
func (c Control) Help() i18n.Ter {
	if c.fd != nil {
		return c.fd.help
	}
	return nil
}

// Required reports whether a field is required, or a list needs an item.
func (c Control) Required() bool {
	switch {
	case c.fd != nil:
		return c.fd.required
	case c.l != nil:
		return c.l.min > 0
	}
	return false
}

// TwoState reports whether the control is a Checkbox: two states, never
// absent, whatever kind Hint gave it. Bool is three-state.
func (c Control) TwoState() bool {
	return c.fd != nil && c.fd.base.box
}

// Active reports whether the node's conditions, and those around it, hold
// now: on initial values before Bind, on submitted text after. A renderer
// hides and disables an inactive node.
func (c Control) Active() bool {
	memo := map[any]bool{}
	switch {
	case c.fd != nil:
		return c.fd.active(memo)
	case c.l != nil:
		return c.l.active(memo)
	case c.f != nil:
		return c.f.active(memo)
	}
	return false
}

// When returns the node's own condition, to serialise for the client.
func (c Control) When() (Cond, bool) {
	var n *condNode
	switch {
	case c.fd != nil:
		n = c.fd.when
	case c.l != nil:
		n = c.l.when
	case c.f != nil:
		n = c.f.when
	}
	return Cond{n}, n != nil
}

// Text is a field's text: what was submitted, else its formatted initial
// value. It is empty for files, groups and lists.
func (c Control) Text() []string {
	if c.fd == nil || c.fd.base.files {
		return nil
	}
	return slices.Clone(c.fd.text)
}

// Choice is one option of a select or radio group.
type Choice struct {
	Value    string
	Label    i18n.Ter
	Selected bool
}

// Options yields a field's choices, marking those its text holds.
func (c Control) Options() iter.Seq[Choice] {
	return func(yield func(Choice) bool) {
		if c.fd == nil {
			return
		}
		for _, o := range c.fd.options {
			v := o.Value()
			if !yield(Choice{Value: v, Label: o.Label(), Selected: slices.Contains(c.fd.text, v)}) {
				return
			}
		}
	}
}

// Attrs yields the HTML attributes a field's rules imply (maxlength, min,
// max, pattern, accept, type, data-ui-max-size) and inputmode="numeric" for
// an integer remote field. When two rules give one attribute, the later
// rule's value wins; attributes come in the order they first appear.
func (c Control) Attrs() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		if c.fd == nil {
			return
		}
		var names []string
		last := map[string]string{}
		if c.fd.kind == KindRemote && c.fd.base.integer {
			names = append(names, "inputmode")
			last["inputmode"] = "numeric"
		}
		for _, st := range c.fd.stages {
			for _, a := range st.attrs {
				if _, ok := last[a[0]]; !ok {
					names = append(names, a[0])
				}
				last[a[0]] = a[1]
			}
		}
		for _, n := range names {
			if !yield(n, last[n]) {
				return
			}
		}
	}
}

// Remote gives a remote field's search link and Labeler, and id: the
// value formatted, or "" when the field is absent or its text didn't
// convert. ok is false for any other field.
func (c Control) Remote() (search interface{ URL() string }, l listing.Labeler, id string, ok bool) {
	if c.fd == nil || c.fd.kind != KindRemote {
		return nil, nil, "", false
	}
	if c.fd.present && c.fd.convAt < 0 && len(c.fd.vals) > 0 {
		if t := c.fd.base.format(c.fd.vals[0]); len(t) > 0 {
			id = t[0]
		}
	}
	return c.fd.search, c.fd.labeler, id, true
}

// Stored yields each stored file with its keep state: posted, after Bind;
// true before. It is an empty iterator, never nil, for any other control.
func (c Control) Stored() iter.Seq2[files.Stored, bool] {
	return func(yield func(files.Stored, bool) bool) {
		if c.fd == nil || !c.fd.base.upload {
			return
		}
		for _, s := range c.fd.stored {
			kept := c.fd.keep == nil || c.fd.keep[s.Key]
			if !yield(s, kept) {
				return
			}
		}
	}
}

// Lost is true for a file field (File, Files, Upload, Uploads) that was
// posted files in a form that failed: the browser won't send them again.
func (c Control) Lost() bool {
	if c.fd == nil || !c.fd.base.files || !c.fd.postedFiles || c.fd.err != nil {
		return false
	}
	return c.Active() && c.fd.form.root().hasError()
}

// Bounds gives a list's effective bounds: max is DefaultMaxItems when
// MaxItems wasn't set. Zero for any other control.
func (c Control) Bounds() (min, max int) {
	if c.l == nil {
		return 0, 0
	}
	return c.l.min, c.l.max
}

// Err is the node's own error.
func (c Control) Err() error {
	switch {
	case c.fd != nil:
		return c.fd.err
	case c.l != nil:
		return c.l.err
	case c.f != nil:
		return c.f.err
	}
	return nil
}

// Children yields a group's members, or a list's items as groups. For a
// list that was neither bound nor given items by Add, it yields MinItems
// blank items built at indices 0..MinItems-1.
func (c Control) Children() iter.Seq[Control] {
	return func(yield func(Control) bool) {
		switch {
		case c.l != nil:
			for _, it := range c.l.displayItems() {
				if !yield(Control{f: it.form}) {
					return
				}
			}
		case c.f != nil:
			for _, n := range c.f.nodes {
				var child Control
				switch n := n.(type) {
				case *field:
					child = Control{fd: n}
				case *Form:
					child = Control{f: n}
				case *list:
					child = Control{l: n}
				}
				if !yield(child) {
					return
				}
			}
		}
	}
}

// Template returns a list's blank row, built at IndexPlaceholder, for a
// client to clone. It is built on first use, so like the rest of a form
// it is not safe for concurrent use.
func (c Control) Template() (Control, bool) {
	if c.l == nil {
		return Control{}, false
	}
	return Control{f: c.l.templateRow().form}, true
}

func init() {
	hook.LabelSet = func(c any) bool { return c.(Control).labelSet() }
}
