package rule

import (
	"context"
	"slices"
)

// MinItems checks that the slice contains at least n elements.
func MinItems[T any](n int) Rule[[]T] {
	return Rule[[]T]{
		check: func(_ context.Context, v []T) error {
			if len(v) < n {
				return ErrMinItems.With(n)
			}
			return nil
		},
	}
}

// MaxItems checks that the slice contains at most n elements.
func MaxItems[T any](n int) Rule[[]T] {
	return Rule[[]T]{
		check: func(_ context.Context, v []T) error {
			if len(v) > n {
				return ErrMaxItems.With(n)
			}
			return nil
		},
	}
}

// Each validates each item of the slice against rules, returning the first failing item's error.
func Each[T any](rules ...Rule[T]) Rule[[]T] {
	copied := slices.Clone(rules)
	return Rule[[]T]{
		check: func(ctx context.Context, v []T) error {
			for _, item := range v {
				if err := Check(ctx, item, copied...); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
