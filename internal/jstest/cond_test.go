package jstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/dop251/goja"
)

// The modules are plain files in the ui module; this test reads them as
// text. goja doesn't load ES modules, so the test drops the export
// keywords, which changes nothing else.
var exportKeyword = regexp.MustCompile(`(?m)^export `)

func load(t *testing.T, path string) *goja.Runtime {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	vm := goja.New()
	if _, err := vm.RunScript(path, exportKeyword.ReplaceAllString(string(src), "")); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return vm
}

type vector struct {
	Fields map[string]string          `json:"fields"`
	When   map[string]json.RawMessage `json:"when"`
	Cond   json.RawMessage            `json:"cond"`
	Values map[string][]string        `json:"values"`
	Want   bool                       `json:"want"`
}

// TestConditionVectors runs the client evaluator against forms'
// conformance vectors: the same files forms' own Go evaluator passes
// (forms-design §7.3).
func TestConditionVectors(t *testing.T) {
	vm := load(t, "../spike/theme/js/cond.js")
	evaluate, ok := goja.AssertFunction(vm.Get("evaluate"))
	if !ok {
		t.Fatal("cond.js defines no evaluate")
	}
	files, err := filepath.Glob("../../forms/testdata/cond/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no vectors: %v", err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var v vector
			if err := json.Unmarshal(b, &v); err != nil {
				t.Fatal(err)
			}
			parse := func(raw json.RawMessage) goja.Value {
				var x any
				if err := json.Unmarshal(raw, &x); err != nil {
					t.Fatal(err)
				}
				return vm.ToValue(x)
			}
			env := vm.NewObject()
			env.Set("kind", func(name string) string { return v.Fields[name] })
			env.Set("values", func(name string) []string { return v.Values[name] })
			env.Set("when", func(name string) []any {
				if raw, ok := v.When[name]; ok {
					return []any{parse(raw).Export()}
				}
				return nil
			})
			got, err := evaluate(goja.Undefined(), parse(v.Cond), env)
			if err != nil {
				t.Fatal(err)
			}
			if got.ToBoolean() != v.Want {
				t.Errorf("got %v, want %v", got.ToBoolean(), v.Want)
			}
		})
	}
}

// TestConditionChains covers what the vectors' single "when" per field
// can't: a field whose activity is a chain (its own condition and its
// group's), as the ui script collects from nested wrappers.
func TestConditionChains(t *testing.T) {
	vm := load(t, "../spike/theme/js/cond.js")
	evaluate, _ := goja.AssertFunction(vm.Get("evaluate"))
	run := func(values map[string][]string) bool {
		t.Helper()
		var cond, groupCond, ownCond any
		json.Unmarshal([]byte(`{"typ":"c","k":"street","op":"filled"}`), &cond)
		json.Unmarshal([]byte(`{"typ":"c","k":"gift","op":"eq","v":"true"}`), &groupCond)
		json.Unmarshal([]byte(`{"typ":"c","k":"country","op":"filled"}`), &ownCond)
		env := vm.NewObject()
		env.Set("kind", func(name string) string { return map[string]string{"gift": "checkbox"}[name] })
		env.Set("values", func(name string) []string { return values[name] })
		env.Set("when", func(name string) []any {
			if name == "street" {
				return []any{ownCond, groupCond}
			}
			return nil
		})
		got, err := evaluate(goja.Undefined(), vm.ToValue(cond), env)
		if err != nil {
			t.Fatal(err)
		}
		return got.ToBoolean()
	}
	all := map[string][]string{"gift": {"true"}, "country": {"EG"}, "street": {"12 Corniche"}}
	if !run(all) {
		t.Error("street with both conditions holding reads empty")
	}
	for _, drop := range []string{"gift", "country"} {
		v := map[string][]string{}
		for k, x := range all {
			if k != drop {
				v[k] = x
			}
		}
		if run(v) {
			t.Errorf("street reads filled with %s's condition failing", drop)
		}
	}
}
