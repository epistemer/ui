package jstest

import (
	"html"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

var htmlAttrRe = regexp.MustCompile(`([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+)))?`)

type fakeLayoutEnv struct {
	vm        *goja.Runtime
	attrs     map[string]string
	listeners map[string][]goja.Callable
	parsed    []string
}

func newLayoutEnv(t *testing.T, initialPath string, initialAttrs map[string]string) *fakeLayoutEnv {
	t.Helper()
	vm := goja.New()
	env := &fakeLayoutEnv{
		vm:        vm,
		attrs:     map[string]string{},
		listeners: map[string][]goja.Callable{},
	}
	for k, v := range initialAttrs {
		env.attrs[strings.ToLower(k)] = v
	}

	root := vm.NewObject()
	root.Set("getAttribute", func(call goja.FunctionCall) goja.Value {
		k := strings.ToLower(call.Argument(0).String())
		if v, ok := env.attrs[k]; ok {
			return vm.ToValue(v)
		}
		return goja.Null()
	})
	root.Set("setAttribute", func(call goja.FunctionCall) goja.Value {
		k := strings.ToLower(call.Argument(0).String())
		env.attrs[k] = call.Argument(1).String()
		return goja.Undefined()
	})
	root.Set("removeAttribute", func(call goja.FunctionCall) goja.Value {
		delete(env.attrs, strings.ToLower(call.Argument(0).String()))
		return goja.Undefined()
	})
	root.Set("dataset", map[string]any{"colorScheme": "dark"})

	addListener := func(call goja.FunctionCall) goja.Value {
		typ := call.Argument(0).String()
		fn, ok := goja.AssertFunction(call.Argument(1))
		if ok {
			env.listeners[typ] = append(env.listeners[typ], fn)
		}
		return goja.Undefined()
	}

	doc := vm.NewObject()
	doc.Set("documentElement", root)
	doc.Set("addEventListener", addListener)
	doc.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		return vm.NewArray()
	})

	win := vm.NewObject()
	win.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	loc := vm.NewObject()
	pathname, search, _ := strings.Cut(initialPath, "?")
	loc.Set("pathname", pathname)
	if search != "" {
		loc.Set("search", "?"+search)
	} else {
		loc.Set("search", "")
	}

	vm.Set("document", doc)
	vm.Set("window", win)
	vm.Set("location", loc)
	vm.Set("matchMedia", func(call goja.FunctionCall) goja.Value {
		m := vm.NewObject()
		m.Set("addEventListener", func(goja.FunctionCall) goja.Value { return goja.Undefined() })
		return m
	})
	vm.Set("DOMParser", func(call goja.ConstructorCall) *goja.Object {
		parser := call.This
		parser.Set("parseFromString", func(call goja.FunctionCall) goja.Value {
			input := call.Argument(0).String()
			env.parsed = append(env.parsed, input)
			parsedAttrs := parseFakeHTMLStartTag(input)
			docEl := vm.NewObject()
			docEl.Set("getAttribute", func(call goja.FunctionCall) goja.Value {
				k := strings.ToLower(call.Argument(0).String())
				if v, ok := parsedAttrs[k]; ok {
					return vm.ToValue(v)
				}
				return goja.Null()
			})
			d := vm.NewObject()
			d.Set("documentElement", docEl)
			return d
		})
		return nil
	})

	src, err := os.ReadFile("../../theme/js/layout.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vm.RunScript("../../theme/js/layout.js", exportKeyword.ReplaceAllString(string(src), "")); err != nil {
		t.Fatalf("layout.js: %v", err)
	}
	return env
}

func parseFakeHTMLStartTag(s string) map[string]string {
	out := map[string]string{}
	start := strings.Index(strings.ToLower(s), "<html")
	if start < 0 {
		return out
	}
	rest := s[start+len("<html"):]
	end := strings.IndexByte(rest, '>')
	if end < 0 {
		return out
	}
	for _, m := range htmlAttrRe.FindAllStringSubmatch(rest[:end], -1) {
		k := strings.ToLower(m[1])
		v := m[2]
		if v == "" && m[3] != "" {
			v = m[3]
		} else if v == "" && m[4] != "" {
			v = m[4]
		}
		out[k] = html.UnescapeString(v)
	}
	return out
}

func (e *fakeLayoutEnv) dispatch(t *testing.T, typ string, detail map[string]any) {
	t.Helper()
	ev := e.vm.NewObject()
	ev.Set("detail", detail)
	for _, fn := range e.listeners[typ] {
		if _, err := fn(goja.Undefined(), ev); err != nil {
			t.Fatalf("dispatch %s: %v", typ, err)
		}
	}
}

