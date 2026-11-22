package theme

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type cfg struct {
	Defaults
	brand, font, radius string
}

func (c *cfg) ThemeBrandColor() string { return c.brand }
func (c *cfg) ThemeFontSans() string   { return c.font }
func (c *cfg) ThemeRadius() string     { return c.radius }

func newCfg() *cfg {
	d := Defaults{}
	return &cfg{brand: d.ThemeBrandColor(), font: d.ThemeFontSans(), radius: d.ThemeRadius()}
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}

func TestBundle(t *testing.T) {
	b, err := newBundle()
	if err != nil {
		t.Fatal(err)
	}
	css := string(b.Files["ui.css"])
	if !strings.HasPrefix(css, "@layer ui.reset, ui.tokens, ui.base, ui.elements, ui.inputs, ui.components, ui.layout;") {
		t.Errorf("ui.css starts %q", css[:80])
	}
	for _, want := range []string{"--ui-color-brand-text: light-dark(", "@layer ui.tokens {", "[hidden]", "prefers-reduced-motion"} {
		if !strings.Contains(css, want) {
			t.Errorf("ui.css lacks %q", want)
		}
	}
	if _, ok := b.Files["ui.js"]; !ok {
		t.Error("no ui.js")
	}
	again, _ := newBundle()
	if again.Hash != b.Hash {
		t.Errorf("hash not stable: %s, %s", b.Hash, again.Hash)
	}
}

func TestHandler(t *testing.T) {
	c := newCfg()
	th, err := New("/ui", c)
	if err != nil {
		t.Fatal(err)
	}
	hash := th.bundle.Hash

	w := get(t, th, "/ui/"+hash+"/ui.css")
	if w.Code != 200 || w.Header().Get("Cache-Control") != immutable || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/css") {
		t.Errorf("ui.css: %d %q %q", w.Code, w.Header().Get("Cache-Control"), w.Header().Get("Content-Type"))
	}
	if w := get(t, th, "/ui/"+hash+"/ui.js"); w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
		t.Errorf("ui.js: %d %q", w.Code, w.Header().Get("Content-Type"))
	}
	if w := get(t, th, "/ui/0123456789ab/ui.css"); w.Code != 404 {
		t.Errorf("another release's hash: %d, want 404", w.Code)
	}

	_, h1 := th.runtime.current()
	w = get(t, th, "/ui/theme/"+h1+".css")
	if w.Code != 200 || w.Header().Get("Cache-Control") != immutable {
		t.Errorf("current theme: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}

	// An admin changes the brand: the hash changes, the old URL still
	// answers, with the new theme and no caching.
	c.brand = "#b4f601"
	_, h2 := th.runtime.current()
	if h2 == h1 {
		t.Fatal("hash unchanged after a change")
	}
	w = get(t, th, "/ui/theme/"+h1+".css")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-cache" || !strings.Contains(w.Body.String(), "--ui-color-brand: #b4f601;") {
		t.Errorf("stale theme URL: %d %q\n%s", w.Code, w.Header().Get("Cache-Control"), w.Body.String())
	}

	var head strings.Builder
	if err := th.Head().Render(context.Background(), &head); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="/ui/` + hash + `/ui.css"`, `href="/ui/theme/` + h2 + `.css"`, `<script type="module" src="/ui/` + hash + `/ui.js">`} {
		if !strings.Contains(head.String(), want) {
			t.Errorf("head lacks %s:\n%s", want, head.String())
		}
	}
}

func TestRuntimeRejectsInjection(t *testing.T) {
	c := newCfg()
	c.brand = "#fff; } body { display: none"
	c.font = `x; } @import "https://evil.example/x.css`
	c.radius = "1rem; } *{"
	css, _ := newRuntime(c).current()
	s := string(css)
	for _, bad := range []string{"display: none", "@import", "evil", "*{"} {
		if strings.Contains(s, bad) {
			t.Errorf("theme stylesheet contains %q:\n%s", bad, s)
		}
	}
	d := Defaults{}
	for _, want := range []string{"--ui-color-brand: " + d.ThemeBrandColor() + ";", "--ui-font-sans: " + d.ThemeFontSans() + ";", "--ui-radius: " + d.ThemeRadius() + ";"} {
		if !strings.Contains(s, want) {
			t.Errorf("invalid value didn't fall back: want %q in\n%s", want, s)
		}
	}
	if strings.Count(s, "{") != strings.Count(s, "}") {
		t.Errorf("unbalanced braces:\n%s", s)
	}
}

