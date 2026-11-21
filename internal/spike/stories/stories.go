// Package stories holds the D4 spike's stories, one file per package;
// the catalogue serves them and the tests check them.
package stories

import (
	"context"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/internal/spike/catalog"
	"github.com/epistemer/ui/internal/spike/elements"
)

func variant(name string, c templ.Component) catalog.Variant {
	return catalog.Variant{Name: name, Render: func(context.Context) templ.Component { return c }}
}

// All returns every story.
func All() []catalog.Story {
	return []catalog.Story{
		{Name: "elements/button", Variants: []catalog.Variant{
			variant("solid", buttonTones(elements.Solid)),
			variant("outline", buttonTones(elements.Outline)),
			variant("ghost", buttonTones(elements.Ghost)),
			variant("sizes", buttonSizes()),
			variant("states", buttonStates()),
		}},
		{Name: "inputs/field", Variants: []catalog.Variant{
			variant("fresh", promotionForm(promotionFresh())),
			variant("filled", promotionForm(promotionFilled())),
			variant("errors", promotionForm(promotionErrors())),
			variant("kinds", promotionForm(shippingFresh())),
			variant("kinds-gift", promotionForm(shippingGift())),
		}},
		{Name: "components/dialog", Variants: []catalog.Variant{
			variant("basic", dialogBasic()),
			variant("custom-trigger", dialogCustomTrigger()),
			variant("confirm", confirmForm()),
		}},
		{Name: "components/menu", Variants: []catalog.Variant{
			variant("basic", menuBasic()),
			variant("in-a-table", menuInTable()),
		}},
	}
}
