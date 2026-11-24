// Command catalog serves ui's own stories: go run ./cmd/catalog.
package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/catalog"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/iconlist"
	"github.com/epistemer/ui/internal/stories"
	"github.com/epistemer/ui/internal/toyshop"
	"github.com/epistemer/ui/theme"
)

func newMux(t *theme.Theme) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/ui/", t)
	// What the htmx stories ask for: a toast, out of band.
	mux.HandleFunc("POST /demo/delete", func(w http.ResponseWriter, r *http.Request) {
		components.ToastSwap(components.Toast(i18n.Static("The store was deleted.")).Tone(elements.Success)).Render(r.Context(), w)
	})
	mux.HandleFunc("GET /demo/stock", func(w http.ResponseWriter, r *http.Request) {
		components.ToastSwap(components.Toast(i18n.Static("Three toys are almost out of stock.")).Tone(elements.Warning)).Render(r.Context(), w)
	})
	stories.Demo(mux)
	// The toy shop's tables and actions, a page of their own with htmx.
	shop := &toyshop.App{Store: toyshop.NewStore(), Theme: t, Scripts: templ.Raw(`<meta name="htmx-config" content='{"includeIndicatorStyles":false,"historyCacheSize":0}'><script src="/assets/htmx.min.js"></script>`)}
	shop.Mount(mux)
	mux.HandleFunc("GET /icons", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		if err := iconsPage(t.Head(), iconlist.All).Render(r.Context(), &b); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:")
		w.Write(b.Bytes())
	})
	mux.Handle("/", catalog.New(t, stories.All()...))
	return mux
}

func main() {
	addr := flag.String("addr", "localhost:7331", "address to listen on")
	flag.Parse()
	t, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ui catalogue on http://%s", *addr)
	log.Printf("toy shop on http://%s%s/orders", *addr, toyshop.Prefix)
	log.Fatal(http.ListenAndServe(*addr, newMux(t)))
}
