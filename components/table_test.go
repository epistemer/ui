package components

import "testing"

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