func TestValidators(t *testing.T) {
	for _, s := range []string{"#abc", "#A1B2C3"} {
		if !isColor(s) {
			t.Errorf("ValidColor(%q) = false", s)
		}
	}
	for _, s := range []string{"abc", "#abcd", "red", "#fff;", "rgb(0 0 0)"} {
		if isColor(s) {
			t.Errorf("ValidColor(%q) = true", s)
		}
	}
	for _, s := range []string{"0", "4px", "0.5rem", "2rem", "32px"} {
		if !isRadius(s) {
			t.Errorf("ValidLength(%q) = false", s)
		}
	}
	for _, s := range []string{"33px", "3rem", "-1px", "1", "1vw", "calc(1px)"} {
		if isRadius(s) {
			t.Errorf("ValidLength(%q) = true", s)
		}
	}
	for _, s := range []string{`system-ui, sans-serif`, `"Noto Sans Arabic", "Segoe UI", sans-serif`, `Inter, 'Segoe UI', sans-serif`} {
		if !isFont(s) {
			t.Errorf("ValidFont(%q) = false", s)
		}
	}
	for _, s := range []string{`Comic Sans`, `"a\"b"`, `x; }`, `url(x)`, `"x"; color: red`, `Inter,`, `'x"`} {
		if isFont(s) {
			t.Errorf("ValidFont(%q) = true", s)
		}
	}
}

func TestPrefix(t *testing.T) {
	for _, p := range []string{"", "ui", "//evil.example", "/ui?x", `/ui"`} {
		if _, err := New(p, Defaults{}); err == nil {
			t.Errorf("New(%q) accepted", p)
		}
	}
}

func TestRuntimeConcurrent(t *testing.T) {
	c := &lockedCfg{cfg: *newCfg()}
	r := newRuntime(c)
	done := make(chan bool)
	for i := range 8 {
		go func() {
			for j := range 50 {
				if i == 0 && j%10 == 0 {
					c.set(fmt.Sprintf("#%06x", j*0x10203))
				}
				css, hash := r.current()
				if len(css) == 0 || len(hash) != 12 {
					t.Error("empty theme")
				}
			}
			done <- true
		}()
	}
	for range 8 {
		<-done
	}
}

type lockedCfg struct {
	mu  sync.Mutex
	cfg cfg
}

func (c *lockedCfg) set(brand string) { c.mu.Lock(); c.cfg.brand = brand; c.mu.Unlock() }
func (c *lockedCfg) ThemeBrandColor() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cfg.brand
}
func (c *lockedCfg) ThemeDangerColor() string  { return c.cfg.ThemeDangerColor() }
func (c *lockedCfg) ThemeWarningColor() string { return c.cfg.ThemeWarningColor() }
func (c *lockedCfg) ThemeSuccessColor() string { return c.cfg.ThemeSuccessColor() }
func (c *lockedCfg) ThemeInfoColor() string    { return c.cfg.ThemeInfoColor() }
func (c *lockedCfg) ThemeFontSans() string     { return c.cfg.ThemeFontSans() }
func (c *lockedCfg) ThemeFontMono() string     { return c.cfg.ThemeFontMono() }
func (c *lockedCfg) ThemeRadius() string       { return c.cfg.ThemeRadius() }
