// Package tablestest is the spike of ui/tables/tablestest (§6.9).
package tablestest

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui/internal/spike/actions"
	"github.com/epistemer/ui/internal/spike/tables"
)

// Bind binds t from a query string, with a gate that admits everything.
func Bind[R any](tb testing.TB, t *tables.Table[R], query string) *tables.State[R] {
	tb.Helper()
	r := httptest.NewRequest("GET", "/?"+query, nil)
	r = r.WithContext(actions.Gate.With(r.Context(), actions.AllowAll))
	return t.Bind(r)
}

// Query is the state's listing query.
func Query[R any](st *tables.State[R]) listing.Query { return st.ListingQuery() }

// Errors lists the filters' errors as "path: key".
func Errors[R any](st *tables.State[R]) []string {
	var out []string
	for e := range st.Errors() {
		out = append(out, fmt.Sprintf("%s: %s", e.Path, key(e.Err)))
	}
	return out
}

func key(err error) string {
	if k, ok := err.(interface{ K() string }); ok {
		return k.K()
	}
	return err.Error()
}

// Check fails the test for what the table's declaration gets wrong.
func Check[R any](tb testing.TB, t *tables.Table[R]) {
	tb.Helper()
	if err := t.Check(); err != nil {
		tb.Error(err)
	}
}
