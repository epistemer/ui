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
	body      *goja.Object
	loc       *goja.Object
	storage   map[string]string
}

func newLayoutEnv(t *testing.T, initialPath string, initialAttrs map[string]string) *fakeLayoutEnv {
	t.Helper()
	return newLayoutEnvStorage(t, initialPath, initialAttrs, map[string]string{})
}

// newLayoutEnvStorage starts a page whose sessionStorage holds storage, as
// a reload in the same tab finds it.
func newLayoutEnvStorage(t *testing.T, initialPath string, initialAttrs, storage map[string]string) *fakeLayoutEnv {
	t.Helper()
	vm := goja.New()
	env := &fakeLayoutEnv{
		vm:        vm,
		attrs:     map[string]string{},
		listeners: map[string][]goja.Callable{},
		storage:   storage,
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
	env.body = vm.NewObject()
	doc.Set("body", env.body)
	doc.Set("documentElement", root)
	doc.Set("addEventListener", addListener)
	doc.Set("querySelectorAll", func(call goja.FunctionCall) goja.Value {
		return vm.NewArray()
	})

	win := vm.NewObject()
	win.Set("addEventListener", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })

	env.loc = vm.NewObject()
	env.goTo(initialPath)
	ss := vm.NewObject()
	ss.Set("getItem", func(call goja.FunctionCall) goja.Value {
		if v, ok := env.storage[call.Argument(0).String()]; ok {
			return vm.ToValue(v)
		}
		return goja.Null()
	})
	ss.Set("setItem", func(call goja.FunctionCall) goja.Value {
		env.storage[call.Argument(0).String()] = call.Argument(1).String()
		return goja.Undefined()
	})
	vm.Set("sessionStorage", ss)
	// URL, as far as layout.js uses it: a path, query and fragment,
	// absolute or relative to this origin.
	if _, err := vm.RunString(`globalThis.URL = function (u) {
		u = String(u).replace(/^https?:\/\/[^/]+/, "");
		const h = u.indexOf("#"); if (h >= 0) u = u.slice(0, h);
		const q = u.indexOf("?");
		this.pathname = q >= 0 ? u.slice(0, q) : u;
		this.search = q >= 0 ? u.slice(q) : "";
	};`); err != nil {
		t.Fatal(err)
	}

	vm.Set("document", doc)
	vm.Set("window", win)
	vm.Set("location", env.loc)
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

// goTo sets the address, as htmx's pushState does before it swaps.
func (e *fakeLayoutEnv) goTo(path string) {
	pathname, search, _ := strings.Cut(path, "?")
	e.loc.Set("pathname", pathname)
	if search != "" {
		e.loc.Set("search", "?"+search)
	} else {
		e.loc.Set("search", "")
	}
	e.loc.Set("href", "http://localhost"+path)
}

// boosted dispatches htmx 2's events for a boosted page to path: the
// cache saves the page, the address changes, then the body is swapped.
func (e *fakeLayoutEnv) boosted(t *testing.T, path, response string) {
	t.Helper()
	e.dispatch(t, "htmx:beforeHistorySave", map[string]any{"path": e.loc.Get("pathname").String() + e.loc.Get("search").String()})
	e.goTo(path)
	e.dispatch(t, "htmx:pushedIntoHistory", map[string]any{"path": path})
	e.dispatch(t, "htmx:afterSwap", map[string]any{"boosted": true, "target": e.body, "xhr": map[string]any{"responseText": response}})
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
		"target":  env.body,
		"xhr":     map[string]any{"responseText": `<main>fragment</main>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr", "class": "app"})

	// Boosted swap with full page updates only lang and dir, and parses only the start tag.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"target":  env.body,
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
		"target":  env.body,
		"xhr":     map[string]any{"responseText": `<!doctype html><html lang="fr"><body></body></html>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "fr", "dir": "", "class": "app"})

	// Bare <html> removes both lang and dir.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"target":  env.body,
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

func TestLayoutBoostedPartIgnored(t *testing.T) {
	env := newLayoutEnv(t, "/en", map[string]string{"lang": "en", "dir": "ltr"})
	// A boosted link with a target of its own swaps a part, not the page.
	env.dispatch(t, "htmx:afterSwap", map[string]any{
		"boosted": true,
		"target":  env.vm.NewObject(),
		"xhr":     map[string]any{"responseText": `<html lang="ar" dir="rtl"><body></body></html>`},
	})
	env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})
}

func TestLayoutHistoryCacheRestore(t *testing.T) {
	env := newLayoutEnv(t, "/en", map[string]string{"lang": "en", "dir": "ltr"})
	env.boosted(t, "/ar", `<!doctype html><html lang="ar" dir="rtl"><body></body></html>`)
	env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl"})
	env.boosted(t, "/fr?x=1", `<html lang="fr"><body></body></html>`)
	env.assertAttrs(t, map[string]string{"lang": "fr", "dir": ""})

	// Back and forward from htmx's cache, which restores only the body;
	// htmx reports the path without its fragment.
	// Back: location already names /ar when htmx saves /fr?x=1.
	env.goTo("/ar")
	env.dispatch(t, "htmx:beforeHistorySave", map[string]any{"path": "/fr?x=1"})
	env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/ar"})
	env.assertAttrs(t, map[string]string{"lang": "ar", "dir": "rtl"})
	env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/en"})
	env.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})
	env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/fr?x=1"})
	env.assertAttrs(t, map[string]string{"lang": "fr", "dir": ""})

	// An unknown path changes nothing.
	env.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/never-seen"})
	env.assertAttrs(t, map[string]string{"lang": "fr", "dir": ""})

	// A reload of /ar in the same tab, then Back to /en from htmx's cache,
	// which lives in sessionStorage too.
	env.goTo("/ar")
	reloaded := newLayoutEnvStorage(t, "/ar", map[string]string{"lang": "ar", "dir": "rtl"}, env.storage)
	reloaded.dispatch(t, "htmx:historyRestore", map[string]any{"path": "/en"})
	reloaded.assertAttrs(t, map[string]string{"lang": "en", "dir": "ltr"})
}
