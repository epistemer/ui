package forms_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

// vector is one file of testdata/cond: fields by kind (text, int,
// checkbox), conditions on some of them, the condition under test in its
// wire form, the submitted values, and whether it holds. A client
// evaluator must pass the same files.
type vector struct {
	Fields map[string]string          `json:"fields"`
	When   map[string]json.RawMessage `json:"when"`
	Cond   json.RawMessage            `json:"cond"`
	Values map[string][]string        `json:"values"`
	Want   bool                       `json:"want"`
}

type wireNode struct {
	Typ  string     `json:"typ"`
	K    string     `json:"k"`
	Op   string     `json:"op"`
	V    string     `json:"v"`
	Args []wireNode `json:"args"`
}

// handles builds a Cond from a wire node through the fields' typed handles.
type handles struct {
	text  map[string]*forms.Field[string]
	ints  map[string]*forms.Field[int]
	boxes map[string]*forms.Field[bool]
}

func (h handles) cond(t *testing.T, n wireNode) forms.Cond {
	t.Helper()
	switch n.Op {
	case "and", "or", "not":
		var args []forms.Cond
		for _, a := range n.Args {
			args = append(args, h.cond(t, a))
		}
		switch n.Op {
		case "and":
			return forms.All(args...)
		case "or":
			return forms.Any(args...)
		}
		return forms.Not(args[0])
	case "eq":
		if f, ok := h.text[n.K]; ok {
			return f.Eq(n.V)
		}
		if f, ok := h.ints[n.K]; ok {
			v, err := strconv.Atoi(n.V)
			if err != nil {
				t.Fatal(err)
			}
			return f.Eq(v)
		}
		if n.V != "true" && n.V != "false" {
			t.Fatalf("a checkbox compares true or false, not %q", n.V)
		}
		return h.boxes[n.K].Eq(n.V == "true")
	case "filled":
		if f, ok := h.text[n.K]; ok {
			return f.Filled()
		}
		if f, ok := h.ints[n.K]; ok {
			return f.Filled()
		}
		return h.boxes[n.K].Filled()
	}
	t.Fatalf("unknown op %q", n.Op)
	return forms.Cond{}
}

func TestConformanceVectors(t *testing.T) {
	files, _ := filepath.Glob("testdata/cond/*.json")
	if len(files) == 0 {
		t.Fatal("no vectors")
	}
	for _, file := range files {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".json"), func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var v vector
			if err := json.Unmarshal(raw, &v); err != nil {
				t.Fatal(err)
			}
			f := forms.New()
			h := handles{map[string]*forms.Field[string]{}, map[string]*forms.Field[int]{}, map[string]*forms.Field[bool]{}}
			names := make([]string, 0, len(v.Fields))
			for name := range v.Fields {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				switch v.Fields[name] {
				case "text":
					h.text[name] = f.Text(name)
				case "int":
					h.ints[name] = f.Int(name)
				case "checkbox":
					h.boxes[name] = f.Checkbox(name)
				default:
					t.Fatalf("field %q: unknown kind %q", name, v.Fields[name])
				}
			}
			for name, w := range v.When {
				var n wireNode
				if err := json.Unmarshal(w, &n); err != nil {
					t.Fatal(err)
				}
				c := h.cond(t, n)
				switch {
				case h.text[name] != nil:
					h.text[name].When(c)
				case h.ints[name] != nil:
					h.ints[name].When(c)
				case h.boxes[name] != nil:
					h.boxes[name].When(c)
				default:
					t.Fatalf("when on unknown field %q", name)
				}
			}
			var n wireNode
			if err := json.Unmarshal(v.Cond, &n); err != nil {
				t.Fatal(err)
			}
			c := h.cond(t, n)
			assertWire(t, c, v.Cond)
			probe := f.Text("probe").When(c)
			values := map[string][]string{"probe": {"x"}}
			for k, vs := range v.Values {
				values[k] = vs
			}
			f.Bind(forms.FromValues(values))
			f.Validate(bg)
			if got := probe.Value() == "x"; got != v.Want {
				t.Errorf("holds = %v, want %v", got, v.Want)
			}
		})
	}
}

func assertWire(t *testing.T, c forms.Cond, want json.RawMessage) {
	t.Helper()
	got, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var g, w any
	json.Unmarshal(got, &g)
	json.Unmarshal(want, &w)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("wire\ngot  %s\nwant %s", got, want)
	}
}

func TestWhenRequired(t *testing.T) {
	newForm := func() (*forms.Form, *forms.Field[color], *forms.Field[int]) {
		f := forms.New()
		typ := f.Enum("type", []color{"red", "blue"}).Required()
		amount := f.Int("amount").When(typ.Eq("red")).Required().Is(rule.Min(1))
		return f, typ, amount
	}
	f, _, amount := newForm()
	bind(f, "type=blue&amount=-5")
	if !f.Validate(bg) {
		t.Errorf("an inactive required field failed the form: %v", f.Err())
	}
	if _, ok := amount.Get(); ok || amount.Err() != nil {
		t.Errorf("a stale value under a false When must be cleared: %v %v", amount.Value(), amount.Err())
	}
	g, _, amount2 := newForm()
	bind(g, "type=red")
	g.Validate(bg)
	if !errors.Is(amount2.Err(), forms.ErrRequired) {
		t.Errorf("an active required field: %v", amount2.Err())
	}
}

func TestEqAfterAsFormatsThroughStages(t *testing.T) {
	f := forms.New()
	c := f.Int("c").As(seconds)
	wire, _ := json.Marshal(c.Eq(90_000_000_000))
	if !strings.Contains(string(wire), `"v":"90"`) {
		t.Errorf("Eq on a converted field must compare the base text: %s", wire)
	}
}

