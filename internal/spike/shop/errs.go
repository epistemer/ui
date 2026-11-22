package shop

import (
	"context"

	"github.com/epistemer/errors"
)

var (
	errNothing  = errors.Key("errors.shop.nothing_chosen", "choose at least one order", errors.Invalid)
	errCantShip = errors.Key("errors.shop.cant_ship", "a refunded order can't be shipped", errors.Conflict)
)

func errorsT(ctx context.Context, err error) string { return errors.T(ctx, err) }
