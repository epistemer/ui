package rule_test

import (
	"context"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/rule"
)

type myInt int

type cmpInt int

func (c cmpInt) Cmp(other cmpInt) int {
	if c < other {
		return -1
	}
	if c > other {
		return 1
	}
	return 0
}

func TestMin(t *testing.T) {
	ctx := context.Background()

	t.Run("int", func(t *testing.T) {
		r := rule.Min(5)
		if err := rule.Check(ctx, 5, r); err != nil {
			t.Errorf("5 should pass Min(5): %v", err)
		}
		if err := rule.Check(ctx, 6, r); err != nil {
			t.Errorf("6 should pass Min(5): %v", err)
		}
		err := rule.Check(ctx, 4, r)
		if !errors.Is(err, rule.ErrMin) {
			t.Errorf("4 should fail Min(5), got %v", err)
		}
		if err != nil && err.Error() != "must be at least 5" {
			t.Errorf("error text %q, want 'must be at least 5'", err.Error())
		}

		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["min"] != "5" {
			t.Errorf("min attr %q, want '5'", attrs["min"])
		}
	})

	t.Run("float", func(t *testing.T) {
		r := rule.Min(3.14)
		if err := rule.Check(ctx, 3.14, r); err != nil {
			t.Errorf("3.14 should pass Min(3.14): %v", err)
		}
		if err := rule.Check(ctx, 3.13, r); !errors.Is(err, rule.ErrMin) {
			t.Errorf("3.13 should fail Min(3.14), got %v", err)
		}
		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["min"] != "3.14" {
			t.Errorf("min attr %q, want '3.14'", attrs["min"])
		}
	})

	t.Run("string", func(t *testing.T) {
		r := rule.Min("c")
		if err := rule.Check(ctx, "c", r); err != nil {
			t.Errorf("'c' should pass Min('c'): %v", err)
		}
		if err := rule.Check(ctx, "b", r); !errors.Is(err, rule.ErrMin) {
			t.Errorf("'b' should fail Min('c'), got %v", err)
		}
		for k, v := range r.Attrs() {
			t.Errorf("strings must have no min/max attrs, got %s=%s", k, v)
		}
	})

	t.Run("custom int", func(t *testing.T) {
		r := rule.Min(myInt(10))
		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["min"] != "10" {
			t.Errorf("underlying int should have min=10, got %q", attrs["min"])
		}
	})
}

func TestMax(t *testing.T) {
	ctx := context.Background()

	t.Run("int", func(t *testing.T) {
		r := rule.Max(10)
		if err := rule.Check(ctx, 10, r); err != nil {
			t.Errorf("10 should pass Max(10): %v", err)
		}
		if err := rule.Check(ctx, 9, r); err != nil {
			t.Errorf("9 should pass Max(10): %v", err)
		}
		err := rule.Check(ctx, 11, r)
		if !errors.Is(err, rule.ErrMax) {
			t.Errorf("11 should fail Max(10), got %v", err)
		}
		if err != nil && err.Error() != "must be at most 10" {
			t.Errorf("error text %q, want 'must be at most 10'", err.Error())
		}

		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["max"] != "10" {
			t.Errorf("max attr %q, want '10'", attrs["max"])
		}
	})

	t.Run("float", func(t *testing.T) {
		r := rule.Max(10.5)
		if err := rule.Check(ctx, 10.5, r); err != nil {
			t.Errorf("10.5 should pass Max(10.5): %v", err)
		}
		if err := rule.Check(ctx, 10.6, r); !errors.Is(err, rule.ErrMax) {
			t.Errorf("10.6 should fail Max(10.5), got %v", err)
		}
		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["max"] != "10.5" {
			t.Errorf("max attr %q, want '10.5'", attrs["max"])
		}
	})

	t.Run("string", func(t *testing.T) {
		r := rule.Max("m")
		if err := rule.Check(ctx, "m", r); err != nil {
			t.Errorf("'m' should pass Max('m'): %v", err)
		}
		if err := rule.Check(ctx, "n", r); !errors.Is(err, rule.ErrMax) {
			t.Errorf("'n' should fail Max('m'), got %v", err)
		}
		for k, v := range r.Attrs() {
			t.Errorf("strings must have no min/max attrs, got %s=%s", k, v)
		}
	})
}

