// Package catalog shows components in their states (visual-interface.md
// §9): a registry of stories and the pages that show each variant in LTR
// and RTL, light and dark, with a width switcher and a token panel that
// re-themes them live. Its URLs are relative, so mount it anywhere:
//
//	mux.Handle("/ui/", t) // the theme the frames link
//	mux.Handle("/catalog/", http.StripPrefix("/catalog", catalog.New(t, stories...)))
//
// catalogtest runs every check on every variant.
package catalog

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

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

var segment = regexp.MustCompile(`^[a-z0-9-]+/[a-z0-9-]+$`)

// New returns a catalogue of stories whose frames link t's stylesheet and
// script; the app mounts t too. It panics on a story name that isn't
// "<segment>/<segment>" of [a-z0-9-], a duplicate story, a story without
// variants, or two variants of one story with the same name.
func New(t *theme.Theme, stories ...Story) *Catalog {
	if t == nil {
		panic("catalog: New with a nil theme")
	}
	seen := map[string]bool{}
	for _, s := range stories {
		switch {
		case !segment.MatchString(s.Name):
			panic(fmt.Sprintf("catalog: story %q: a name is <package>/<name>, of a-z, 0-9 and -", s.Name))
		case seen[s.Name]:
			panic(fmt.Sprintf("catalog: story %q registered twice", s.Name))
		case len(s.Variants) == 0:
			panic(fmt.Sprintf("catalog: story %q has no variants", s.Name))
		}
		seen[s.Name] = true
		names := map[string]bool{}
		for _, v := range s.Variants {
			if names[v.Name] || v.Name == "" || v.Render == nil {
				panic(fmt.Sprintf("catalog: story %q: variant %q is empty, has no Render, or appears twice", s.Name, v.Name))
			}
			names[v.Name] = true
		}
	}
	c := &Catalog{theme: t, stories: slices.Clone(stories), mux: http.NewServeMux()}
	c.mux.Handle("GET /assets/", http.FileServerFS(assets))
	c.mux.HandleFunc("GET /{$}", c.index)
	c.mux.HandleFunc("GET /story", c.story)
	c.mux.HandleFunc("GET /frame", c.frame)
	c.mux.HandleFunc("GET /tokens.json", c.tokens)
	return c
}

// ServeHTTP serves the index, a page per story, a frame per variant,
// direction and scheme, the catalogue's assets and the token panel.
func (c *Catalog) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Mounted with http.StripPrefix("/catalog/", …), the path loses its
	// leading slash, and the mux would answer with a redirect to the root,
	// out of the mount. Put the slash back.
	if !strings.HasPrefix(r.URL.Path, "/") {
		r2 := *r
		u := *r.URL
		u.Path = "/" + u.Path
		if u.RawPath != "" {
			u.RawPath = "/" + u.RawPath
		}
		r2.URL = &u
		r = &r2
	}
	c.mux.ServeHTTP(w, r)
}

// Stories returns the registered stories.
func (c *Catalog) Stories() []Story { return slices.Clone(c.stories) }

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
	// A relative Location, written as it is: http.Redirect would make it
	// absolute from the path StripPrefix left, losing the mount point.
	w.Header().Set("Location", storyURL(c.stories[0].Name))
	w.WriteHeader(http.StatusFound)
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
	// Render first, so a failing story is a 500 rather than half a page.
	var b bytes.Buffer
	if err := c.Render(r.Context(), &b); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The catalogue runs under the CSP ui promises apps (ADR 0005).
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:")
	w.Write(b.Bytes())
}

func storyURL(name string) string {
	return "story?" + url.Values{"name": {name}}.Encode()
}

func frameURL(story, variant, dir, scheme string) string {
	return "frame?" + url.Values{"story": {story}, "variant": {variant}, "dir": {dir}, "scheme": {scheme}}.Encode()
}
