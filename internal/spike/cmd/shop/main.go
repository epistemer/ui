// Command shop runs the D7.2 spike's toy shop: go run ./internal/spike/cmd/shop.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/epistemer/ui/internal/spike/assets"
	"github.com/epistemer/ui/internal/spike/shop"
	"github.com/epistemer/ui/theme"
)

func main() {
	addr := flag.String("addr", "localhost:7332", "address to listen on")
	flag.Parse()
	th, err := theme.New("/ui/", theme.Defaults{})
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", th)
	mux.Handle("GET /spike/", http.StripPrefix("/spike/", http.FileServerFS(assets.FS)))
	app := &shop.App{Store: shop.NewStore(), Theme: th}
	app.Mount(mux)
	mux.Handle("GET /{$}", http.RedirectHandler("/orders", http.StatusSeeOther))
	log.Printf("shop on http://%s/orders", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
