package rule_test

import (
	"context"
	"fmt"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/rule"
)

func ExampleCheck() {
	ctx := context.Background()
	minRule := rule.MinLen(3)
	maxRule := rule.MaxLen(10)

	err := rule.Check(ctx, "hello", minRule, maxRule)
	fmt.Printf("valid: %v\n", err == nil)

	err = rule.Check(ctx, "hi", minRule, maxRule)
	fmt.Printf("error: %s\n", errors.T(ctx, err))

	// Output:
	// valid: true
	// error: must be at least 3 characters
}

var errCode = errors.Key("errors.example.code", "code must be at least 5 characters", errors.Invalid)

func ExampleRule_Err() {
	ctx := context.Background()
	r := rule.MinLen(5).Err(errCode)

	err := rule.Check(ctx, "abc", r)
	fmt.Printf("error: %s\n", errors.T(ctx, err))

	// Output:
	// error: code must be at least 5 characters
}
