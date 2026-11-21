package forms

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"regexp"
	"strconv"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
)

// Form holds a form's nodes (fields, groups and lists) in declaration
// order and their state for one request. Embed *Form in a struct of typed
// handles; see the package documentation. A group, and each item of a
// list, is a *Form too: a sub-form.
type Form struct {
	parent *Form  // nil for the root; for a list item, the list's form
	item   *list  // set for a list item
	name   string // a group's name
	index  int    // a list item's index, renumbered after Bind

	nodes  []any // *field, *Form (a group), *list
	names  map[string]bool
	checks []func(context.Context) error

	label    i18n.Ter
	labelSet bool
	when     *condNode

	building bool // a list item or template row being built: may declare after Bind

	// Root only.
	bound     bool
	validated bool

	err error // the form-level error: unreadable, a check, Fail
}

// New returns an empty root form.
func New() *Form {
	return &Form{names: map[string]bool{}}
}

var fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (f *Form) root() *Form {
	for f.parent != nil {
		f = f.parent
	}
	return f
}

// isBound reports whether the form was bound. A list item or template row
// being built, and the forms inside it, may still be declared; nothing
// outside it may.
func (f *Form) isBound() bool {
	for g := f; g != nil; g = g.parent {
		if g.building {
			return false
		}
	}
	return f.root().bound
}

func (f *Form) path() string {
	switch {
	case f.parent == nil:
		return ""
	case f.item != nil:
		if f.index < 0 {
			return f.item.path() + "[" + IndexPlaceholder + "]"
		}
		return f.item.path() + "[" + strconv.Itoa(f.index) + "]"
	default:
		return join(f.parent.path(), f.name)
	}
}

func join(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}

// register checks a node's name and records it; it panics on a malformed
// or duplicate name, or after Bind.
func (f *Form) register(name, what string) {
	if f.isBound() {
		panic(fmt.Sprintf("forms: %s %q declared after Bind", what, name))
	}
	if !fieldName.MatchString(name) {
		panic(fmt.Sprintf("forms: %s name %q: a name is lower-case letters, digits and _, starting with a letter", what, name))
	}
	if f.names[name] {
		panic(fmt.Sprintf("forms: %q declared twice", name))
	}
	f.names[name] = true
}

func (f *Form) add(fd *field) {
	f.register(fd.name, "field")
	fd.form = f
	f.nodes = append(f.nodes, fd)
}

func (f *Form) mustBeRoot(method string) {
	if f.parent != nil {
		panic(fmt.Sprintf("forms: %s on the sub-form %q; call it on the root form", method, f.path()))
	}
}

// Bind records the submitted text of every field, in declaration order,
// decides which are present, and runs their conversions. A conversion
// failure becomes that field's error; no rule runs. Names the form did not
// declare are ignored. A second Bind, or Bind on a sub-form, panics.
func (f *Form) Bind(src Source) {
	f.mustBeRoot("Bind")
	if f.bound {
		panic("forms: Bind called twice")
	}
	if src == nil {
		panic("forms: Bind(nil)")
	}
	f.bound = true
	f.bindNodes(src)
}

// bindNodes binds f's nodes and reports whether any field saw input.
func (f *Form) bindNodes(src Source) bool {
	seen := false
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			if n.bind(src) {
				seen = true
			}
		case *Form:
			if n.bindNodes(src) {
				seen = true
			}
		case *list:
			if n.bind(src) {
				seen = true
			}
		}
	}
	return seen
}

// Validate runs every field's checks, then the checks of groups and of the
// form, and reports whether the form is valid (interface §5.1, §9.2). It
// panics before Bind, and on a sub-form.
func (f *Form) Validate(ctx context.Context) bool {
	f.mustBeRoot("Validate")
	if !f.bound {
		panic("forms: Validate before Bind")
	}
	// Activity first, all of it read before anything is cleared; then
	// every inactive node is cleared before any rule runs.
	memo := map[any]bool{}
	f.resolve(memo)
	f.clearInactive(memo)
	f.validateNodes(ctx, memo)
	f.validated = true
	return f.Valid()
}

func (f *Form) resolve(memo map[any]bool) {
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			n.active(memo)
		case *Form:
			n.active(memo)
			n.resolve(memo)
		case *list:
			n.active(memo)
			for _, it := range n.items {
				it.form.resolve(memo)
			}
		}
	}
}

func (f *Form) clearInactive(memo map[any]bool) {
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			if !n.active(memo) {
				n.clear()
			}
		case *Form:
			if !n.active(memo) {
				n.err = nil
			}
			n.clearInactive(memo)
		case *list:
			if !n.active(memo) {
				n.items, n.err = nil, nil
				continue
			}
			for _, it := range n.items {
				it.form.clearInactive(memo)
			}
		}
	}
}

