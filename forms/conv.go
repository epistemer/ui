package forms

import (
	"fmt"
)

// Conv converts a field's value from T to U and back. Parse runs at Bind;
// a declared error it returns is shown as it is, any other becomes the
// cause of the field's invalid error. Format turns a value given with Set
// back into T, for display.
type Conv[T, U any] struct {
	Parse  func(T) (U, error)
	Format func(U) T
}

// As appends c to the field's conversion chain and returns the handle of
// the new type. Rules added before As check T, rules after check U. The
// handle As was called on is dead: using it panics. Label, placeholder,
// help, required-ness, its errors, When, options and kind carry over. As
// after Set panics: call Set on the handle As returns.
//
//	f.Cooldown = f.Int("cooldown").Is(rule.Min(0)).As(seconds) // *Field[time.Duration]
func (h *Field[T]) As[U any](c Conv[T, U]) *Field[U] {
	fd := h.builder("As")
	if c.Parse == nil || c.Format == nil {
		panic(fmt.Sprintf("forms: As on field %q: a Conv needs both Parse and Format", fd.name))
	}
	if fd.vals != nil {
		panic(fmt.Sprintf("forms: As on field %q after Set; call Set on the handle As returns", fd.name))
	}
	fd.stages = append(fd.stages, &stage{
		parse:  func(v any) (any, error) { return c.Parse(typed[T](v)) },
		format: func(u any) any { return c.Format(typed[U](u)) },
	})
	fd.live = h.stage + 1
	return &Field[U]{f: fd, stage: fd.live}
}

// typed returns v as a T. A nil interface value boxed in an any is untyped,
// so it becomes T's zero value instead of panicking.
func typed[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}
