package forms

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
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
func (f *Form) List[G any](name string, build func(g *Form) G) *List[G] {
	if build == nil {
		panic(fmt.Sprintf("forms: List %q with a nil build", name))
	}
	f.register(name, "list")
	l := &list{form: f, name: name, label: i18n.Static(name), max: DefaultMaxItems,
		build: func(g *Form) any { return build(g) }}
	f.nodes = append(f.nodes, l)
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
	build    func(*Form) any

	items    []item
	template *item // built on demand at IndexPlaceholder
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
	if n < 0 || n > h.l.max {
		panic(fmt.Sprintf("forms: MinItems(%d) on list %q: at least 0 and at most MaxItems (%d)", n, h.l.name, h.l.max))
	}
	h.l.min = n
	return h
}

// MaxItems binds at most n items; more is ErrMaxItems, and the rest are
// not built.
func (h *List[G]) MaxItems(n int) *List[G] {
	h.l.builder("MaxItems")
	if n < 1 || n < h.l.min {
		panic(fmt.Sprintf("forms: MaxItems(%d) on list %q: at least 1 and at least MinItems (%d)", n, h.l.name, h.l.min))
	}
	h.l.max = n
	return h
}

// Items returns the list's items, in order.
func (h *List[G]) Items() []G {
	out := make([]G, len(h.l.items))
	for i, it := range h.l.items {
		out[i] = it.value.(G)
	}
	return out
}

// Add appends an item, for initial values on an edit page, and returns it.
// It panics after Bind.
func (h *List[G]) Add() G {
	h.l.builder("Add")
	it := h.l.newItem(len(h.l.items))
	h.l.items = append(h.l.items, it)
	return it.value.(G)
}

// Err returns the list's own error (MinItems, MaxItems), or nil.
func (h *List[G]) Err() error { return h.l.err }

func (l *list) newItem(index int) item {
	r := l.form.root()
	was := r.building
	r.building = true
	defer func() { r.building = was }()
	g := &Form{parent: l.form, item: l, index: index, names: map[string]bool{}}
	return item{form: g, value: l.build(g)}
}

var itemIndex = regexp.MustCompile(`^(0|[1-9][0-9]{0,5})\](\.|$)`)

// bind builds and binds one item per submitted index (interface §5.8,
// §9.2) and reports whether any of them saw input.
func (l *list) bind(src Source) bool {
	l.items, l.err = nil, nil
	prefix := l.path() + "["
	seenIdx := map[int]bool{}
	var indices []int
	for name := range src.Names() {
		rest, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		m := itemIndex.FindStringSubmatch(rest)
		if m == nil {
			continue
		}
		i, _ := strconv.Atoi(m[1])
		if !seenIdx[i] {
			seenIdx[i] = true
			indices = append(indices, i)
		}
	}
	slices.Sort(indices)
	if len(indices) > l.max {
		l.err = ErrMaxItems.With(l.max)
		indices = indices[:l.max]
	}
	seen := false
	for _, i := range indices {
		it := l.newItem(i)
		if it.form.bindNodes(src) {
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