// validateNodes validates f's active nodes and reports whether one failed.
func (f *Form) validateNodes(ctx context.Context, memo map[any]bool) bool {
	failed := false
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			if n.active(memo) && n.validate(ctx) != nil {
				failed = true
			}
		case *Form:
			if n.active(memo) && n.validateNodes(ctx, memo) {
				failed = true
			}
		case *list:
			if n.active(memo) && n.validate(ctx, memo) {
				failed = true
			}
		}
	}
	for _, check := range f.checks {
		// A check that Fails a field or the form and returns nil still
		// stops the rest.
		if failed || f.hasError() {
			break
		}
		if err := check(ctx); err != nil {
			f.err = err
		}
	}
	return failed || f.hasError()
}

func (f *Form) hasError() bool {
	for range f.Errors() {
		return true
	}
	return false
}

// Submit reads r, binds and validates. An unreadable request records
// ErrUnreadable on the form and returns false.
func (f *Form) Submit(ctx context.Context, r *http.Request) bool {
	f.mustBeRoot("Submit")
	if f.bound {
		panic("forms: Submit after Bind")
	}
	src, err := FromRequest(r)
	if err != nil {
		f.Bind(FromValues(nil)) // no initial value survives
		f.validated = true
		f.err = ErrUnreadable.Wrap(err)
		return false
	}
	f.Bind(src)
	return f.Validate(ctx)
}

// Valid reports whether Validate ran and no error remains in this form or
// sub-form.
func (f *Form) Valid() bool {
	return f.root().validated && f.Err() == nil
}

// Fail sets the form's own error, such as one from the service layer,
// replacing any earlier one. Valid is then false.
func (f *Form) Fail(err error) {
	if err == nil {
		panic("forms: Fail(nil)")
	}
	f.err = err
}

// Check adds a check that belongs to no single field. A form's checks run
// in order once every node in it passed; the first failure is the form's
// own error. A group's checks run when the group is active.
func (f *Form) Check(fn func(ctx context.Context) error) {
	if fn == nil {
		panic("forms: Check(nil)")
	}
	if f.isBound() {
		panic("forms: Check after Bind")
	}
	f.checks = append(f.checks, fn)
}

// Label sets a group's label. The default is its name. On the root it is
// legal and unused; a list item takes its list's label, so Label on one
// panics.
func (f *Form) Label(l i18n.Ter) *Form {
	if l == nil {
		panic("forms: Label(nil) on a form")
	}
	if f.item != nil {
		panic(fmt.Sprintf("forms: Label on the list item %q; label the list", f.path()))
	}
	if f.isBound() {
		panic("forms: Label after Bind")
	}
	f.label, f.labelSet = l, true
	return f
}

// When makes a group active only while c holds; an inactive group's
// fields are cleared at Validate. On the root it is legal and has no
// effect: the root is always active. On a list item it panics (put the
// condition on the item's fields or on the list); so does When twice, a
// zero Cond, or a condition that closes a cycle.
func (f *Form) When(c Cond) *Form {
	if f.item != nil {
		panic(fmt.Sprintf("forms: When on the list item %q; put the condition on its fields or on the list", f.path()))
	}
	if f.isBound() {
		panic("forms: When after Bind")
	}
	f.when = checkWhen(f, f.path(), f.when, c)
	return f
}

// Active reports whether the form's conditions, and those of every form
// around it, hold. The root form is always active.
func (f *Form) Active() bool { return f.active(map[any]bool{}) }

func (f *Form) active(memo map[any]bool) bool {
	if f.parent == nil {
		return true
	}
	if v, ok := memo[f]; ok {
		return v
	}
	var v bool
	if f.item != nil && f.index < 0 {
		// A template row: its fields report their own conditions, so a
		// client can clone it while the list is still hidden.
		v = true
	} else {
		v = f.parent.active(memo)
		if v && f.item != nil {
			v = f.item.active(memo)
		}
	}
	if v && f.when != nil {
		v = f.when.eval(memo)
	}
	memo[f] = v
	return v
}

// Err is nil when the form has no error. Otherwise it joins one
// *FieldError per failing node: the form's own error first, then nodes in
// declaration order, groups and list items in place.
func (f *Form) Err() error {
	var errs []error
	for e := range f.Errors() {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// Errors yields what Err joins, in the same order.
func (f *Form) Errors() iter.Seq[*FieldError] {
	return func(yield func(*FieldError) bool) { f.yieldErrors(yield) }
}

func (f *Form) yieldErrors(yield func(*FieldError) bool) bool {
	if f.err != nil {
		var label i18n.Ter
		if f.parent != nil {
			label = f.groupLabel()
		}
		if !yield(&FieldError{Path: f.path(), Label: label, Err: f.err}) {
			return false
		}
	}
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			if n.err != nil && !yield(&FieldError{Path: n.path(), Label: n.label, Err: n.err}) {
				return false
			}
		case *Form:
			if !n.yieldErrors(yield) {
				return false
			}
		case *list:
			if n.err != nil && !yield(&FieldError{Path: n.path(), Label: n.label, Err: n.err}) {
				return false
			}
			for _, it := range n.items {
				if !it.form.yieldErrors(yield) {
					return false
				}
			}
		}
	}
	return true
}

func (f *Form) groupLabel() i18n.Ter {
	switch {
	case f.label != nil:
		return f.label
	case f.item != nil:
		return f.item.label
	default:
		return i18n.Static(f.name)
	}
}
