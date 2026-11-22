package rule_test

import (
	"context"
	"fmt"
	"math"
	"regexp"
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
	date := i18n.Static("2026-11-21")
	for _, tc := range []struct {
		err  error
		key  string
		text string
	}{
		{rule.ErrMinLen.With(3), "min_len", "must be at least 3 characters"},
		{rule.ErrMaxLen.With(255), "max_len", "must be at most 255 characters"},
		{rule.ErrMatches, "matches", "is not in the expected format"},
		{rule.ErrEmail, "email", "must be an email address"},
		{rule.ErrURL, "url", "must be a URL"},
		{rule.ErrPhone, "phone", "must be a phone number"},
		{rule.ErrCountryCode, "country_code", "must be a country code"},
		{rule.ErrCurrencyCode, "currency_code", "must be a currency code"},
		{rule.ErrPostCode, "post_code", "must be a postal code"},
		{rule.ErrMin.With("0"), "min", "must be at least 0"},
		{rule.ErrMax.With("9"), "max", "must be at most 9"},
		{rule.ErrBetween.With("1", "9"), "between", "must be between 1 and 9"},
		{rule.ErrOneOf, "one_of", "is not one of the allowed values"},
		{rule.ErrNotBefore.With(date), "not_before", "must not be before 2026-11-21"},
		{rule.ErrNotAfter.With(date), "not_after", "must not be after 2026-11-21"},
		{rule.ErrMinSize.With("1 KB"), "min_size", "must be at least 1 KB"},
		{rule.ErrMaxSize.With("5 MB"), "max_size", "must be at most 5 MB"},
		{rule.ErrContentType.With("image/png, image/jpeg"), "content_type", "must be one of these file types: image/png, image/jpeg"},
		{rule.ErrMinFiles.With(1), "min_files", "choose at least 1"},
		{rule.ErrMaxFiles.With(3), "max_files", "choose at most 3"},
		{rule.ErrMinItems.With(1), "min_items", "choose at least 1"},
		{rule.ErrMaxItems.With(4), "max_items", "choose at most 4"},
	} {
		key := "errors.forms.rule." + tc.key
		if got := errors.KindOf(tc.err); got != errors.Invalid {
			t.Errorf("%s: kind %v, want invalid", key, got)
		}
		if got := errors.Public(tc.err).K(); got != key {
			t.Errorf("key %q, want %q", got, key)
		}
		if got := errors.T(context.Background(), tc.err); got != tc.text {
			t.Errorf("%s: text %q, want %q", key, got, tc.text)
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

func TestReleaseRuleFixes(t *testing.T) {
	nan := math.NaN()
	for name, fn := range map[string]func(){
		"Min NaN":     func() { rule.Min(nan) },
		"Between NaN": func() { rule.Between(0, nan) },
		"MinLen -1":   func() { rule.MinLen(-1) },
		"MaxSize -1":  func() { rule.MaxSize(-1) },
		"MinFiles -1": func() { rule.MinFiles(-1) },
		"MaxFiles -1": func() { rule.MaxFiles(-1) },
		"MinItems -1": func() { rule.MinItems[string](-1) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
	for k := range rule.Matches(regexp.MustCompile(`(?i)abc`)).Attrs() {
		t.Errorf("a Go-only expression must not become a pattern attribute (%s)", k)
	}
	n := 0
	for range rule.Matches(regexp.MustCompile(`[a-z]{3}(?:-x)?`)).Attrs() {
		n++
	}
	if n != 1 {
		t.Error("a portable expression keeps its pattern attribute")
	}
	for _, bad := range []string{"http://:80", "http://a b.com/"} {
		if err := rule.Check(context.Background(), bad, rule.URL()); err == nil {
			t.Errorf("URL accepted %q", bad)
		}
	}
}
