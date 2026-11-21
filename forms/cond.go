package forms

import (
	"encoding/json"
	"fmt"
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
// sees.
func (h *Field[T]) formatText(v T) string {
	var x any = v
	for k := h.stage; k > 0; k-- {
		x = h.f.stages[k].format(x)
	}
	if t := h.f.base.format(x); len(t) > 0 {
		return t[0]
	}
	return ""
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
	if c.n == nil {
		panic(fmt.Sprintf("forms: When on field %q with a zero Cond", fd.name))
	}
	if fd.when != nil {
		panic(fmt.Sprintf("forms: When called twice on field %q; combine the conditions with All or Any", fd.name))
	}
	c.n.walk(func(ref *field) {
		if ref.form.root() != fd.form.root() {
			panic(fmt.Sprintf("forms: field %q's condition reads field %q of another form", fd.name, ref.name))
		}
		if ref == fd || ref.dependsOn(fd) {
			panic(fmt.Sprintf("forms: field %q's condition reads field %q, which depends on %q: a cycle", fd.name, ref.name, fd.name))
		}
	})
	fd.when = c.n
	return h
}

func (n *condNode) walk(fn func(*field)) {
	if n.field != nil {
		fn(n.field)
	}
	for _, a := range n.args {
		a.walk(fn)
	}
}

// dependsOn reports whether fd's activity reads target, directly or not.
func (fd *field) dependsOn(target *field) bool {
	if fd.when == nil {
		return false
	}
	found := false
	fd.when.walk(func(ref *field) {
		if ref == target || ref.dependsOn(target) {
			found = true
		}
	})
	return found
}

// active reports whether fd's condition holds, reading the fields' current
// state: bound text, or initial values before Bind. memo caches one pass.
func (fd *field) active(memo map[*field]bool) bool {
	if fd.when == nil {
		return true
	}
	if v, ok := memo[fd]; ok {
		return v
	}
	v := fd.when.eval(memo)
	memo[fd] = v
	return v
}

func (n *condNode) eval(memo map[*field]bool) bool {
	switch n.op {
	case "eq":
		return n.field.active(memo) && n.field.present && len(n.field.text) > 0 && n.field.text[0] == n.text
	case "filled":
		if !n.field.active(memo) || !n.field.present {
			return false
		}
		if n.field.base.always { // a Checkbox: filled when ticked
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
