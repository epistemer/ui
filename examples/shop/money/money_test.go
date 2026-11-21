package money_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/examples/shop/money"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

func TestDecimalPrecision(t *testing.T) {
	const exact = "0.10000000000000000001"
	f := forms.New()
	d := money.Decimal(f, "price").Is(rule.AtLeast(decimal.Zero), rule.AtMost(decimal.NewFromInt(1)))
	neg := money.Decimal(f, "neg").Is(rule.AtLeast(decimal.Zero))
	f.Bind(forms.FromValues(url.Values{"price": {exact}, "neg": {"-0.5"}}))
	f.Validate(context.Background())
	if d.Err() != nil || d.Value().String() != exact {
		t.Errorf("price %s, %v: want %s exactly", d.Value(), d.Err(), exact)
	}
	if !errors.Is(neg.Err(), rule.ErrMin) {
		t.Errorf("neg: %v", neg.Err())
	}
}
