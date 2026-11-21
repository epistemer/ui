package forms

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Cond is a condition on other fields, built from their typed handles:
//
//	f.Fixed = Decimal(f.Form, "fixed").When(f.Type.Eq(models.PromotionTypeFixed)).Required()
//
// The server evaluates it during Validate; a renderer serialises it with
// MarshalJSON for the client, which must pass the conformance vectors in
// testdata/cond. Both compare the same text, so they agree.
type Cond struct{ n *condNode }

type condNode struct {
	op    string // "eq", "filled", "and", "or", "not"
	field *field // eq, filled
	text  string // eq: the value as the field formats it
	args  []*condNode
}

// Eq holds when the field is active, present, and its text equals v as
// the field formats it. On a field of several values or of files it
// panics.
func (h *Field[T]) Eq(v T) Cond {
	fd := h.use("Eq")
	fd.mustBeScalar("Eq")
	return Cond{&condNode{op: "eq", field: fd, text: h.formatText(v)}}
}

// In holds when Eq holds for one of vs.
func (h *Field[T]) In(vs ...T) Cond {
	fd := h.use("In")
	fd.mustBeScalar("In")
	or := &condNode{op: "or"}
	for _, v := range vs {
		or.args = append(or.args, &condNode{op: "eq", field: fd, text: h.formatText(v)})
	}
	return Cond{or}
}

// Filled holds when the field is active and present. A Checkbox is filled
// when ticked.
func (h *Field[T]) Filled() Cond {
	return Cond{&condNode{op: "filled", field: h.use("Filled")}}
}

func (fd *field) mustBeScalar(method string) {
	if fd.base.multi || fd.base.files {
		panic(fmt.Sprintf("forms: %s on field %q, whose value is a list or a file: compare a single value", method, fd.name))
	}
}

// formatText formats v at this handle's stage down to the text a client
// sees, trimmed as Bind trims submitted text.
func (h *Field[T]) formatText(v T) string {
	var x any = v
	for k := h.stage; k > 0; k-- {
		x = h.f.stages[k].format(x)
	}
	t := h.f.base.format(x)
	if len(t) == 0 {
		return ""
	}
	if h.f.raw {
		return t[0]
	}
	return strings.TrimSpace(t[0])
}

// All holds when every c holds; All() holds.
func All(cs ...Cond) Cond { return Cond{combine("and", cs)} }

// Any holds when some c holds; Any() does not.
func Any(cs ...Cond) Cond { return Cond{combine("or", cs)} }

// Not holds when c does not.
func Not(c Cond) Cond { return Cond{combine("not", []Cond{c})} }

func combine(op string, cs []Cond) *condNode {
	n := &condNode{op: op}
	for i, c := range cs {
		if c.n == nil {
			panic(fmt.Sprintf("forms: argument %d of a condition is a zero Cond", i))
		}
		n.args = append(n.args, c.n)
	}
	return n
}

// When makes the field active only while c holds. An inactive field is
// cleared at Validate: its value, text and error are dropped and its rules
// don't run. When twice on one field, a zero Cond, or a condition that
// closes a cycle panics.
func (h *Field[T]) When(c Cond) *Field[T] {
	fd := h.builder("When")
	fd.when = checkWhen(fd, fd.path(), fd.when, c)
	return h
}

// checkWhen validates a condition given to node's When and returns it.
// A condition may read fields of the same form or of forms around it, or
// of a group anywhere, but never into a list's items from outside them.
func checkWhen(node any, path string, existing *condNode, c Cond) *condNode {
	if c.n == nil {
		panic(fmt.Sprintf("forms: When on %q with a zero Cond", path))
	}
	if existing != nil {
		panic(fmt.Sprintf("forms: When called twice on %q; combine the conditions with All or Any", path))
	}
	mine := chain(node)
	c.n.walk(func(ref *field) {
		theirs := chain(ref)
		if theirs[len(theirs)-1] != mine[len(mine)-1] {
			panic(fmt.Sprintf("forms: %q's condition reads field %q of another form", path, ref.path()))
		}
		for _, x := range theirs {
			if f, ok := x.(*Form); ok && f.item != nil && !slices.Contains(mine, x) {
				panic(fmt.Sprintf("forms: %q's condition reads %q inside a list's item from outside it", path, ref.path()))
			}
		}
		if requires(ref, node, map[any]bool{}) {
			panic(fmt.Sprintf("forms: %q's condition reads %q, whose activity depends on %q: a cycle", path, ref.path(), path))
		}
	})
	return c.n
}

