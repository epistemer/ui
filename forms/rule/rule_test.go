package rule_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/i18ntest"
	"github.com/epistemer/ui/forms/rule"
)

var (
	errTaken  = errors.Key("errors.test.taken", "is taken", errors.Conflict)
	errCustom = errors.Key("errors.test.custom", "is custom", errors.Invalid)
	errArg    = errors.Key1[int]("errors.test.arg", "needs %d", errors.Invalid)
)

func failWith(err error) rule.Rule[string] {
	return rule.New(func(context.Context, string) error { return err })
}

var pass = rule.New(func(context.Context, string) error { return nil })

func TestCheckRunsInOrderAndStops(t *testing.T) {
	var ran []string
	mark := func(name string, err error) rule.Rule[string] {
		return rule.New(func(context.Context, string) error { ran = append(ran, name); return err })
	}
	err := rule.Check(context.Background(), "x", mark("a", nil), mark("b", errTaken), mark("c", nil))
	if !errors.Is(err, errTaken) {
		t.Fatalf("got %v, want errTaken", err)
	}
	if !slices.Equal(ran, []string{"a", "b"}) {
		t.Errorf("ran %v, want [a b]: rules stop at the first failure", ran)
	}
	if err := rule.Check(context.Background(), "x"); err != nil {
		t.Errorf("no rules: %v", err)
	}
	if err := rule.Check(context.Background(), "x", rule.Rule[string]{}, pass); err != nil {
		t.Errorf("the zero Rule must pass: %v", err)
	}
}

func TestErrReplacesTheError(t *testing.T) {
	r := failWith(errTaken)
	custom := r.Err(errCustom)
	ctx := context.Background()
	if err := rule.Check(ctx, "x", custom); !errors.Is(err, errCustom) {
		t.Errorf("Err: got %v, want errCustom", err)
	}
	if err := rule.Check(ctx, "x", r); !errors.Is(err, errTaken) {
		t.Errorf("Err must return a copy; the original now reports %v", err)
	}
	if err := rule.Check(ctx, "x", pass.Err(errCustom)); err != nil {
		t.Errorf("a passing rule with Err: %v", err)
	}
	if err := rule.Check(ctx, "x", failWith(errTaken).Err(errArg.With(3))); !errors.Is(err, errArg) {
		t.Errorf("Err with an occurrence: %v", err)
	}
	if err := rule.Check(ctx, "x", failWith(errTaken).Err(fmt.Errorf("context: %w", errCustom))); !errors.Is(err, errCustom) {
		t.Errorf("Err with a wrapped declaration: %v", err)
	}
}

func TestDeclarationPanics(t *testing.T) {
	for name, f := range map[string]func(){
		"New(nil)":              func() { rule.New[string](nil) },
		"Err(nil)":              func() { pass.Err(nil) },
		"Err(private)":          func() { pass.Err(fmt.Errorf("plain")) },
		"Err(Def1 without arg)": func() { pass.Err(errArg) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			f()
		})
	}
}

func TestAttrsOfACustomRule(t *testing.T) {
	for k, v := range pass.Attrs() {
		t.Errorf("a custom rule implies no attributes, got %s=%q", k, v)
	}
}

func TestDefaultErrors(t *testing.T) {
	for _, tc := range []struct {
		err  error
		key  string
		text string
	}{
		{rule.ErrMaxLen.With(255), "errors.forms.rule.max_len", "must be at most 255 characters"},
		{rule.ErrEmail, "errors.forms.rule.email", "must be an email address"},
		{rule.ErrBetween.With("1", "9"), "errors.forms.rule.between", "must be between 1 and 9"},
	} {
		if got := errors.KindOf(tc.err); got != errors.Invalid {
			t.Errorf("%s: kind %v, want invalid", tc.key, got)
		}
		if got := errors.Public(tc.err).K(); got != tc.key {
			t.Errorf("key %q, want %q", got, tc.key)
		}
		if got := tc.err.Error(); got != tc.text {
			t.Errorf("%s: Error() %q, want %q", tc.key, got, tc.text)
		}
	}
}

func TestDefaultErrorTranslated(t *testing.T) {
	tr := i18ntest.New(t, i18n.Locale{Code: "en", NativeName: "English"})
	ctx := i18ntest.Context(t, tr, "en")
	if err := tr.Set("en", "errors.forms.rule.max_len", "no more than %d letters"); err != nil {
		t.Fatal(err)
	}
	if got := errors.T(ctx, rule.ErrMaxLen.With(10)); !strings.Contains(got, "no more than 10 letters") {
		t.Errorf("got %q", got)
	}
}
