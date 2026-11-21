package forms

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/option"
	"github.com/epistemer/ui/forms/internal/hook"
	"github.com/epistemer/ui/forms/rule"
)

// Field is the handle of one field whose value has type T. Declare it with
// a constructor on Form (Text, Int, Enum, …); its builder methods chain.
type Field[T any] struct {
	f     *field
	stage int // the conversion stage this handle reads; As adds one
}

// field is a field's state, shared by its handles. Values are kept per
// conversion stage, untyped: only handles are typed.
type field struct {
	form *Form
	name string
	kind Kind

	label, placeholder, help i18n.Ter
	labelSet                 bool
	required                 bool
	requiredErr, invalidErr  error
	raw                      bool // Text only: keep white space
	dateOnly                 bool // Date: time rules compare dates
	options                  []option.SelectOption
	optionsSet               bool // Options or Enum: the text must be one of options
	when                     *condNode

	base   codec
	stages []*stage // stages[0] holds the base type's rules
	live   int      // the stage of the only handle still usable

	// State.
	present bool
	text    []string
	vals    []any // per stage, up to the stage that failed to convert
	convAt  int   // the stage whose conversion failed; -1 for none
	convErr error
	err     error // the field's error: conversion, required, rule or Fail
}

// codec reads the base type from submitted input and writes it back.
type codec struct {
	text   bool // Text: the only constructor Raw applies to
	multi  bool // Texts, Files: every item, not the last
	files  bool // File, Files
	always bool // Checkbox: never absent
	box    bool // Checkbox: conditions read its value, true or false, not its text
	parse  func(in input) (any, error)
	format func(v any) []string
}

type input struct {
	texts []string
	files []*multipart.FileHeader
}

type stage struct {
	parse  func(any) (any, error) // from the previous stage's value; nil for stage 0
	format func(any) any          // to the previous stage's value
	rules  []func(context.Context, any) error
	attrs  [][2]string
}

func newField[T any](form *Form, name string, kind Kind, c codec) *Field[T] {
	fd := &field{
		name:   name,
		kind:   kind,
		label:  i18n.Static(name),
		base:   c,
		stages: []*stage{{}},
		convAt: -1,
	}
	form.add(fd)
	return &Field[T]{f: fd}
}

func (fd *field) path() string { return join(fd.form.path(), fd.name) }

// use returns the field state after checking that h is the live handle.
func (h *Field[T]) use(method string) *field {
	if h.stage != h.f.live {
		panic(fmt.Sprintf("forms: %s on field %q's handle after As; use the handle As returned", method, h.f.name))
	}
	return h.f
}

// builder is use for builder methods, which are legal only before Bind.
func (h *Field[T]) builder(method string) *field {
	fd := h.use(method)
	if fd.form.isBound() {
		panic(fmt.Sprintf("forms: %s on field %q after Bind", method, fd.path()))
	}
	return fd
}

// Label sets the field's label. The default is its name, untranslated.
func (h *Field[T]) Label(l i18n.Ter) *Field[T] {
	fd := h.builder("Label")
	if l == nil {
		panic(fmt.Sprintf("forms: Label(nil) on field %q", fd.name))
	}
	fd.label, fd.labelSet = l, true
	return h
}

// Placeholder sets the placeholder. The default is the label.
func (h *Field[T]) Placeholder(p i18n.Ter) *Field[T] {
	fd := h.builder("Placeholder")
	if p == nil {
		panic(fmt.Sprintf("forms: Placeholder(nil) on field %q", fd.name))
	}
	fd.placeholder = p
	return h
}

// Help sets a help text shown with the field.
func (h *Field[T]) Help(t i18n.Ter) *Field[T] {
	fd := h.builder("Help")
	if t == nil {
		panic(fmt.Sprintf("forms: Help(nil) on field %q", fd.name))
	}
	fd.help = t
	return h
}

// Raw keeps the white space around a Text field's value, for passwords. On
// any other field it panics.
func (h *Field[T]) Raw() *Field[T] {
	fd := h.builder("Raw")
	if !fd.base.text || h.stage != 0 {
		panic(fmt.Sprintf("forms: Raw on field %q, which is not a Text field", fd.name))
	}
	fd.raw = true
	return h
}

