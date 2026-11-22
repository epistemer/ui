package catalogtest

import (
	"context"
	"slices"
	"testing"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/catalog"
)

// Check passes on a story with an app class its CSS selects, an external
// id, and ui's own button.
func TestCheck(t *testing.T) {
	story := catalog.Story{Name: "shop/card", Variants: []catalog.Variant{{
		Name: "plain",
		Render: func(context.Context) templ.Component {
			return templ.Raw(`<div class="shop-card"><a href="#page">Top</a><button class="ui-button" type="button">Buy</button></div>`)
		},
	}}}
	Check(t, []catalog.Story{story}, CSS(`.shop-card { color: inherit; }`), Prefix("shop-"), External("page"))
}

func TestUnused(t *testing.T) {
	css := `.shop-card { color: red; } .shop-badge { color: red; } .ui-x { color: red; }`
	got := unused(css, map[string]bool{".shop-card": true}, []string{"ui-", "shop-"})
	if !slices.Equal(got, []string{".shop-badge", ".ui-x"}) {
		t.Errorf("unused: %q", got)
	}
}
