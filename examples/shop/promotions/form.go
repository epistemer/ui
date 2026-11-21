// Package promotions is the shop's promotion form: conditions, a group and
// a list (ui/forms design §11.2).
package promotions

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/examples/shop/models"
	"github.com/epistemer/ui/examples/shop/money"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

// ErrNameTaken is reported when another promotion has the name.
var ErrNameTaken = errors.Key("errors.shop.promotion_name_taken", "another promotion has this name", errors.Conflict)

// Service is what the form needs from the service layer.
type Service interface {
	PromotionNameTaken(ctx context.Context, name string) (bool, error)
}

var (
	zero    = decimal.Zero
	ten     = decimal.NewFromInt(10)
	hundred = decimal.NewFromInt(100)
)

// PercentageForm is the percentage group.
type PercentageForm struct {
	*forms.Form
	Value, Max, Min *forms.Field[decimal.Decimal]
}

func newPercentage(g *forms.Form) *PercentageForm {
	p := &PercentageForm{Form: g}
	p.Value = money.Decimal(g, "value").Required().Is(rule.AtLeast(zero), rule.AtMost(hundred))
	p.Max = money.Decimal(g, "max")
	p.Min = money.Decimal(g, "min").Required()
	return p
}

// PointsForm is the points group.
type PointsForm struct {
	*forms.Form
	Points *forms.Field[int]
}

func newPoints(g *forms.Form) *PointsForm {
	return &PointsForm{Form: g, Points: g.Int("points").Required().Is(rule.Min(1))}
}

// ConstraintForm is one item of the constraints list.
type ConstraintForm struct {
	*forms.Form
	Type                 *forms.Field[models.ConstraintType]
	MinAmount, MaxAmount *forms.Field[decimal.Decimal]
	MinQuantity          *forms.Field[int32]
}

func newConstraint(g *forms.Form) *ConstraintForm {
	c := &ConstraintForm{Form: g}
	c.Type = g.Enum("type", models.AllConstraintTypeValues()).Required()
	monetary := c.Type.Eq(models.ConstraintTypeMonetary)
	c.MinAmount = money.Decimal(g, "min_amount").When(monetary).Required()
	c.MaxAmount = money.Decimal(g, "max_amount").When(monetary)
	c.MinQuantity = g.Number[int32]("min_quantity").When(c.Type.Eq(models.ConstraintTypeQuantity)).Required().Is(rule.Min[int32](1))
	return c
}

// PromotionForm is the promotion form.
type PromotionForm struct {
	*forms.Form
	Name        *forms.Field[string]
	Type        *forms.Field[models.PromotionType]
	Base        *forms.Field[models.DiscountBase]
	MinQuantity *forms.Field[int32]
	Fixed       *forms.Field[decimal.Decimal]
	Absolute    *forms.Field[decimal.Decimal]
	Percentage  *PercentageForm
	Points      *PointsForm
	Constraints *forms.List[*ConstraintForm]
}

// NewPromotionForm declares the form; s checks that the name is free.
func NewPromotionForm(s Service) *PromotionForm {
	f := &PromotionForm{Form: forms.New()}

	f.Name = f.Text("name").Required().
		Is(rule.MinLen(3), rule.MaxLen(255), uniqueName(s))

	f.Type = f.Enum("type", models.AllPromotionTypeValues()).Required()
	is := f.Type.Eq // every condition below is typed against PromotionType

	f.Base = f.Enum("base", models.AllDiscountBaseValues()).
		When(is(models.PromotionTypePercentage)).Required()

	f.MinQuantity = f.Number[int32]("min_quantity").
		Is(rule.Between[int32](0, 100))

	f.Fixed = money.Decimal(f.Form, "fixed").
		When(is(models.PromotionTypeFixed)).Required().
		Is(rule.AtLeast(zero), rule.AtMost(ten))

	f.Absolute = money.Decimal(f.Form, "absolute").
		When(is(models.PromotionTypeAbsolute)).Required().
		Is(rule.AtLeast(zero), rule.AtMost(hundred))

	f.Percentage = f.Group("percentage", newPercentage)
	f.Percentage.When(is(models.PromotionTypePercentage))

	f.Points = f.Group("points", newPoints)
	f.Points.When(is(models.PromotionTypePoints))

	f.Constraints = f.List("constraints", newConstraint).MinItems(1)

	return f
}

func uniqueName(s Service) rule.Rule[string] {
	return rule.New(func(ctx context.Context, name string) error {
		taken, err := s.PromotionNameTaken(ctx, name)
		if err != nil {
			return err
		}
		if taken {
			return ErrNameTaken
		}
		return nil
	})
}