// Required makes the field required: absent, it reports ErrRequired. A
// required Checkbox must be ticked.
func (h *Field[T]) Required() *Field[T] {
	h.builder("Required").required = true
	return h
}

// RequiredWith makes the field required, reporting err when it is absent.
func (h *Field[T]) RequiredWith(err error) *Field[T] {
	fd := h.builder("RequiredWith")
	mustBePublic("RequiredWith on field "+fd.name, err)
	fd.required, fd.requiredErr = true, err
	return h
}

// InvalidWith sets the error shown when the submitted text can't be
// converted. The default is ErrInvalid.
func (h *Field[T]) InvalidWith(err error) *Field[T] {
	fd := h.builder("InvalidWith")
	mustBePublic("InvalidWith on field "+fd.name, err)
	fd.invalidErr = err
	return h
}

// Is adds rules on the value of this handle's type. They run in order and
// stop at the first failure.
func (h *Field[T]) Is(rules ...rule.Rule[T]) *Field[T] {
	fd := h.builder("Is")
	st := fd.stages[h.stage]
	for _, r := range rules {
		if fd.dateOnly && h.stage == 0 {
			r = hook.DateOnly(r).(rule.Rule[T])
		}
		st.rules = append(st.rules, func(ctx context.Context, v any) error {
			return rule.Check(ctx, typed[T](v), r)
		})
		for k, v := range r.Attrs() {
			st.attrs = append(st.attrs, [2]string{k, v})
		}
	}
	return h
}

// Options gives the field's choices and makes the submitted text one of
// their Value()s (ErrNotAnOption otherwise). On an Enum it replaces the
// choices.
func (h *Field[T]) Options(opts ...option.SelectOption) *Field[T] {
	fd := h.builder("Options")
	for i, o := range opts {
		if o == nil {
			panic(fmt.Sprintf("forms: Options on field %q: option %d is nil", fd.name, i))
		}
	}
	fd.options = append(make([]option.SelectOption, 0, len(opts)), opts...)
	fd.optionsSet = true
	if fd.kind == KindText && fd.base.multi {
		fd.kind = KindMultiSelect
	} else if fd.kind == KindText || fd.kind == KindNumber {
		fd.kind = KindSelect
	}
	return h
}

// Hint overrides the field's kind, such as KindMultiline for a Text.
func (h *Field[T]) Hint(k Kind) *Field[T] {
	fd := h.builder("Hint")
	if k.String() == "" || k == KindGroup || k == KindList {
		panic(fmt.Sprintf("forms: Hint(%d) on field %q: not a field kind", k, fd.name))
	}
	fd.kind = k
	return h
}

// Set gives the field an initial value, for an edit page, and marks it
// present. It panics after Bind.
func (h *Field[T]) Set(v T) *Field[T] {
	fd := h.use("Set")
	if fd.form.isBound() {
		panic(fmt.Sprintf("forms: Set on field %q after Bind", fd.path()))
	}
	vals := make([]any, h.stage+1)
	vals[h.stage] = v
	for k := h.stage; k > 0; k-- {
		vals[k-1] = fd.stages[k].format(vals[k])
	}
	fd.vals = vals
	fd.text = fd.base.format(vals[0])
	fd.present = true
	return h
}

// SetPtr is Set(*p) when p is not nil, for nullable columns.
func (h *Field[T]) SetPtr(p *T) *Field[T] {
	if fd := h.use("SetPtr"); fd.form.isBound() {
		panic(fmt.Sprintf("forms: SetPtr on field %q after Bind", fd.path()))
	}
	if p != nil {
		h.Set(*p)
	}
	return h
}

// Value returns the field's value, or the zero T when the field is absent,
// inactive or in error.
func (h *Field[T]) Value() T {
	v, _ := h.Get()
	return v
}

// Get returns the field's value and whether it has one: present, active
// (after Validate) and without error.
func (h *Field[T]) Get() (T, bool) {
	fd := h.use("Get")
	var zero T
	if !fd.present || fd.err != nil || len(fd.vals) <= h.stage {
		return zero, false
	}
	return typed[T](fd.vals[h.stage]), true
}

