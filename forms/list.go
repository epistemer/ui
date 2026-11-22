package forms

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/epistemer/i18n"
)

// DefaultMaxItems is how many items a list binds when MaxItems wasn't set.
const DefaultMaxItems = 256

// IndexPlaceholder stands for an item's index in a list's template row,
// which a client clones and renumbers: constraints[__i__].type.
const IndexPlaceholder = "__i__"

// Group declares a nested form under name and returns what build returns,
// usually a struct of handles built on the sub-form. Its fields post as
// name.<field>.
//
//	f.Percentage = f.Group("percentage", newPercentage) // *PercentageForm
func (f *Form) Group[G any](name string, build func(g *Form) G) G {
	if build == nil {
		panic(fmt.Sprintf("forms: Group %q with a nil build", name))
	}
	f.register(name, "group")
	g := &Form{parent: f, name: name, names: map[string]bool{}, label: i18n.Static(name)}
	f.nodes = append(f.nodes, g)
	return build(g)
}

// List declares a repeatable nested form under name; build declares one
// item's fields on its sub-form. An item's fields post as
// name[<index>].<field>.
//
// A row whose controls all post nothing when empty (unchecked checkboxes,
// a multi-select with nothing chosen, controls a condition disabled) is
// dropped on bind before its rules run. Text inputs, selects (Bool, Enum, a
// single Options), and file inputs always post; a list whose rows could
// hold only the first kind adds a field of the second.
func (f *Form) List[G any](name string, build func(g *Form) G) *List[G] {
	if build == nil {
		panic(fmt.Sprintf("forms: List %q with a nil build", name))
	}
	f.register(name, "list")
	l := &list{form: f, name: name, label: i18n.Static(name), max: DefaultMaxItems,
		build: func(g *Form) any { return build(g) }}
	f.nodes = append(f.nodes, l)
	l.templateRow()
	return &List[G]{l}
}

// List is the handle of a repeatable nested form whose items are G.
type List[G any] struct{ l *list }

type list struct {
	form     *Form
	name     string
	label    i18n.Ter
	labelSet bool
	when     *condNode
	min, max int
	maxSet   bool
	build    func(*Form) any

	bound    bool
	items    []item
	blanks   []item // MinItems blank rows for Control.Children before Bind or Add
	template *item  // built on demand at IndexPlaceholder
	err      error
}

type item struct {
	form  *Form
	value any // what build returned
}

func (l *list) path() string { return join(l.form.path(), l.name) }

func (l *list) builder(method string) {
	if l.form.isBound() {
		panic(fmt.Sprintf("forms: %s on list %q after Bind", method, l.name))
	}
}

// Label sets the list's label. The default is its name.
func (h *List[G]) Label(t i18n.Ter) *List[G] {
	h.l.builder("Label")
	if t == nil {
		panic(fmt.Sprintf("forms: Label(nil) on list %q", h.l.name))
	}
	h.l.label, h.l.labelSet = t, true
	return h
}

// When makes the list active only while c holds; an inactive list has no
// items after Validate.
func (h *List[G]) When(c Cond) *List[G] {
	h.l.builder("When")
	h.l.when = checkWhen(h.l, h.l.path(), h.l.when, c)
	return h
}

// MinItems makes the list need at least n items (ErrMinItems otherwise).
func (h *List[G]) MinItems(n int) *List[G] {
	h.l.builder("MinItems")
	if n < 0 || h.l.maxSet && n > h.l.max {
		panic(fmt.Sprintf("forms: MinItems(%d) on list %q: at least 0 and at most MaxItems (%d)", n, h.l.name, h.l.max))
	}
	h.l.min, h.l.blanks = n, nil
	if !h.l.maxSet && n > h.l.max {
		h.l.max = n // the default limit rises to the minimum
	}
	return h
}

// MaxItems binds at most n items; more is ErrMaxItems, and the rest are
// not built.
func (h *List[G]) MaxItems(n int) *List[G] {
	h.l.builder("MaxItems")
	if n < 1 || n < h.l.min {
		panic(fmt.Sprintf("forms: MaxItems(%d) on list %q: at least 1 and at least MinItems (%d)", n, h.l.name, h.l.min))
	}
	h.l.max, h.l.maxSet = n, true
	return h
}

// Items returns the list's items, in order.
func (h *List[G]) Items() []G {
	out := make([]G, len(h.l.items))
	for i, it := range h.l.items {
		out[i], _ = it.value.(G) // a nil interface G stays nil
	}
	return out
}

