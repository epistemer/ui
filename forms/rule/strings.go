package rule

import (
	"context"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MinLen checks that a string has at least n runes. It implies the minlength attribute.
func MinLen(n int) Rule[string] {
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
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if !anchored.MatchString(v) {
				return ErrMatches
			}
			return nil
		},
	}.with("pattern", re.String())
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
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
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

// PostCode checks that the string matches the postal code format for country.
// It panics at construction if country is unknown.
func PostCode(country string) Rule[string] {
	re, ok := postcodeRegexes[country]
	if !ok {
		panic("rule: unknown country for PostCode: " + country)
	}
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if !re.MatchString(v) {
				return ErrPostCode
			}
			return nil
		},
	}
}
