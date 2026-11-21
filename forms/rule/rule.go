package rule

import (
	"context"
	"iter"

	"github.com/epistemer/errors"
)

// Rule checks a value of type T. The zero Rule passes every value.
type Rule[T any] struct {
	check func(ctx context.Context, v T) error
	err   error
	attrs []attr
	time  *timeBound // NotBefore and NotAfter, for hook.DateOnly
}

type attr struct{ name, value string }

// New makes a custom rule. check returns nil to pass, or the error people
// should see, which should be an errors declaration. New panics on a nil
// check.
func New[T any](check func(ctx context.Context, v T) error) Rule[T] {
	if check == nil {
		panic("rule: New(nil)")
	}
	return Rule[T]{check: check}
}

// Err returns a copy of r that reports err when it fails. It panics on an
// err people would see as Unknown: a nil error, or one with no public part
// in its tree, such as a Def1 not given its argument.
func (r Rule[T]) Err(err error) Rule[T] {
	mustBePublic("Err", err)
	r.err = err
	return r
}

// Attrs yields the HTML attributes the rule implies, such as maxlength.
func (r Rule[T]) Attrs() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, a := range r.attrs {
			if !yield(a.name, a.value) {
				return
			}
		}
	}
}

// Check runs rules on v in order and returns the first failure, or nil.
func Check[T any](ctx context.Context, v T, rules ...Rule[T]) error {
	for _, r := range rules {
		if err := r.run(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (r Rule[T]) run(ctx context.Context, v T) error {
	if r.check == nil {
		return nil
	}
	err := r.check(ctx, v)
	if err == nil {
		return nil
	}
	if r.err != nil {
		return r.err
	}
	return err
}

// with returns r with the HTML attributes a catalogue constructor implies.
func (r Rule[T]) with(name, value string) Rule[T] {
	r.attrs = append(r.attrs[:len(r.attrs):len(r.attrs)], attr{name, value})
	return r
}

// mustBePublic panics unless people would see err's own text.
func mustBePublic(where string, err error) {
	if err == nil {
		panic("rule: " + where + "(nil)")
	}
	for range errors.All(err) {
		return
	}
	panic("rule: " + where + ": people would see Unknown for " + err.Error() +
		"; pass an errors declaration (a Def1–Def3 needs .With)")
}
