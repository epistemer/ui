package spike

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/epistemer/ui/theme"
	"github.com/epistemer/ui/uitest"
)

func serve(t *testing.T) *http.ServeMux {
	t.Helper()
	th, err := theme.New("/ui", theme.Defaults{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", th)
	(&App{Theme: th}).Mount(mux)
	return mux
}

func get(t *testing.T, mux *http.ServeMux, method, target string, body url.Values) (int, string) {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, strings.NewReader(body.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// css is ui's stylesheet plus the spike's, for CheckSelectors.
func css(t *testing.T, mux *http.ServeMux) string {
	t.Helper()
	_, page := get(t, mux, "GET", Prefix+"/order", nil)
	href := regexp.MustCompile(`href="(/ui/[^"]+/ui\.css)"`).FindStringSubmatch(page)
	_, ui := get(t, mux, "GET", href[1], nil)
	own, err := os.ReadFile("assets/spike.css")
	if err != nil {
		t.Fatal(err)
	}
	return ui + string(own)
}

var external = uitest.External("ui-action-form", "ui-action-dialog-body", "ui-toasts")

// The pages pass the id, name and selector checks with every new shape:
// the script-off combobox, a list with its template, an upload.
func TestPages(t *testing.T) {
	mux := serve(t)
	styles := css(t, mux)
	for _, p := range []string{"/order", "/table"} {
		code, html := get(t, mux, "GET", Prefix+p, nil)
		if code != 200 {
			t.Fatalf("%s: %d %s", p, code, html)
		}
		uitest.CheckIDs(t, html, external)
		uitest.CheckNames(t, html)
		uitest.CheckSelectors(t, html, styles, uitest.Prefix("ui-"))
	}
}

// A refused search link renders no data-ui-search, so the script leaves
// the text input of the id (R7).
func TestRefused(t *testing.T) {
	_, html := get(t, serve(t), "GET", Prefix+"/table?refused_id=4", nil)
	m := regexp.MustCompile(`<ui-combobox[^>]*>\s*<input[^>]*name="refused_id"`).FindString(html)
	if m == "" || strings.Contains(m, "data-ui-search") {
		t.Errorf("the refused combobox: %q", m)
	}
	if !strings.Contains(html, `class="ui-combobox-chosen" id="`) || !strings.Contains(html, ">Dina Farouk</span>") {
		t.Error("the refused field doesn't show the chosen name")
	}
}

// LabelOf: memoized per request (R10); text that didn't convert is never
// looked up; a miss shows the id; an error fails the render (R8).
func TestLabelOf(t *testing.T) {
	mux := serve(t)
	form := url.Values{"customer_id": {"7"}}
	for i := range 5 {
		k := "lines[" + string(rune('0'+i)) + "]"
		form.Set(k+".product_id", "103")
		form.Set(k+".qty", "")
	}
	productTable.calls = 0
	code, html := get(t, mux, "POST", Prefix+"/order", form)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", code)
	}
	if productTable.calls != 1 {
		t.Errorf("five lines with one product made %d calls, want 1", productTable.calls)
	}
	if n := strings.Count(html, ">Brick 2x2 yellow</span>"); n != 5 {
		t.Errorf("%d names shown, want 5", n)
	}

	customerTable.calls = 0
	_, html = get(t, mux, "POST", Prefix+"/order", url.Values{"customer_id": {"abc"}})
	if customerTable.calls != 0 {
		t.Errorf("text that didn't convert was looked up %d times", customerTable.calls)
	}
	if !strings.Contains(html, `value="abc"`) {
		t.Error("the typed text isn't kept")
	}

	_, html = get(t, mux, "POST", Prefix+"/order", url.Values{"customer_id": {"999"}})
	if !strings.Contains(html, `>999</span>`) {
		t.Error("a miss doesn't show the id")
	}

	customerTable.fail = true
	defer func() { customerTable.fail = false }()
	if code, _ := get(t, mux, "GET", Prefix+"/order", nil); code != 500 {
		t.Errorf("a failing Labeler gave %d, want 500", code)
	}
}

// An answer is a ul[data-ui-suggestions] of values and texts, with
// data-ui-more past the limit, and no ids or roles (§3.3).
func TestSuggestions(t *testing.T) {
	mux := serve(t)
	_, all := get(t, mux, "GET", Prefix+"/search/products?q=", nil)
	if !strings.HasPrefix(strings.TrimSpace(all), "<ul data-ui-suggestions data-ui-more>") || strings.Count(all, "<li ") != SuggestionLimit {
		t.Errorf("a long answer: %s", all)
	}
	if strings.Contains(all, " id=") || strings.Contains(all, "role=") {
		t.Errorf("an answer carries ids or roles: %s", all)
	}
	_, few := get(t, mux, "GET", Prefix+"/search/customers?q=zei", nil)
	if strings.Contains(few, "data-ui-more") || !strings.Contains(few, `<li data-value="26">Zeid Amin</li>`) {
		t.Errorf("a short answer: %s", few)
	}
}

// Kept keys: one not given is dropped, a repeat counts once (§4.2).
func TestKeep(t *testing.T) {
	mux := serve(t)
	_, html := get(t, mux, "POST", Prefix+"/order", url.Values{
		"customer_id": {"7"}, "lines[0].product_id": {"103"}, "lines[0].qty": {"1"},
		"receipt.keep": {"k-82", "k-82", "k-99"},
	})
	if !strings.Contains(html, "receipt kept [k-82]") {
		t.Errorf("kept keys: %s", regexp.MustCompile(`Saved:[^<]*`).FindString(html))
	}
}
