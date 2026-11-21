package rule

import (
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
)

// The catalogue's default errors, all of kind errors.Invalid. Keys are
// permanent: renaming one orphans its translations. Texts follow errors'
// convention, lower case with no final period; the UI capitalizes them.
var (
	ErrMinLen       = errors.Key1[int]("errors.forms.rule.min_len", "must be at least %d characters", errors.Invalid)
	ErrMaxLen       = errors.Key1[int]("errors.forms.rule.max_len", "must be at most %d characters", errors.Invalid)
	ErrMatches      = errors.Key("errors.forms.rule.matches", "is not in the expected format", errors.Invalid)
	ErrEmail        = errors.Key("errors.forms.rule.email", "must be an email address", errors.Invalid)
	ErrURL          = errors.Key("errors.forms.rule.url", "must be a URL", errors.Invalid)
	ErrPhone        = errors.Key("errors.forms.rule.phone", "must be a phone number", errors.Invalid)
	ErrCountryCode  = errors.Key("errors.forms.rule.country_code", "must be a country code", errors.Invalid)
	ErrCurrencyCode = errors.Key("errors.forms.rule.currency_code", "must be a currency code", errors.Invalid)
	ErrPostCode     = errors.Key("errors.forms.rule.post_code", "must be a postal code", errors.Invalid)
	ErrMin          = errors.Key1[string]("errors.forms.rule.min", "must be at least %s", errors.Invalid)
	ErrMax          = errors.Key1[string]("errors.forms.rule.max", "must be at most %s", errors.Invalid)
	ErrBetween      = errors.Key2[string, string]("errors.forms.rule.between", "must be between %s and %s", errors.Invalid)
	ErrOneOf        = errors.Key("errors.forms.rule.one_of", "is not one of the allowed values", errors.Invalid)
	ErrNotBefore    = errors.Key1[i18n.Ter]("errors.forms.rule.not_before", "must not be before %s", errors.Invalid)
	ErrNotAfter     = errors.Key1[i18n.Ter]("errors.forms.rule.not_after", "must not be after %s", errors.Invalid)
	ErrMinSize      = errors.Key1[string]("errors.forms.rule.min_size", "must be at least %s", errors.Invalid)
	ErrMaxSize      = errors.Key1[string]("errors.forms.rule.max_size", "must be at most %s", errors.Invalid)
	ErrContentType  = errors.Key1[string]("errors.forms.rule.content_type", "must be one of these file types: %s", errors.Invalid)
	ErrMinItems     = errors.Key1[int]("errors.forms.rule.min_items", "choose at least %d", errors.Invalid)
	ErrMaxItems     = errors.Key1[int]("errors.forms.rule.max_items", "choose at most %d", errors.Invalid)
)
