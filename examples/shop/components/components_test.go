package components_test

import (
	"testing"

	"github.com/epistemer/ui/examples/shop/components"
	"github.com/epistemer/ui/uitest"
)

func TestCartModal(t *testing.T) {
	html := uitest.Render(t, components.CartModalPage())
	uitest.CheckIDs(t, html)
	uitest.Golden(t, "cart_modal", html)
}

func TestDropdown(t *testing.T) {
	html := uitest.Render(t, components.DropdownPage())
	uitest.CheckIDs(t, html)
	uitest.Golden(t, "dropdown", html)
}

func TestOrderTable(t *testing.T) {
	orders := []components.Order{
		{ID: "1042", Customer: "Alice", Total: "$42.00"},
		{ID: "1043", Customer: "Bob", Total: "$15.50"},
	}
	html := uitest.Render(t, components.OrderTablePage(orders))
	uitest.CheckIDs(t, html)
	uitest.Golden(t, "order_table", html)

	// Also verify OrderTable rendering rows directly.
	tableHTML := uitest.Render(t, components.OrderTable(orders))
	uitest.CheckIDs(t, tableHTML)
}

func TestProductCard(t *testing.T) {
	p1 := components.Product{ID: "p-1", Name: "Loose Leaf Tea", Price: "$12.00"}
	p2 := components.Product{ID: "p-2", Name: "Coffee Mug", Price: "$8.50"}
	html := uitest.Render(t, components.ProductCardPage(p1, p2))
	uitest.CheckIDs(t, html)
	uitest.Golden(t, "product_card", html)
}
