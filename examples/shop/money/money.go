// Package money is the shop's decimal fields: forms never imports a
// decimal library, so the app wraps Parsed once.
package money

import (
	"github.com/shopspring/decimal"

	"github.com/epistemer/ui/forms"
)

// Decimal declares a decimal field.
func Decimal(f *forms.Form, name string) *forms.Field[decimal.Decimal] {
	return f.Parsed[decimal.Decimal](name)
}
