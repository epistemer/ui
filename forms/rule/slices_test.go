package rule_test

import (
	"context"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/rule"
)

func TestMinItems(t *testing.T) {
	ctx := context.Background()
	r := rule.MinItems[int](2)

	for _, tc := range []struct {
		input []int
		pass  bool
	}{
		{[]int{1, 2}, true},
		{[]int{1, 2, 3}, true},
		{[]int{1}, false},
		{[]int{}, false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("MinItems(2) on %v: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrMinItems) {
			t.Errorf("MinItems(2) on %v: got %v, want ErrMinItems", tc.input, err)
		}
		if !tc.pass && err != nil && err.Error() != "choose at least 2" {
			t.Errorf("error text %q, want 'choose at least 2'", err.Error())
		}
	}
}

func TestMaxItems(t *testing.T) {
	ctx := context.Background()
	r := rule.MaxItems[string](2)

	for _, tc := range []struct {
		input []string
		pass  bool
	}{
		{[]string{"a", "b"}, true},
		{[]string{"a"}, true},
		{[]string{}, true},
		{[]string{"a", "b", "c"}, false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("MaxItems(2) on %v: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrMaxItems) {
			t.Errorf("MaxItems(2) on %v: got %v, want ErrMaxItems", tc.input, err)
		}
		if !tc.pass && err != nil && err.Error() != "choose at most 2" {
			t.Errorf("error text %q, want 'choose at most 2'", err.Error())
		}
	}
}

func TestEach(t *testing.T) {
	ctx := context.Background()
	r := rule.Each(rule.MinLen(2), rule.MaxLen(4))

	if err := rule.Check(ctx, []string{"ab", "abc", "abcd"}, r); err != nil {
		t.Errorf("expected slice to pass Each: %v", err)
	}

	// First item "ab" passes, second item "a" fails MinLen(2)
	err := rule.Check(ctx, []string{"ab", "a", "too_long"}, r)
	if !errors.Is(err, rule.ErrMinLen) {
		t.Errorf("expected ErrMinLen on second item, got %v", err)
	}

	// Empty slice passes
	if err := rule.Check(ctx, []string{}, r); err != nil {
		t.Errorf("empty slice should pass Each: %v", err)
	}
}
