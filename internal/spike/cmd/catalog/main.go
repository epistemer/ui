// Command catalog serves the D4 spike's stories: go run ./internal/spike/cmd/catalog.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/epistemer/ui/internal/spike/catalog"
	"github.com/epistemer/ui/internal/spike/stories"
	"github.com/epistemer/ui/internal/spike/theme"
)

func main() {
	addr := flag.String("addr", "localhost:7331", "address to listen on")
	flag.Parse()
	t, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ui catalogue on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, catalog.New(t, stories.All()...)))
}
