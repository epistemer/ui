package ui_test

import (
	"testing"

	"github.com/epistemer/ui"
)

// A function declared over the bare interface takes a ui.Link, and the
// reverse, with no conversion.
func takesBare(l interface{ URL() string }) string { return l.URL() }

func TestURL(t *testing.T) {
	var l ui.Link = ui.URL("/docs")
	if got := takesBare(l); got != "/docs" {
		t.Errorf("URL(%q).URL() = %q", "/docs", got)
	}
	if got := ui.URL("").URL(); got != "" {
		t.Errorf("URL(\"\").URL() = %q", got)
	}
}
