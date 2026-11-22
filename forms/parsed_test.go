package forms_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epistemer/ui/forms"
)

// A body net/http already parsed binds from what it kept, not as a blank
// form (the D7.2 spike found a handler that read ids first).
func TestFromRequestAfterParseForm(t *testing.T) {
	r := httptest.NewRequest("POST", "/?q=query", strings.NewReader("name=Bear&id=1&id=2"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	f := forms.New()
	name := f.Text("name")
	if !f.Submit(context.Background(), r) || name.Value() != "Bear" {
		t.Fatalf("after ParseForm: valid %v, name %q", f.Valid(), name.Value())
	}
	src, err := forms.FromRequest(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := src.Values("q"); got != nil {
		t.Errorf("the query leaked into the body's values: %v", got)
	}
}

func TestFromRequestAfterParseMultipartForm(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("name", "Kite")
	mw.Close()
	r := httptest.NewRequest("POST", "/", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	f := forms.New()
	name := f.Text("name")
	if !f.Submit(context.Background(), r) || name.Value() != "Kite" {
		t.Fatalf("after ParseMultipartForm: valid %v, name %q", f.Valid(), name.Value())
	}
}
