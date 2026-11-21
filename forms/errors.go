package forms

import (
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
)

// Errors forms reports itself, all of kind errors.Invalid. Keys are
// permanent. Texts follow errors' convention: lower case, no final period.
var (
	ErrUnreadable  = errors.Key("errors.forms.unreadable", "the form could not be read", errors.Invalid)
	ErrInvalid     = errors.Key("errors.forms.invalid", "is not valid", errors.Invalid)
	ErrNotAnOption = errors.Key("errors.forms.not_an_option", "is not one of the choices", errors.Invalid)
	ErrRequired    = errors.Key("errors.forms.required", "is required", errors.Invalid)
	ErrMinItems    = errors.Key1[int]("errors.forms.min_items", "add at least %d", errors.Invalid)
	ErrMaxItems    = errors.Key1[int]("errors.forms.max_items", "add at most %d", errors.Invalid)
)

// FieldError ties an error to the node that reported it.
type FieldError struct {
	Path  string // "name", "percentage.value", "constraints[2].type"; "" for the form
	Label i18n.Ter
	Err   error
}

// Error returns "<path>: <err>", or the error's text alone for the form.
func (e *FieldError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}
	return e.Path + ": " + e.Err.Error()
}

// Unwrap returns the node's error.
func (e *FieldError) Unwrap() error { return e.Err }

// public reports whether people would see err's own text, not Unknown.
func public(err error) bool {
	for range errors.All(err) {
		return true
	}
	return false
}

func mustBePublic(where string, err error) {
	if err == nil {
		panic("forms: " + where + ": nil error")
	}
	if !public(err) {
		panic("forms: " + where + ": people would see Unknown for " + err.Error() +
			"; pass an errors declaration (a Def1–Def3 needs .With)")
	}
}
