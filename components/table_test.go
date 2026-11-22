package components

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/column"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/tables"
)

func TestMarked(t *testing.T) {
	join := func(ps []piece) (s string) {
		for _, p := range ps {
			if p.Hit {
				s += "[" + p.Text + "]"
			} else {
				s += p.Text
			}
		}
		return s
	}
	for _, c := range []struct{ text, q, want string }{
		{"Layla Adel", "la", "[La]y[la] Adel"},
		{"Layla", "", "Layla"},
		{"Kelvin K", "k", "Kelvin K"},             // the Kelvin sign lowers shorter: no marks
		{"bad \xff byte", "bad", "bad \xff byte"}, // invalid UTF-8: no marks
		{"abc", "K", "abc"},
	} {
		if got := join(marked(c.text, c.q)); got != c.want {
			t.Errorf("marked(%q, %q) = %q, want %q", c.text, c.q, got, c.want)
		}
	}
}

type errLabeler struct{ err error }

func (l errLabeler) LabelOf(context.Context, string) (string, bool, error) {
	return "", false, l.err
}

func TestTablePageChipError(t *testing.T) {
	type row struct{ ID int64 }
	idCol := column.Column[row]{
		Name:  "id",
		Label: i18n.Key("table.test_items.id", "ID"),
		Kind:  column.Integer,
		Get:   func(r row) column.Value { return column.Value{Int: r.ID} },
	}
	boom := errors.New("label store down")
	tb := tables.New("items", idCol,
		tables.Col(idCol).Filter().Remote(ui.URL("/items/search"), errLabeler{err: boom}),
	).Label(i18n.Static("Items"))
	tv := TableOf(tb)

	ctx := actions.Gate.With(context.Background(), actions.AllowAll)
	r := httptest.NewRequest("GET", "/items?id=5", nil).WithContext(ctx)
	v := tb.Bind(r).Page(nil, 0)
	var buf bytes.Buffer
	err := tv.Page(v).Render(ctx, &buf)
	if !errors.Is(err, boom) {
		t.Fatalf("Page().Render error = %v, want %v", err, boom)
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %d bytes on error", buf.Len())
	}
}
