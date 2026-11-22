// Command catalog serves ui's own stories: go run ./cmd/catalog.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/epistemer/ui/catalog"
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
	mux.Handle("/", catalog.New(t, stories.All()...))
	log.Fatal(http.ListenAndServe(*addr, mux))
}
