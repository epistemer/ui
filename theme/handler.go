package theme

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// Theme serves ui's assets and the run-time theme under a prefix, and
// renders the head tags that load them.
type Theme struct {
	prefix  string // such as "/ui", without a trailing slash
	bundle  *bundle
	runtime *runtimeTheme
}

// New returns a Theme mounted at prefix, reading run-time values from c.
// The prefix is a path on the app's own origin: it starts with one slash.
func New(prefix string, c Config) (*Theme, error) {
	if !strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "//") || strings.ContainsAny(prefix, `\?#"'<> `) {
		return nil, fmt.Errorf("theme: prefix %q: want a path on this origin, such as /ui", prefix)
	}
	b, err := newBundle()
	if err != nil {
		return nil, err
	}
	return &Theme{prefix: strings.TrimSuffix(prefix, "/"), bundle: b, runtime: newRuntime(c)}, nil
}

const immutable = "public, max-age=31536000, immutable"

// ServeHTTP serves <prefix>/<hash>/<file> and <prefix>/theme/<hash>.css.
func (t *Theme) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rest, ok := strings.CutPrefix(r.URL.Path, t.prefix+"/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if name, ok := strings.CutPrefix(rest, "theme/"); ok {
		css, hash := t.runtime.current()
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if name == hash+".css" {
			w.Header().Set("Cache-Control", immutable)
		} else {
			// A page rendered before a change: give it the current theme,
			// and don't let anything cache it under the old name.
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.Write(css)
		return
	}
	hash, file, ok := strings.Cut(rest, "/")
	if !ok || hash != t.bundle.Hash {
		http.NotFound(w, r)
		return
	}
	b, ok := t.bundle.Files[file]
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(file, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(file, ".js"):
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	}
	w.Header().Set("Cache-Control", immutable)
	w.Write(b)
}

// Head renders the stylesheet, the theme stylesheet and the script, in
// that order: the theme's tokens come after ui's own, in the same layer.
func (t *Theme) Head() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, hash := t.runtime.current()
		base := t.prefix + "/" + t.bundle.Hash + "/"
		_, err := io.WriteString(w,
			`<link rel="stylesheet" href="`+templ.EscapeString(base+"ui.css")+`">`+
				`<link rel="stylesheet" href="`+templ.EscapeString(t.prefix+"/theme/"+hash+".css")+`">`+
				`<script type="module" src="`+templ.EscapeString(base+"ui.js")+`"></script>`)
		return err
	})
}