// Add appends an item, for initial values on an edit page, and returns it.
// It panics after Bind.
func (h *List[G]) Add() G {
	h.l.builder("Add")
	h.l.blanks = nil
	it := h.l.newItem(len(h.l.items))
	h.l.items = append(h.l.items, it)
	v, _ := it.value.(G)
	return v
}

// Err returns the list's own error (MinItems, MaxItems), or nil.
func (h *List[G]) Err() error { return h.l.err }

func (l *list) newItem(index int) item {
	g := &Form{parent: l.form, item: l, index: index, names: map[string]bool{}, building: true}
	v := l.build(g)
	g.building = false
	return item{form: g, value: v}
}

// displayItems returns the items Control.Children yields: l.items when the
// list was bound or given items by Add, else MinItems blank items built at
// indices 0..MinItems-1.
func (l *list) displayItems() []item {
	if l.bound || len(l.items) > 0 || l.min == 0 {
		return l.items
	}
	if len(l.blanks) != l.min {
		l.blanks = make([]item, l.min)
		for i := range l.min {
			l.blanks[i] = l.newItem(i)
		}
	}
	return l.blanks
}

// template returns the list's template row, built once.
func (l *list) templateRow() *item {
	if l.template == nil {
		it := l.newItem(-1)
		l.template = &it
	}
	return l.template
}

// parseIndex reads "<index>].<rest>" from what follows "<list>[" in a
// name and returns the index and the rest.
func parseIndex(s string) (int, string, bool) {
	end := strings.IndexByte(s, ']')
	if end < 1 || end > 6 || s[0] == '0' && end > 1 {
		return 0, "", false
	}
	n := 0
	for _, c := range s[:end] {
		if c < '0' || c > '9' {
			return 0, "", false
		}
		n = n*10 + int(c-'0')
	}
	rest, ok := strings.CutPrefix(s[end+1:], ".")
	return n, rest, ok && rest != ""
}

// declares reports whether path, relative to f, names a field f declares:
// "name", "group.name", "list[3].name".
func (f *Form) declares(path string) bool {
	for _, n := range f.nodes {
		switch n := n.(type) {
		case *field:
			if path == n.name {
				return true
			}
		case *Form:
			if rest, ok := strings.CutPrefix(path, n.name+"."); ok && n.declares(rest) {
				return true
			}
		case *list:
			if rest, ok := strings.CutPrefix(path, n.name+"["); ok {
				if _, inner, ok := parseIndex(rest); ok && n.templateRow().form.declares(inner) {
					return true
				}
			}
		}
	}
	return false
}

// bind builds and binds one item per submitted index (interface §5.8,
// §9.2) and reports whether any of them saw input.
func (l *list) bind(src Source, names []string) bool {
	l.bound, l.items, l.blanks, l.err = true, nil, nil, nil
	prefix := l.path() + "["
	tpl := l.templateRow().form
	seenIdx := map[int]bool{}
	var indices []int
	// names is sorted: the names under this list are one run.
	for i := sort.SearchStrings(names, prefix); i < len(names) && strings.HasPrefix(names[i], prefix); i++ {
		// An index counts only when one of its names reaches a field an
		// item declares, so junk never builds items or uses up MaxItems.
		idx, rest, ok := parseIndex(names[i][len(prefix):])
		if !ok || seenIdx[idx] || !tpl.declares(rest) {
			continue
		}
		seenIdx[idx] = true
		indices = append(indices, idx)
	}
	slices.Sort(indices)
	if len(indices) > l.max {
		l.err = ErrMaxItems.With(l.max)
		indices = indices[:l.max]
	}
	seen := false
	for _, i := range indices {
		it := l.newItem(i)
		if it.form.bindNodes(src, names) {
			l.items = append(l.items, it)
			seen = true
		}
	}
	for k, it := range l.items {
		it.form.index = k // renumber: paths in errors and a re-rendered page follow
	}
	return seen
}

func (l *list) active(memo map[any]bool) bool {
	if v, ok := memo[l]; ok {
		return v
	}
	v := l.form.active(memo) && (l.when == nil || l.when.eval(memo))
	memo[l] = v
	return v
}

// validate validates the items, then the list's limits, and reports
// whether anything failed.
func (l *list) validate(ctx context.Context, memo map[any]bool) bool {
	failed := false
	for _, it := range l.items {
		if it.form.validateNodes(ctx, memo) {
			failed = true
		}
	}
	if l.err == nil && len(l.items) < l.min {
		l.err = ErrMinItems.With(l.min)
	}
	return failed || l.err != nil
}
