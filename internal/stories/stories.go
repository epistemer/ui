// Package stories holds the D4 spike's stories, one file per package;
// the catalogue serves them and the tests check them.
package stories

import (
	"context"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/catalog"
	"github.com/epistemer/ui/elements"
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
		{Name: "icons/all", Variants: []catalog.Variant{variant("set", allIcons())}},
		{Name: "elements/badge", Variants: []catalog.Variant{variant("tones", badges())}},
		{Name: "elements/alert", Variants: []catalog.Variant{variant("levels", alerts())}},
		{Name: "elements/card", Variants: []catalog.Variant{variant("regions", cards())}},
		{Name: "elements/avatar", Variants: []catalog.Variant{variant("kinds", avatars())}},
		{Name: "elements/spinner", Variants: []catalog.Variant{variant("sizes", spinners())}},
		{Name: "elements/tooltip", Variants: []catalog.Variant{variant("icon-button", tooltips())}},
		{Name: "elements/separator", Variants: []catalog.Variant{variant("both", separators())}},
		{Name: "elements/heading", Variants: []catalog.Variant{variant("sizes", headings())}},
		{Name: "elements/values", Variants: []catalog.Variant{variant("kinds", valueList()), variant("empty", valueEmpties())}},
		{Name: "inputs/field", Variants: []catalog.Variant{
			variant("fresh", promotionForm(promotionFresh())),
			variant("filled", promotionForm(promotionFilled())),
			variant("errors", promotionForm(promotionErrors())),
			variant("kinds", promotionForm(shippingFresh())),
			variant("kinds-gift", promotionForm(shippingGift())),
			variant("sign-in", promotionForm(signInFresh())),
			variant("sign-in-errors", promotionForm(signInErrors())),
		}},
		{Name: "components/dialog", Variants: []catalog.Variant{
			variant("basic", dialogBasic()),
			variant("custom-trigger", dialogCustomTrigger()),
			variant("confirm", confirmForm()),
			variant("confirm-htmx", confirmHtmx()),
		}},
		{Name: "components/menu", Variants: []catalog.Variant{
			variant("basic", menuBasic()),
			variant("icon-trigger", menuIcon()),
			variant("in-a-table", menuInTable()),
		}},
		{Name: "components/tabs", Variants: []catalog.Variant{variant("navigation", tabsStory())}},
		{Name: "components/toast", Variants: []catalog.Variant{variant("levels", toastsStory())}},
		{Name: "components/detail-list", Variants: []catalog.Variant{variant("in-a-card", detailStory())}},
	}
}
