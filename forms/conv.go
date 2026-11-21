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
// help, required-ness, its errors, options and kind carry over.
//
//	f.Cooldown = f.Int("cooldown").Is(rule.Min(0)).As(seconds) // *Field[time.Duration]
func (h *Field[T]) As[U any](c Conv[T, U]) *Field[U] {
	fd := h.builder("As")
	if c.Parse == nil || c.Format == nil {
		panic(fmt.Sprintf("forms: As on field %q: a Conv needs both Parse and Format", fd.name))
	}
	fd.stages = append(fd.stages, &stage{
		parse:  func(v any) (any, error) { return c.Parse(v.(T)) },
		format: func(u any) any { return c.Format(u.(U)) },
	})
	fd.live = h.stage + 1
	return &Field[U]{f: fd, stage: fd.live}
}
