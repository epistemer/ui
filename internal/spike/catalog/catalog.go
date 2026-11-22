// Package catalog is the D4 spike of ui/catalog (visual-design.md §8): a
// registry of stories and the page that shows them in LTR and RTL, light
// and dark, with a token panel that re-themes them live.
package catalog

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"

	"github.com/a-h/templ"

	"github.com/epistemer/ui/internal/themehook"
	"github.com/epistemer/ui/theme"
)

// Variant is one state of a story.
type Variant struct {
	Name   string
	Render func(ctx context.Context) templ.Component
}

// Story is a component shown in its variants. Name is "<package>/<name>".
type Story struct {
	Name     string
	Variants []Variant
}

// Catalog serves stories.
type Catalog struct {
	theme   *theme.Theme
	stories []Story
	mux     *http.ServeMux
}

//go:embed assets/*
var assets embed.FS

// New returns a catalogue of stories, styled and themed by t, which it
// serves under /ui/.
func New(t *theme.Theme, stories ...Story) *Catalog {
	c := &Catalog{theme: t, stories: stories, mux: http.NewServeMux()}
	c.mux.Handle("GET /ui/", t)
	c.mux.Handle("GET /assets/", http.FileServerFS(assets))
	c.mux.HandleFunc("GET /{$}", c.index)
	c.mux.HandleFunc("GET /story", c.story)
	c.mux.HandleFunc("GET /frame", c.frame)
	c.mux.HandleFunc("GET /tokens.json", c.tokens)
	return c
}

func (c *Catalog) ServeHTTP(w http.ResponseWriter, r *http.Request) { c.mux.ServeHTTP(w, r) }

// Stories returns the registered stories.
func (c *Catalog) Stories() []Story { return c.stories }

func (c *Catalog) find(name string) (Story, bool) {
	i := slices.IndexFunc(c.stories, func(s Story) bool { return s.Name == name })
	if i < 0 {
		return Story{}, false
	}
	return c.stories[i], true
}

func (c *Catalog) index(w http.ResponseWriter, r *http.Request) {
	if len(c.stories) == 0 {
		http.Error(w, "no stories", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, storyURL(c.stories[0].Name), http.StatusFound)
}

func (c *Catalog) story(w http.ResponseWriter, r *http.Request) {
	s, ok := c.find(r.URL.Query().Get("name"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, r, shell(c.theme.Head(), c.stories, s))
}

// Frame directions and schemes, as the shell lays them out.
var (
	dirs    = []string{"ltr", "rtl"}
	schemes = []string{"light", "dark"}
)

func (c *Catalog) frame(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s, ok := c.find(q.Get("story"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	i := slices.IndexFunc(s.Variants, func(v Variant) bool { return v.Name == q.Get("variant") })
	dir, scheme := q.Get("dir"), q.Get("scheme")
	if i < 0 || !slices.Contains(dirs, dir) || !slices.Contains(schemes, scheme) {
		http.NotFound(w, r)
		return
	}
	v := s.Variants[i]
	render(w, r, frame(c.theme.Head(), s.Name+" — "+v.Name, dir, scheme, v.Render(r.Context())))
}

// tokens answers the token panel: the run-time tokens for the values in
// the query (themeschema's key names), derived in Go as the theme
// stylesheet derives them.
func (c *Catalog) tokens(w http.ResponseWriter, r *http.Request) {
	values := map[string]string{}
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			values[k] = v[0]
		}
	}
	out := map[string]string{}
	for _, kv := range themehook.Tokens(values) {
		out[kv[0]] = kv[1]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The catalogue runs under the CSP ui promises apps (ADR 0005).
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:")
	if err := c.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func storyURL(name string) string {
	return "/story?" + url.Values{"name": {name}}.Encode()
}

func frameURL(story, variant, dir, scheme string) string {
	return "/frame?" + url.Values{"story": {story}, "variant": {variant}, "dir": {dir}, "scheme": {scheme}}.Encode()
}
