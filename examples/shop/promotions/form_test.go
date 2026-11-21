package promotions_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/epistemer/ui/examples/shop/models"
	"github.com/epistemer/ui/examples/shop/promotions"
	"github.com/epistemer/ui/forms/formstest"
)

type service struct{ taken string }

func (s service) PromotionNameTaken(_ context.Context, name string) (bool, error) {
	return name == s.taken, nil
}

var bg = context.Background()

func TestFixedWithAMonetaryConstraint(t *testing.T) {
	f := promotions.NewPromotionForm(service{})
	ok := formstest.Submit(t, bg, f.Form, "name=Gold&type=fixed&fixed=2.50"+
		"&constraints[0].type=monetary&constraints[0].min_amount=100&constraints[0].max_amount=5000"+
		"&percentage.value=150") // stale: the group is inactive
	if !ok {
		t.Fatalf("errors %q", formstest.Errors(f.Form))
	}
	if f.Fixed.Value().String() != "2.5" || f.Percentage.Active() || f.Percentage.Value.Ptr() != nil {
		t.Errorf("fixed %s, percentage active %v", f.Fixed.Value(), f.Percentage.Active())
	}
	c := f.Constraints.Items()[0]
	if c.Type.Value() != models.ConstraintTypeMonetary || c.MinAmount.Value().String() != "100" || c.MinQuantity.Ptr() != nil {
		t.Errorf("constraint %v %v", c.Type.Value(), c.MinAmount.Value())
	}
}

func TestErrors(t *testing.T) {
	f := promotions.NewPromotionForm(service{taken: "Gold"})
	formstest.Submit(t, bg, f.Form, "name=Gold&type=percentage&base=total&percentage.value=150"+
		"&constraints[2].type=quantity&constraints[5].type=monetary")
	want := []string{
		"name: errors.shop.promotion_name_taken",
		"percentage.value: errors.forms.rule.max",
		"percentage.min: errors.forms.required",
		"constraints[0].min_quantity: errors.forms.required",
		"constraints[1].min_amount: errors.forms.required",
	}
	if got := formstest.Errors(f.Form); !slices.Equal(got, want) {
		t.Errorf("errors\ngot  %q\nwant %q", got, want)
	}
}

func TestNoConstraints(t *testing.T) {
	f := promotions.NewPromotionForm(service{})
	formstest.Submit(t, bg, f.Form, "name=Gold&type=points&points.points=5")
	if got := formstest.Errors(f.Form); !slices.Equal(got, []string{"constraints: errors.forms.min_items"}) {
		t.Errorf("errors %q", got)
	}
}

func TestConditionsForTheClient(t *testing.T) {
	f := promotions.NewPromotionForm(service{})
	tpl, _ := f.Constraints.Control().Template()
	for c := range tpl.Children() {
		if c.Path() != "constraints[__i__].min_amount" {
			continue
		}
		w, _ := c.When()
		b, _ := json.Marshal(w)
		if string(b) != `{"typ":"c","k":"constraints[__i__].type","op":"eq","v":"monetary"}` {
			t.Errorf("template condition %s", b)
		}
	}
}
