package toyshop

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/icons"
	"github.com/epistemer/ui/tables"
)

// route stands in for handlers' generated link: a URL and a method.
type route struct{ url, method string }

func (r route) URL() string    { return r.url }
func (r route) Method() string { return r.method }

// Prefix is where cmd/catalog mounts the shop.
const Prefix = "/shop"

func get(f string, a ...any) route  { return route{Prefix + fmt.Sprintf(f, a...), "GET"} }
func post(f string, a ...any) route { return route{Prefix + fmt.Sprintf(f, a...), "POST"} }

// Routes, as handlers would generate their builders.
func OrderRoute(id int64) route        { return get("/orders/%d", id) }
func OrderRefund(id int64) route       { return post("/orders/%d/refund", id) }
func OrderReassign(id int64) route     { return get("/orders/%d/reassign", id) }
func OrderReassignPost(id int64) route { return post("/orders/%d/reassign", id) }
func OrderDelete(id int64) route       { return post("/orders/%d/delete", id) }
func OrderArchive(id int64) route      { return post("/orders/%d/archive", id) }
func OrdersShip() route                { return post("/orders/ship") }
func OrdersTag() route                 { return get("/orders/tag") }
func OrdersTagPost() route             { return post("/orders/tag") }
func CustomerRoute(id int64) route     { return get("/customers/%d", id) }
func OrdersRoute() route               { return get("/orders") }

// Gate stands in for a guard: it refuses archiving, as a guard without
// the permission would.
func Gate(l ui.Link) bool { return !strings.HasSuffix(l.URL(), "/archive") }

// ErrNotPaid is the state rule the refund handler runs too.
var ErrNotPaid = errors.Key("errors.shop.not_paid", "only a paid order can be refunded", errors.Conflict)

// CanRefund is the rule behind Refund, for the button and the handler.
func CanRefund(o OrderListView) error {
	if o.Status != Paid {
		return ErrNotPaid
	}
	return nil
}

var (
	View     = actions.New(i18n.Key("page.orders.view", "View")).Icon(icons.Eye())
	Refund   = actions.New(i18n.Key("page.orders.refund", "Refund")).Danger().Confirm(i18n.Key("page.orders.refund_question", "Refund this order?"))
	Reassign = actions.New(i18n.Key("page.orders.reassign", "Reassign")).Dialog()
	Delete   = actions.New(i18n.Key("page.orders.delete", "Delete")).Danger().Confirm(i18n.Key("page.orders.delete_question", "Delete this order?"))
	Archive  = actions.New(i18n.Key("page.orders.archive", "Archive"))
	Ship     = actions.New(i18n.Key("page.orders.ship", "Mark shipped")).Primary().Confirm(i18n.Key("page.orders.ship_question", "Mark the selected orders shipped?"))
	Tag      = actions.New(i18n.Key("page.orders.tag", "Tag")).Dialog()
)

func orderActions(ctx context.Context, o OrderListView) []actions.Item {
	return actions.Resolve(ctx,
		View.To(OrderRoute(o.ID)),
		Reassign.To(OrderReassign(o.ID)),
		Archive.To(OrderArchive(o.ID)),
		Refund.To(OrderRefund(o.ID)).Unless(CanRefund(o)),
		Delete.To(OrderDelete(o.ID)).If(o.Status != Shipped),
	)
}

func bulkOrderActions(ctx context.Context) []actions.Item {
	return actions.Resolve(ctx, Ship.To(OrdersShip()), Tag.To(OrdersTag()))
}

func money(o OrderListView) i18n.Ter {
	return i18n.Static(fmt.Sprintf("EGP %d.%02d", o.Total/100, o.Total%100))
}

var Orders = tables.New("orders", oc.ID,
	tables.Col(oc.Email).Searchable().Sortable(),
	tables.Col(oc.Customer).Searchable().Link(func(o OrderListView) ui.Link { return CustomerRoute(o.CustomerID) }),
	tables.Col(oc.Status).Filter(),
	tables.Col(oc.Total).Sortable().Amount(money),
	tables.Col(oc.PlacedAt).Sortable().Filter(),
	tables.Col(oc.Assignee),
).
	Label(i18n.Key("page.orders.title", "Orders")).
	RowLabel(oc.Email).
	Sort(oc.PlacedAt, tables.Desc).
	Sizes(10, 25, 50).
	Actions(orderActions).
	Bulk(bulkOrderActions)

var Toys = tables.New("toys", tc.ID,
	tables.Col(tc.Name).Searchable().Sortable(),
	tables.Col(tc.Stock).Sortable().Filter(),
	tables.Col(tc.Electric).Filter(),
).
	Label(i18n.Key("page.toys.title", "Toys")).
	Sizes(5, 10).
	Prefixed()

// Toned badges for the status, through Cell.
func statusCell(o OrderListView) templ.Component {
	tone := map[OrderStatus]elements.Tone{Paid: elements.Success, Pending: elements.Warning, Refunded: elements.Neutral, Shipped: elements.Info}[o.Status]
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return elements.Badge().Tone(tone).Render(templ.WithChildren(ctx, templ.Raw(templ.EscapeString(o.Status.Label().T(ctx)))), w)
	})
}

var (
	OrdersView = components.TableOf(Orders).Cell(oc.Status, statusCell)
	ToysView   = components.TableOf(Toys)
)

// orNothing is c, or nothing for nil.
func orNothing(c templ.Component) templ.Component {
	if c == nil {
		return templ.NopComponent
	}
	return c
}