func TestConditionPanics(t *testing.T) {
	for name, fn := range map[string]func(){
		"cycle": func() {
			f := forms.New()
			a, b := f.Text("a"), f.Text("b")
			b.When(a.Filled())
			a.When(b.Filled())
		},
		"self":        func() { f := forms.New(); a := f.Text("a"); a.When(a.Filled()) },
		"When twice":  func() { f := forms.New(); a, b := f.Text("a"), f.Text("b"); b.When(a.Filled()); b.When(a.Filled()) },
		"zero Cond":   func() { forms.New().Text("a").When(forms.Cond{}) },
		"zero in All": func() { forms.All(forms.Cond{}) },
		"Eq on Texts": func() { forms.New().Texts("a").Eq(nil) },
		"In on Files": func() { forms.New().Files("a").In() },
		"other form":  func() { a := forms.New().Text("a"); forms.New().Text("b").When(a.Filled()) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
	if _, err := json.Marshal(forms.Cond{}); err == nil {
		t.Error("MarshalJSON of a zero Cond must fail")
	}
}

func TestInactiveClearedBeforeAnyRule(t *testing.T) {
	f := forms.New()
	gate := f.Checkbox("gate")
	var seen string
	f.Text("first").Is(rule.New(func(context.Context, string) error { seen = laterValue(); return nil }))
	later := f.Text("later").When(gate.Filled())
	laterValue = later.Value
	bind(f, "first=x&later=stale")
	f.Validate(bg)
	if seen != "" {
		t.Errorf("a rule read %q from a later inactive field", seen)
	}
}

var laterValue func() string

func TestCheckboxConditions(t *testing.T) {
	for _, tc := range []struct {
		q                       string
		eqTrue, eqFalse, filled bool
	}{
		{"box=on", true, false, true},
		{"box=true", true, false, true},
		{"box=false&box=on", true, false, true},
		{"", false, true, false},
		{"box=false", false, true, false},
	} {
		f := forms.New()
		box := f.Checkbox("box")
		a := f.Text("a").When(box.Eq(true))
		b := f.Text("b").When(box.Eq(false))
		c := f.Text("c").When(box.Filled())
		bind(f, tc.q+"&a=1&b=1&c=1")
		f.Validate(bg)
		if (a.Value() == "1") != tc.eqTrue || (b.Value() == "1") != tc.eqFalse || (c.Value() == "1") != tc.filled {
			t.Errorf("%q: Eq(true) %v, Eq(false) %v, Filled %v", tc.q, a.Value() == "1", b.Value() == "1", c.Value() == "1")
		}
	}
}

func TestLongChainsStayFast(t *testing.T) {
	f := forms.New()
	prev := f.Text("f0")
	start := time.Now()
	for i := 1; i <= 40; i++ {
		next := f.Text("f" + strconv.Itoa(i))
		next.When(forms.All(prev.In("a", "b", "c", "d"), prev.Filled()))
		prev = next
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("declaring a 40-field chain took %v", d)
	}
}

func TestConditionOnConvertedField(t *testing.T) {
	f := forms.New()
	raw := f.Int("secs")
	before := raw.Eq(90) // built before As: stays valid
	c := raw.As(seconds)
	after := c.Eq(90 * time.Second)
	x := f.Text("x").When(before)
	y := f.Text("y").When(after)
	bind(f, "secs=90&x=1&y=1")
	f.Validate(bg)
	if x.Value() != "1" || y.Value() != "1" {
		t.Errorf("before As %q, after As %q", x.Value(), y.Value())
	}
}

func TestInactiveConversionErrorDoesNotFail(t *testing.T) {
	f := forms.New()
	gate := f.Text("gate")
	n := f.Int("n").When(gate.Filled())
	bind(f, "n=abc")
	if !f.Validate(bg) || n.Err() != nil {
		t.Errorf("an inactive field's conversion error failed the form: %v", f.Err())
	}
}

func TestCyclesInAnyOrder(t *testing.T) {
	f := forms.New()
	a, b, c := f.Text("a"), f.Text("b"), f.Text("c")
	a.When(forms.Not(b.Filled()))
	c.When(forms.Any(a.In("x")))
	defer func() {
		if recover() == nil {
			t.Error("closing a -> b -> c -> a in the middle did not panic")
		}
	}()
	b.When(c.Filled())
}

func TestMoreConditionPanics(t *testing.T) {
	bound := forms.New()
	h := bound.Text("x")
	g := bound.Text("g")
	bound.Bind(forms.FromValues(nil))
	raw := forms.New().Int("r")
	raw.As(seconds)
	for name, fn := range map[string]func(){
		"When after Bind":   func() { h.When(g.Filled()) },
		"Not zero":          func() { forms.Not(forms.Cond{}) },
		"Any zero":          func() { forms.Any(forms.Cond{}) },
		"Eq on File":        func() { forms.New().File("f").Eq(nil) },
		"In on Texts":       func() { forms.New().Texts("t").In() },
		"Eq on dead handle": func() { raw.Eq(1) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

func TestWireEdges(t *testing.T) {
	f := forms.New()
	a := f.Text("a")
	assertWire(t, a.Eq(""), json.RawMessage(`{"typ":"c","k":"a","op":"eq","v":""}`))
	assertWire(t, a.In(), json.RawMessage(`{"typ":"l","op":"or","args":[]}`))
	assertWire(t, a.Eq("  padded "), json.RawMessage(`{"typ":"c","k":"a","op":"eq","v":"padded"}`))
}
