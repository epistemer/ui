package forms_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

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
				}
			}
			for name, w := range v.When {
				var n wireNode
				json.Unmarshal(w, &n)
				h.text[name].When(h.cond(t, n))
			}
			var n wireNode
			json.Unmarshal(v.Cond, &n)
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
