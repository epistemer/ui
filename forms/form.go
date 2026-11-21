package forms

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"regexp"

	"github.com/epistemer/errors"
)

// Form holds a form's fields in declaration order and their state for one
// request. Embed *Form in a struct of typed handles; see the package
// documentation.
type Form struct {
	path   string // "" for the root
	fields []*field
	names  map[string]bool
	checks []func(context.Context) error

	bound     bool
	validated bool
	err       error // the form-level error: unreadable, a check, Fail
}

// New returns an empty root form.
func New() *Form {
	return &Form{names: map[string]bool{}}
}

var fieldName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// add registers a field; it panics on a malformed or duplicate name, or
// after Bind.
func (f *Form) add(fd *field) {
	if f.bound {
		panic(fmt.Sprintf("forms: field %q declared after Bind", fd.name))
	}
	if !fieldName.MatchString(fd.name) {
		panic(fmt.Sprintf("forms: field name %q: a name is lower-case letters, digits and _, starting with a letter", fd.name))
	}
	if f.names[fd.name] {
		panic(fmt.Sprintf("forms: field %q declared twice", fd.name))
	}
	f.names[fd.name] = true
	fd.form = f
	f.fields = append(f.fields, fd)
}

// Bind records the submitted text of every field, in declaration order,
// decides which are present, and runs their conversions. A conversion
// failure becomes that field's error; no rule runs. Names the form did not
// declare are ignored. A second Bind panics.
func (f *Form) Bind(src Source) {
	if f.bound {
		panic("forms: Bind called twice")
	}
	if src == nil {
		panic("forms: Bind(nil)")
	}
	f.bound = true
	for _, fd := range f.fields {
		fd.bind(src)
	}
}

// Validate runs every field's checks and then the form's Check functions,
// and reports whether the form is valid (interface §5.1). It panics before
// Bind.
func (f *Form) Validate(ctx context.Context) bool {
	if !f.bound {
		panic("forms: Validate before Bind")
	}
	failed := false
	for _, fd := range f.fields {
		if fd.validate(ctx) != nil {
			failed = true
		}
	}
	if !failed && f.err == nil {
		for _, check := range f.checks {
			if err := check(ctx); err != nil {
				f.err = err
				break
			}
		}
	}
	f.validated = true
	return f.Valid()
}

// Submit reads r, binds and validates. An unreadable request records
// ErrUnreadable on the form and returns false.
func (f *Form) Submit(ctx context.Context, r *http.Request) bool {
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

// Valid reports whether Validate ran and no node or form-level error
// remains.
func (f *Form) Valid() bool {
	return f.validated && f.Err() == nil
}

// Fail sets the form-level error, such as one from the service layer,
// replacing any earlier one. Valid is then false.
func (f *Form) Fail(err error) {
	if err == nil {
		panic("forms: Fail(nil)")
	}
	f.err = err
}

// Check adds a check that belongs to no single field. Checks run in order
// after every field passed; the first failure is the form-level error.
func (f *Form) Check(fn func(ctx context.Context) error) {
	if fn == nil {
		panic("forms: Check(nil)")
	}
	if f.bound {
		panic("forms: Check after Bind")
	}
	f.checks = append(f.checks, fn)
}

// Err is nil when the form has no error. Otherwise it joins one
// *FieldError per failing node: the form-level error first, then fields
// in declaration order.
func (f *Form) Err() error {
	var errs []error
	for e := range f.Errors() {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

// Errors yields what Err joins, in the same order.
func (f *Form) Errors() iter.Seq[*FieldError] {
	return func(yield func(*FieldError) bool) {
		if f.err != nil && !yield(&FieldError{Path: f.path, Err: f.err}) {
			return
		}
		for _, fd := range f.fields {
			if fd.err != nil && !yield(&FieldError{Path: fd.path(), Label: fd.label, Err: fd.err}) {
				return
			}
		}
	}
}
