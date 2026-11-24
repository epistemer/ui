package stories

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epistemer/ui/catalog/catalogtest"
	"github.com/epistemer/ui/internal/themehook"
)

// TestStories runs every check on every variant of ui's own stories, and
// the reverse check on ui's CSS: every component rule is used by some
// story (visual-interface.md §9.1).
func TestStories(t *testing.T) {
	themehook.ReverseCheck = true
	themehook.ScriptSelectors = map[string]bool{
		".ui-tristate-track":             true,
		".ui-tristate-track[data-state]": true,
		".ui-tristate-knob":              true,
		".ui-tristate-state":             true,
	}
	defer func() {
		themehook.ReverseCheck = false
		themehook.ScriptSelectors = nil
	}()
	catalogtest.Check(t, All())
}

func TestDemoCustomerSearch(t *testing.T) {
	mux := http.NewServeMux()
	Demo(mux)

	// Substring, case-insensitive.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/demo/customers/search?q=ANN", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `data-value="7"`) || !strings.Contains(w.Body.String(), "Ann Lee") || strings.Contains(w.Body.String(), "data-ui-more") {
		t.Errorf("q=ANN: %d\n%s", w.Code, w.Body.String())
	}

	// Empty q fetches SuggestionLimit+1 so data-ui-more is rendered.
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/demo/customers/search", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "data-ui-more") {
		t.Errorf("empty q should include data-ui-more: %d\n%s", w.Code, w.Body.String())
	}
}
