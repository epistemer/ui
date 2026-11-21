package catalog_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/internal/spike/catalog"
	"github.com/epistemer/ui/internal/spike/theme"
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
	if w := get("/"); w.Code != 302 || w.Header().Get("Location") != "/story?name=x%2Fhello" {
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
	w = get("/tokens.json?brand=%23b4f601")
	var tokens map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	if tokens["--ui-color-brand"] != "#b4f601" || tokens["--ui-color-on-brand"] != "#000000" {
		t.Errorf("tokens: %v", tokens)
	}
}