// chain returns x and every node around it, innermost first, ending with
// the root form. A list item's chain passes through its list.
func chain(x any) []any {
	var out []any
	for x != nil {
		out = append(out, x)
		switch n := x.(type) {
		case *field:
			x = n.form
		case *list:
			x = n.form
		case *Form:
			switch {
			case n.item != nil:
				x = n.item
			case n.parent != nil:
				x = n.parent
			default:
				x = nil
			}
		}
	}
	return out
}

func whenOf(x any) *condNode {
	switch n := x.(type) {
	case *field:
		return n.when
	case *list:
		return n.when
	case *Form:
		return n.when
	}
	return nil
}

// requires reports whether ref's activity depends on target's: target is
// around ref, or a condition on ref's chain reads a field that requires it.
// Each field is visited once, so chains and diamonds stay linear.
func requires(ref *field, target any, seen map[any]bool) bool {
	refs := chain(ref)
	if slices.Contains(refs, target) {
		return true
	}
	found := false
	for _, x := range refs {
		if w := whenOf(x); w != nil {
			w.walk(func(r *field) {
				if !found && !seen[r] {
					seen[r] = true
					found = requires(r, target, seen)
				}
			})
		}
		if found {
			return true
		}
	}
	return false
}

func (n *condNode) walk(fn func(*field)) {
	if n.field != nil {
		fn(n.field)
	}
	for _, a := range n.args {
		a.walk(fn)
	}
}

// active reports whether fd and every node around it are active, reading
// the fields' current state: bound text, or initial values before Bind.
// memo caches one pass.
func (fd *field) active(memo map[any]bool) bool {
	if memo == nil {
		memo = map[any]bool{}
	}
	if v, ok := memo[fd]; ok {
		return v
	}
	v := fd.form.active(memo) && (fd.when == nil || fd.when.eval(memo))
	memo[fd] = v
	return v
}

func (n *condNode) eval(memo map[any]bool) bool {
	switch n.op {
	case "eq":
		if !n.field.active(memo) || !n.field.present {
			return false
		}
		if n.field.base.box { // a Checkbox compares "true" or "false", as a client reads .checked
			return len(n.field.vals) > 0 && fmt.Sprint(n.field.vals[0]) == n.text
		}
		return len(n.field.text) > 0 && n.field.text[0] == n.text
	case "filled":
		if !n.field.active(memo) || !n.field.present {
			return false
		}
		if n.field.base.box { // a Checkbox: filled when ticked
			return len(n.field.vals) > 0 && n.field.vals[0] == true
		}
		return true
	case "and":
		for _, a := range n.args {
			if !a.eval(memo) {
				return false
			}
		}
		return true
	case "or":
		for _, a := range n.args {
			if a.eval(memo) {
				return true
			}
		}
		return false
	case "not":
		return !n.args[0].eval(memo)
	}
	panic("forms: unknown condition op " + n.op)
}

// wire is the JSON shape of a condition (forms-design §7.3).
type wire struct {
	Typ  string  `json:"typ"`
	K    string  `json:"k,omitempty"`
	Op   string  `json:"op"`
	V    *string `json:"v,omitempty"`
	Args *[]wire `json:"args,omitempty"` // a pointer, so an empty list is still written
}

func (n *condNode) wire() wire {
	switch n.op {
	case "eq":
		v := n.text
		return wire{Typ: "c", K: n.field.path(), Op: "eq", V: &v}
	case "filled":
		return wire{Typ: "c", K: n.field.path(), Op: "filled"}
	}
	args := []wire{}
	for _, a := range n.args {
		args = append(args, a.wire())
	}
	return wire{Typ: "l", Op: n.op, Args: &args}
}

// MarshalJSON writes the condition for the client:
// {"typ":"c","k":"<path>","op":"eq","v":"<text>"}, {"typ":"c","k":"<path>","op":"filled"},
// or {"typ":"l","op":"and"|"or"|"not","args":[…]}.
func (c Cond) MarshalJSON() ([]byte, error) {
	if c.n == nil {
		return nil, fmt.Errorf("forms: MarshalJSON of a zero Cond")
	}
	return json.Marshal(c.n.wire())
}
