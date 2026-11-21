package rule_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/rule"
)

func TestMinLen(t *testing.T) {
	ctx := context.Background()
	r := rule.MinLen(3)

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"abc", true},
		{"héé", true}, // 3 runes, 5 bytes
		{"abcd", true},
		{"ab", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("MinLen(3) on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrMinLen) {
			t.Errorf("MinLen(3) on %q: got %v, want ErrMinLen", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "minlength" && v == "3" {
			found = true
		}
	}
	if !found {
		t.Error("MinLen(3) missing minlength=3 attribute")
	}
}

func TestMaxLen(t *testing.T) {
	ctx := context.Background()
	r := rule.MaxLen(3)

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"abc", true},
		{"héé", true}, // 3 runes, 5 bytes: must pass rune count
		{"hééé", false},
		{"abcd", false},
		{"", true},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("MaxLen(3) on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrMaxLen) {
			t.Errorf("MaxLen(3) on %q: got %v, want ErrMaxLen", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "maxlength" && v == "3" {
			found = true
		}
	}
	if !found {
		t.Error("MaxLen(3) missing maxlength=3 attribute")
	}
}

func TestMatches(t *testing.T) {
	ctx := context.Background()
	re := regexp.MustCompile("[a-z]+")
	r := rule.Matches(re)

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"abc", true},
		{"abc1", false}, // must be anchored
		{"1abc", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("Matches on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrMatches) {
			t.Errorf("Matches on %q: got %v, want ErrMatches", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "pattern" && v == re.String() {
			found = true
		}
	}
	if !found {
		t.Errorf("Matches missing pattern=%s attribute", re.String())
	}

	defer func() {
		if recover() == nil {
			t.Error("Matches(nil) did not panic")
		}
	}()
	rule.Matches(nil)
}

func TestEmail(t *testing.T) {
	ctx := context.Background()
	r := rule.Email()

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"user@example.com", true},
		{"user.name+tag@sub.example.co.uk", true},
		{"Barry <barry@example.com>", false}, // display name must fail
		{"<barry@example.com>", false},
		{"barry@example.com (Barry)", false},
		{"plainaddress", false},
		{"@missinguser.com", false},
		{"missingdomain@", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("Email on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrEmail) {
			t.Errorf("Email on %q: got %v, want ErrEmail", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "type" && v == "email" {
			found = true
		}
	}
	if !found {
		t.Error("Email missing type=email attribute")
	}
}

func TestURL(t *testing.T) {
	ctx := context.Background()
	r := rule.URL()

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://example.com/path?foo=bar#frag", true},
		{"javascript:alert(1)", false},
		{"/relative", false},
		{"ftp://example.com", false},
		{"http://", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("URL on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrURL) {
			t.Errorf("URL on %q: got %v, want ErrURL", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "type" && v == "url" {
			found = true
		}
	}
	if !found {
		t.Error("URL missing type=url attribute")
	}
}

func TestPhone(t *testing.T) {
	ctx := context.Background()
	r := rule.Phone()

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"+1 (555) 234-5678", true},
		{"+44 20 7946 0991", true},
		{"+201012345678", true},
		{"+1234567890", true},
		{"(555) 234-5678", false},    // missing plus
		{"5552345678", false},        // missing plus
		{"+0123456789", false},       // starts with 0
		{"+12345", false},            // too short
		{"+1234567890123456", false}, // too long
		{"+1 (555) ABC-1234", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("Phone on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrPhone) {
			t.Errorf("Phone on %q: got %v, want ErrPhone", tc.input, err)
		}
	}

	found := false
	for k, v := range r.Attrs() {
		if k == "type" && v == "tel" {
			found = true
		}
	}
	if !found {
		t.Error("Phone missing type=tel attribute")
	}
}

func TestCountryCode(t *testing.T) {
	ctx := context.Background()
	r := rule.CountryCode()

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"US", true},
		{"GB", true},
		{"DE", true},
		{"FR", true},
		{"EG", true},
		{"us", false},
		{"USA", false},
		{"ZZ", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("CountryCode on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrCountryCode) {
			t.Errorf("CountryCode on %q: got %v, want ErrCountryCode", tc.input, err)
		}
	}

	for k, v := range r.Attrs() {
		t.Errorf("CountryCode should have no attributes, got %s=%s", k, v)
	}
}

func TestCurrencyCode(t *testing.T) {
	ctx := context.Background()
	r := rule.CurrencyCode()

	for _, tc := range []struct {
		input string
		pass  bool
	}{
		{"USD", true},
		{"EUR", true},
		{"GBP", true},
		{"JPY", true},
		{"VED", true},
		{"ZWG", true},
		{"usd", false},
		{"US", false},
		{"ZZZ", false},
		{"", false},
	} {
		err := rule.Check(ctx, tc.input, r)
		if tc.pass && err != nil {
			t.Errorf("CurrencyCode on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrCurrencyCode) {
			t.Errorf("CurrencyCode on %q: got %v, want ErrCurrencyCode", tc.input, err)
		}
	}

	for k, v := range r.Attrs() {
		t.Errorf("CurrencyCode should have no attributes, got %s=%s", k, v)
	}
}

func TestPostCode(t *testing.T) {
	ctx := context.Background()
	usRule := rule.PostCode("US")
	gbRule := rule.PostCode("GB")

	for _, tc := range []struct {
		r     rule.Rule[string]
		input string
		pass  bool
	}{
		{usRule, "12345", true},
		{usRule, "12345-6789", true},
		{usRule, "1234", false},
		{usRule, "ABCDE", false},
		{gbRule, "SW1A 1AA", true},
		{gbRule, "EC1A 1BB", true},
		{gbRule, "12345", false},
	} {
		err := rule.Check(ctx, tc.input, tc.r)
		if tc.pass && err != nil {
			t.Errorf("PostCode on %q: unexpected error %v", tc.input, err)
		}
		if !tc.pass && !errors.Is(err, rule.ErrPostCode) {
			t.Errorf("PostCode on %q: got %v, want ErrPostCode", tc.input, err)
		}
	}

	for k, v := range usRule.Attrs() {
		t.Errorf("PostCode should have no attributes, got %s=%s", k, v)
	}

	defer func() {
		if recover() == nil {
			t.Error("PostCode with unknown country did not panic")
		}
	}()
	rule.PostCode("UNKNOWN_COUNTRY")
}
