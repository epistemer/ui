package catalog_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/catalog"
	"github.com/epistemer/ui/theme"
)

func newCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()
	th, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	hello := templ.Raw(`<p>hello</p>`)
	return catalog.New(th, catalog.Story{Name: "x/hello", Variants: []catalog.Variant{
		{Name: "plain", Render: func(context.Context) templ.Component { return hello }},
		{Name: "with & space", Render: func(context.Context) templ.Component { return hello }},
	}})
}

func TestPages(t *testing.T) {
	c := newCatalog(t)
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	if w := get("/"); w.Code != 302 || w.Header().Get("Location") != "story?name=x%2Fhello" {
		t.Errorf("index: %d %q", w.Code, w.Header().Get("Location"))
	}
	w := get("/story?name=x/hello")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `aria-current="page"`) || strings.Count(w.Body.String(), "<iframe") != 8 ||
		!strings.Contains(w.Body.String(), `variant=with+%26+space`) || !strings.Contains(w.Body.String(), `id="variant-1"`) {
		t.Errorf("story page: %d\n%s", w.Code, w.Body.String())
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "style-src 'self'") {
		t.Errorf("CSP %q", csp)
	}
	w = get("/frame?story=x/hello&variant=plain&dir=rtl&scheme=dark")
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `dir="rtl"`) || !strings.Contains(body, `data-color-scheme="dark"`) || !strings.Contains(body, "<p>hello</p>") {
		t.Errorf("frame: %d\n%s", w.Code, body)
	}
	if w := get("/frame?story=x%2Fhello&variant=with+%26+space&dir=ltr&scheme=light"); w.Code != 200 {
		t.Errorf("escaped variant: %d", w.Code)
	}
	for _, bad := range []string{"/frame?story=x/hello&variant=plain&dir=up&scheme=dark", "/frame?story=x/none&variant=plain&dir=ltr&scheme=dark", "/story?name=none"} {
		if w := get(bad); w.Code != 404 {
			t.Errorf("%s: %d, want 404", bad, w.Code)
		}
	}
	w = get("/tokens.json?brand_color=%23b4f601")
	var tokens map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens["--ui-color-brand"] != "#b4f601" || tokens["--ui-color-on-brand"] != "#000000" {
		t.Errorf("tokens: %v", tokens)
	}
}

// Mounted under a prefix, every link stays inside it.
func TestMounted(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/catalog/", http.StripPrefix("/catalog", newCatalog(t)))
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	if w := get("/catalog/"); w.Code != 302 || w.Header().Get("Location") != "story?name=x%2Fhello" {
		t.Errorf("index: %d %q", w.Code, w.Header().Get("Location"))
	}
	body := get("/catalog/story?name=x%2Fhello").Body.String()
	for _, abs := range []string{`href="/assets`, `src="/assets`, `src="/frame`, `href="/story`} {
		if strings.Contains(body, abs) {
			t.Errorf("story page links outside the mount: %s", abs)
		}
	}
	if w := get("/catalog/assets/catalog.css"); w.Code != 200 {
		t.Errorf("assets under the mount: %d", w.Code)
	}
	frame := get("/catalog/frame?story=x%2Fhello&variant=plain&dir=ltr&scheme=light")
	if frame.Code != 200 || strings.Contains(frame.Body.String(), `href="/assets`) {
		t.Errorf("frame under the mount: %d\n%s", frame.Code, frame.Body.String())
	}
	if w := get("/catalog/tokens.json?brand_color=%23000000"); w.Code != 200 || !strings.Contains(w.Body.String(), "--ui-color-brand") {
		t.Errorf("tokens under the mount: %d", w.Code)
	}
}

// Mounted with a trailing slash in the prefix, nothing escapes either.
func TestMountedTrailingSlash(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/catalog/", http.StripPrefix("/catalog/", newCatalog(t)))
	for _, path := range []string{"/catalog/", "/catalog/story?name=x%2Fhello", "/catalog/frame?story=x%2Fhello&variant=plain&dir=ltr&scheme=light"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if loc := w.Header().Get("Location"); strings.HasPrefix(loc, "/") || w.Code >= 300 && w.Code != 302 {
			t.Errorf("%s: %d, Location %q", path, w.Code, loc)
		}
	}
}

func TestNewPanics(t *testing.T) {
	th, _ := theme.New("/ui", theme.Defaults{})
	v := catalog.Variant{Name: "a", Render: func(context.Context) templ.Component { return templ.NopComponent }}
	bad := map[string][]catalog.Story{
		"bad name":     {{Name: "Elements/Button", Variants: []catalog.Variant{v}}},
		"one segment":  {{Name: "button", Variants: []catalog.Variant{v}}},
		"duplicate":    {{Name: "x/y", Variants: []catalog.Variant{v}}, {Name: "x/y", Variants: []catalog.Variant{v}}},
		"no variants":  {{Name: "x/y"}},
		"same variant": {{Name: "x/y", Variants: []catalog.Variant{v, v}}},
	}
	for name, stories := range bad {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: New accepted it", name)
				}
			}()
			catalog.New(th, stories...)
		}()
	}
}
