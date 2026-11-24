package rule

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MinLen checks that a string has at least n runes. It implies the minlength attribute.
func MinLen(n int) Rule[string] {
	mustNotBeNegative("MinLen", int64(n))
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if utf8.RuneCountInString(v) < n {
				return ErrMinLen.With(n)
			}
			return nil
		},
	}.with("minlength", strconv.Itoa(n))
}

// MaxLen checks that a string has at most n runes. It implies the maxlength attribute.
func MaxLen(n int) Rule[string] {
	mustNotBeNegative("MaxLen", int64(n))
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if utf8.RuneCountInString(v) > n {
				return ErrMaxLen.With(n)
			}
			return nil
		},
	}.with("maxlength", strconv.Itoa(n))
}

// Matches checks that the whole string matches re. It compiles an anchored expression
// at construction. Matches panics if re is nil. It implies the pattern attribute.
func Matches(re *regexp.Regexp) Rule[string] {
	if re == nil {
		panic("rule: Matches(nil)")
	}
	anchored := regexp.MustCompile("^(?:" + re.String() + ")$")
	r := Rule[string]{
		check: func(_ context.Context, v string) error {
			if !anchored.MatchString(v) {
				return ErrMatches
			}
			return nil
		},
	}
	if jsCompatible(re.String()) {
		r = r.with("pattern", re.String())
	}
	return r
}

// Email checks that the string is a valid email address with no display name.
// It implies type="email".
func Email() Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			addr, err := mail.ParseAddress(v)
			if err != nil || addr.Name != "" || addr.Address != v {
				return ErrEmail
			}
			return nil
		},
	}.with("type", "email")
}

// URL checks that the string is an absolute http or https URL with a non-empty host.
// It implies type="url".
func URL() Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			u, err := url.Parse(v)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || strings.ContainsAny(u.Host, " \t") {
				return ErrURL
			}
			return nil
		},
	}.with("type", "url")
}

var phoneRegex = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// Phone checks that the string, after stripping spaces, dashes, dots, and parentheses,
// matches E.164 with a leading plus sign. It implies type="tel".
func Phone() Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			cleaned := strings.Map(func(r rune) rune {
				switch r {
				case ' ', '-', '.', '(', ')':
					return -1
				default:
					return r
				}
			}, v)
			if !phoneRegex.MatchString(cleaned) {
				return ErrPhone
			}
			return nil
		},
	}.with("type", "tel")
}

// CountryCode checks that the string is a valid ISO 3166-1 alpha-2 upper-case code.
func CountryCode() Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if _, ok := iso3166Alpha2[v]; !ok {
				return ErrCountryCode
			}
			return nil
		},
	}
}

// CurrencyCode checks that the string is a valid ISO 4217 alphabetic upper-case code.
func CurrencyCode() Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if _, ok := iso4217[v]; !ok {
				return ErrCurrencyCode
			}
			return nil
		},
	}
}

// PostCode checks the format of a postal code for country, an ISO 3166-1
// alpha-2 code, compared trimmed and in upper case. A country it doesn't
// know passes every value.
func PostCode(country string) Rule[string] {
	re, ok := postcodeRegexes[strings.ToUpper(strings.TrimSpace(country))]
	if !ok {
		return Rule[string]{} // the zero Rule passes every value
	}
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if !re.MatchString(strings.ToUpper(v)) {
				return ErrPostCode
			}
			return nil
		},
	}
}

// HasPostCode reports whether PostCode knows country's format, comparing
// as PostCode does.
func HasPostCode(country string) bool {
	_, ok := postcodeRegexes[strings.ToUpper(strings.TrimSpace(country))]
	return ok
}

// goOnly lists regexp syntax that RE2 accepts and a browser's pattern
// attribute does not: flags, named groups in Go's form, \A and \z, POSIX
// classes. With any of it, Matches checks on the server only.
var goOnly = []string{"(?i", "(?m", "(?s", "(?U", "(?P<", `\A`, `\z`, "[[:"}

func jsCompatible(expr string) bool {
	for _, s := range goOnly {
		if strings.Contains(expr, s) {
			return false
		}
	}
	return true
}

func mustNotBeNegative(where string, n int64) {
	if n < 0 {
		panic(fmt.Sprintf("rule: %s(%d): a limit can't be negative", where, n))
	}
}