func TestBetween(t *testing.T) {
	ctx := context.Background()

	t.Run("int", func(t *testing.T) {
		r := rule.Between(1, 5)
		for _, tc := range []struct {
			val  int
			pass bool
		}{
			{0, false},
			{1, true},
			{3, true},
			{5, true},
			{6, false},
		} {
			err := rule.Check(ctx, tc.val, r)
			if tc.pass && err != nil {
				t.Errorf("%d should pass Between(1, 5): %v", tc.val, err)
			}
			if !tc.pass && !errors.Is(err, rule.ErrBetween) {
				t.Errorf("%d should fail Between(1, 5), got %v", tc.val, err)
			}
			if !tc.pass && err != nil && err.Error() != "must be between 1 and 5" {
				t.Errorf("error text %q, want 'must be between 1 and 5'", err.Error())
			}
		}

		attrs := map[string]string{}
		for k, v := range r.Attrs() {
			attrs[k] = v
		}
		if attrs["min"] != "1" || attrs["max"] != "5" {
			t.Errorf("Between(1, 5) attrs got min=%q max=%q, want 1 and 5", attrs["min"], attrs["max"])
		}
	})

	t.Run("panic on lo > hi", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("Between(5, 1) did not panic")
			}
		}()
		rule.Between(5, 1)
	})

	t.Run("string", func(t *testing.T) {
		r := rule.Between("b", "f")
		if err := rule.Check(ctx, "a", r); !errors.Is(err, rule.ErrBetween) {
			t.Errorf("'a' should fail Between('b', 'f'), got %v", err)
		}
		if err := rule.Check(ctx, "c", r); err != nil {
			t.Errorf("'c' should pass Between('b', 'f'): %v", err)
		}
		for k, v := range r.Attrs() {
			t.Errorf("strings must have no min/max attrs, got %s=%s", k, v)
		}
	})
}

func TestAtLeastAndAtMost(t *testing.T) {
	ctx := context.Background()

	t.Run("AtLeast", func(t *testing.T) {
		r := rule.AtLeast(cmpInt(10))
		if err := rule.Check(ctx, cmpInt(10), r); err != nil {
			t.Errorf("10 should pass AtLeast(10): %v", err)
		}
		if err := rule.Check(ctx, cmpInt(11), r); err != nil {
			t.Errorf("11 should pass AtLeast(10): %v", err)
		}
		err := rule.Check(ctx, cmpInt(9), r)
		if !errors.Is(err, rule.ErrMin) {
			t.Errorf("9 should fail AtLeast(10), got %v", err)
		}
		if err != nil && err.Error() != "must be at least 10" {
			t.Errorf("error text %q, want 'must be at least 10'", err.Error())
		}
		for k, v := range r.Attrs() {
			t.Errorf("AtLeast should have no attrs, got %s=%s", k, v)
		}
	})

	t.Run("AtMost", func(t *testing.T) {
		r := rule.AtMost(cmpInt(10))
		if err := rule.Check(ctx, cmpInt(10), r); err != nil {
			t.Errorf("10 should pass AtMost(10): %v", err)
		}
		if err := rule.Check(ctx, cmpInt(9), r); err != nil {
			t.Errorf("9 should pass AtMost(10): %v", err)
		}
		err := rule.Check(ctx, cmpInt(11), r)
		if !errors.Is(err, rule.ErrMax) {
			t.Errorf("11 should fail AtMost(10), got %v", err)
		}
		if err != nil && err.Error() != "must be at most 10" {
			t.Errorf("error text %q, want 'must be at most 10'", err.Error())
		}
		for k, v := range r.Attrs() {
			t.Errorf("AtMost should have no attrs, got %s=%s", k, v)
		}
	})
}

func TestOneOf(t *testing.T) {
	ctx := context.Background()
	r := rule.OneOf("apple", "banana", "cherry")

	if err := rule.Check(ctx, "banana", r); err != nil {
		t.Errorf("'banana' should pass OneOf: %v", err)
	}
	err := rule.Check(ctx, "orange", r)
	if !errors.Is(err, rule.ErrOneOf) {
		t.Errorf("'orange' should fail OneOf, got %v", err)
	}

	for k, v := range r.Attrs() {
		t.Errorf("OneOf should have no attrs, got %s=%s", k, v)
	}
}
