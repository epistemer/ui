package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epistemer/ui/internal/iconlist"
	"github.com/epistemer/ui/theme"
)

func TestIconsRoute(t *testing.T) {
	th, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	mux := newMux(th)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/icons", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /icons: status %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"ArrowStart", "Pencil", "Trash2", "ExternalLink", "Upload", "Download", "Calendar", `<svg class="ui-icon"`} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /icons missing %q", want)
		}
	}
	if len(iconlist.All) < 2410 {
		t.Errorf("iconlist.All has %d entries, expected >= 2410", len(iconlist.All))
	}
}