func (e *fakeLayoutEnv) assertAttrs(t *testing.T, want map[string]string) {
	t.Helper()
	for k, w := range want {
		got, ok := e.attrs[k]
		if w == "" {
			if ok {
				t.Errorf("attr %q = %q, want removed", k, got)
			}
		} else if !ok || got != w {
			t.Errorf("attr %q = %q (present=%v), want %q", k, got, ok, w)
		}
	}
}

func TestLayoutBoostedHTMLAttrs(t *testing.T) {
	env := newLayoutEnv(t, "/en", map[string]string{
		"lang":  "en",
		"dir":   "ltr",
		"class": "app",
	})

	// Non-boosted swap must not touch lang/dir.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": false,
		"xhr":     map[string]any{"responseText": `<!doctype html><html lang="ar" dir="rtl"><body></body></html>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr", "class": "app"})

	// Boosted swap with a fragment (no <html> start tag) leaves lang/dir alone.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"xhr":     map[string]any{"responseText": `<main>fragment</main>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr", "class": "app"})

	// Boosted swap with full page updates only lang and dir, and parses only the start tag.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"xhr": map[string]any{
			"responseText": `<!doctype html><html lang="ar" dir="rtl" class="other" data-extra="1"><body><script>boom()</script></body></html>`,
		},
	})
	env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl", "class": "app", "data-extra": ""})
	if len(env.parsed) != 1 || strings.Contains(env.parsed[0], "<script>") || !strings.HasSuffix(env.parsed[0], "</html>") {
		t.Errorf("DOMParser received %q, want only <html> start tag + </html>", env.parsed)
	}

	// Missing attribute on the response removes it from documentElement.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"xhr":     map[string]any{"responseText": `<!doctype html><html lang="fr"><body></body></html>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "fr", "dir": "", "class": "app"})

	// Bare <html> removes both lang and dir.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"xhr":     map[string]any{"responseText": `<html><body></body></html>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "", "dir": "", "class": "app"})

	// htmx:historyCacheMissLoad updates lang/dir from xhr, and the following
	// cacheMiss historyRestore does not overwrite it.
	env.dispatch(t, "htmx:historyCacheMissLoad", map[string]any{
		"path": "/ar-eg",
		"xhr":  map[string]any{"responseText": `<!doctype html><html lang="ar-EG" dir="rtl"><body></body></html>`},
	})
	env.dispatch(t, "htmx:historyRestore", map[string]any{
		"path":      "/ar-eg",
		"cacheMiss": true,
	})
	env.assertAttrs(t, map[string]string{"lang": "ar-EG", "dir": "rtl", "class": "app"})
}

func TestLayoutHistoryCacheRestore(t *testing.T) {
	t.Run("pushed_and_replaced_after_swap", func(t *testing.T) {
		env := newLayoutEnv(t, "/en", map[string]string{"lang": "en", "dir": "ltr"})

		env.dispatch(t, "htmx:afterSwap", map[string]any{
			"boosted": true,
			"xhr":     map[string]any{"responseText": `<html lang="ar" dir="rtl"><body></body></html>`},
		})
		env.dispatch(t, "htmx:pushedIntoHistory", map[string]any{"path": "/ar"})

		env.dispatch(t, "htmx:afterSwap", map[string]any{
			"boosted": true,
			"xhr":     map[string]any{"responseText": `<html lang="fr"><body></body></html>`},
		})
		env.dispatch(t, "htmx:replacedInHistory", map[string]any{"path": "/fr"})

		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/ar"})
		env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl"})

		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/fr"})
		env.assertAttrs(t, map[string]string{"lang": "fr", "dir": ""})

		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/en"})
		env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})

		// Unknown path changes nothing.
		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/never-seen"})
		env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})
	})

	t.Run("htmx2_order_push_before_swap", func(t *testing.T) {
		env := newLayoutEnv(t, "/en", map[string]string{"lang": "en", "dir": "ltr"})

		// htmx 2 fires beforeHistorySave, beforeHistoryUpdate, pushedIntoHistory, then afterSwap.
		env.dispatch(t, "htmx:beforeHistorySave", map[string]any{"path": "/en"})
		env.dispatch(t, "htmx:beforeHistoryUpdate", map[string]any{"history": map[string]any{"path": "/ar"}})
		env.dispatch(t, "htmx:pushedIntoHistory", map[string]any{"path": "/ar"})
		env.dispatch(t, "htmx:afterSwap", map[string]any{
			"boosted": true,
			"xhr":     map[string]any{"responseText": `<!doctype html><html lang="ar" dir="rtl"><body></body></html>`},
		})
		env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl"})

		// Back to /en, then forward to /ar from cache.
		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/en"})
		env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})

		env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/ar"})
		env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl"})
	})
}
