// Package tablestest tests tables without HTML (server-state-interface.md
// §3.6).
//
//	st := tablestest.Bind(t, Orders, "q=ann&status=paid&sort=-placed_at&page=2")
//	tablestest.Query(st)  // the listing.Query, to compare
//	tablestest.Errors(st) // ["placed_at: errors.tables.range"]
//	tablestest.Check(t, Orders)
package tablestest

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui/actions/actionstest"
	"github.com/epistemer/ui/tables"
)

// Bind binds a query string with a gate that admits every link but what
// actionstest.Refuse names.
func Bind[R any](tb testing.TB, t *tables.Table[R], query string, opts ...actionstest.Option) *tables.State[R] {
	tb.Helper()
	r := httptest.NewRequest("GET", "/?"+query, nil)
	return t.Bind(r.WithContext(actionstest.Context(tb, opts...)))
}

// Query is the state's listing query.
func Query[R any](st *tables.State[R]) listing.Query { return st.ListingQuery() }

// Errors lists the filters' errors as "path: key".
func Errors[R any](st *tables.State[R]) []string {
	var out []string
	for e := range st.Errors() {
		key := e.Err.Error()
		if k, ok := e.Err.(interface{ K() string }); ok {
			key = k.K()
		}
		out = append(out, fmt.Sprintf("%s: %s", e.Path, key))
	}
	return out
}

// Check fails the test for what the table's declaration gets wrong.
func Check[R any](tb testing.TB, t *tables.Table[R]) {
	tb.Helper()
	if err := t.Check(); err != nil {
		tb.Error(err)
	}
}
