// Command catalog serves ui's own stories: go run ./cmd/catalog.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/catalog"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/stories"
	"github.com/epistemer/ui/theme"
)

func main() {
	addr := flag.String("addr", "localhost:7331", "address to listen on")
	flag.Parse()
	t, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ui catalogue on http://%s", *addr)
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
	mux.Handle("/", catalog.New(t, stories.All()...))
	log.Fatal(http.ListenAndServe(*addr, mux))
}
