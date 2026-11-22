package stories

import (
	"context"
	"io"
	"net/http/httptest"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/inputs"
	"github.com/epistemer/ui/internal/toyshop"
)

var shopStore = toyshop.NewStore()

// ordersTable renders the toy shop's orders for a query string, as its
// handler would.
func ordersTable(query string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		r := httptest.NewRequest("GET", toyshop.Prefix+"/orders?"+query, nil)
		ctx = inputs.WithLabelCache(actions.Gate.With(ctx, toyshop.Gate))
		st := toyshop.Orders.Bind(r.WithContext(ctx))
		v := st.Rejected()
		if st.Valid() {
			rows, total := shopStore.ListOrderListView(st)
			v = st.Page(rows, total)
		}
		if err := toyshop.OrdersView.Page(v).Render(ctx, w); err != nil {
			return err
		}
		return shellParts().Render(ctx, w)
	})
}

// pager renders the orders' pager alone, without its table.
func pager(query string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		r := httptest.NewRequest("GET", toyshop.Prefix+"/orders?"+query, nil)
		ctx = actions.Gate.With(ctx, toyshop.Gate)
		st := toyshop.Orders.Bind(r.WithContext(ctx))
		rows, total := shopStore.ListOrderListView(st)
		p := st.Page(rows, total).Pager
		p.Target = "" // alone on its page: plain links
		return components.Pagination(p).Render(ctx, w)
	})
}