// Ptr returns a pointer to the value, or nil when Get reports none, for
// nullable columns.
func (h *Field[T]) Ptr() *T {
	if v, ok := h.Get(); ok {
		return &v
	}
	return nil
}

// Err returns the field's error, or nil.
func (h *Field[T]) Err() error { return h.use("Err").err }

// Fail attaches an error to the field after Validate, such as a unique
// violation from the database.
func (h *Field[T]) Fail(err error) {
	fd := h.use("Fail")
	if err == nil {
		panic(fmt.Sprintf("forms: Fail(nil) on field %q", fd.name))
	}
	fd.err = err
}

// bind binds fd from src and reports whether src had anything under its
// name.
func (fd *field) bind(src Source) bool {
	name := fd.path()
	rawTexts, rawFiles := src.Values(name), src.Files(name)
	fd.vals, fd.text, fd.present, fd.convAt, fd.convErr, fd.err = nil, nil, false, -1, nil, nil
	var in input
	if fd.base.files {
		for _, fh := range rawFiles {
			if fh != nil && fh.Filename != "" {
				in.files = append(in.files, fh)
			}
		}
		if !fd.base.multi && len(in.files) > 1 {
			in.files = in.files[len(in.files)-1:]
		}
		fd.present = len(in.files) > 0
	} else {
		for _, t := range rawTexts {
			if !fd.raw {
				t = strings.TrimSpace(t)
			}
			if t != "" {
				in.texts = append(in.texts, t)
			}
		}
		if !fd.base.multi && len(in.texts) > 1 {
			in.texts = in.texts[len(in.texts)-1:]
		}
		fd.text = in.texts
		fd.present = len(in.texts) > 0 || fd.base.always
	}
	seen := len(rawTexts) > 0 || len(rawFiles) > 0
	if !fd.present {
		return seen
	}
	v, err := fd.base.parse(in)
	if err != nil {
		fd.convAt, fd.convErr = 0, err
		fd.err = fd.conversionError()
		return seen
	}
	fd.vals = []any{v}
	for k := 1; k < len(fd.stages); k++ {
		v, err = fd.stages[k].parse(v)
		if err != nil {
			fd.convAt, fd.convErr = k, err
			fd.err = fd.conversionError()
			return seen
		}
		fd.vals = append(fd.vals, v)
	}
	return seen
}

// clear drops an inactive field's value, text and error.
func (fd *field) clear() {
	fd.vals, fd.text, fd.present, fd.convAt, fd.convErr, fd.err = nil, nil, false, -1, nil, nil
}

// validate decides the field's error (interface §5.1, §9.2) and returns it.
func (fd *field) validate(ctx context.Context) error {
	fd.err = nil
	switch {
	case !fd.present:
		if fd.required {
			fd.err = fd.requiredError()
		}
	case fd.base.always && fd.required && len(fd.vals) > 0 && fd.vals[0] == false:
		fd.err = fd.requiredError()
	default:
		fd.err = fd.check(ctx)
	}
	return fd.err
}

func (fd *field) check(ctx context.Context) error {
	if fd.optionsSet {
		for _, t := range fd.text {
			if !fd.isOption(t) {
				return ErrNotAnOption
			}
		}
	}
	for k, st := range fd.stages {
		if fd.convAt == k {
			return fd.conversionError()
		}
		for _, r := range st.rules {
			if err := r(ctx, fd.vals[k]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (fd *field) isOption(t string) bool {
	for _, o := range fd.options {
		if o.Value() == t {
			return true
		}
	}
	return false
}

func (fd *field) requiredError() error {
	if fd.requiredErr != nil {
		return fd.requiredErr
	}
	return ErrRequired
}

// conversionError shows a declared conversion error as it is, and wraps
// any other as the cause of the field's invalid error.
func (fd *field) conversionError() error {
	if public(fd.convErr) {
		return fd.convErr
	}
	if fd.invalidErr == nil {
		return ErrInvalid.Wrap(fd.convErr)
	}
	if d, ok := fd.invalidErr.(*errors.Def); ok {
		return d.Wrap(fd.convErr)
	}
	return fmt.Errorf("%w: %w", fd.invalidErr, fd.convErr)
}
