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
