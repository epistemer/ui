package uitest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epistemer/ui/internal/kerneltest"
	"github.com/epistemer/ui/uitest"
)

func readFixture(t *testing.T, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", filename, err)
	}
	return string(data)
}

func TestCheckIDs_Fixtures(t *testing.T) {
	t.Run("duplicate id", func(t *testing.T) {
		html := readFixture(t, "duplicate_id.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)

		if !fake.failed {
			t.Fatal("expected failure on duplicate id")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"id\"") && strings.Contains(err, "\"alpha\"") && strings.Contains(err, "span") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'id', tag 'span', and id 'alpha'; got errors: %v", fake.errors)
		}
	})

	t.Run("for pointing nowhere", func(t *testing.T) {
		html := readFixture(t, "broken_for.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)

		if !fake.failed {
			t.Fatal("expected failure on broken for reference")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"for\"") && strings.Contains(err, "\"nonexistent-input\"") && strings.Contains(err, "label") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'for', tag 'label', and id 'nonexistent-input'; got: %v", fake.errors)
		}
	})

	t.Run("broken hx-target", func(t *testing.T) {
		html := readFixture(t, "broken_hx_target.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)

		if !fake.failed {
			t.Fatal("expected failure on broken hx-target")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"hx-target\"") && strings.Contains(err, "\"missing-pane\"") && strings.Contains(err, "button") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'hx-target', tag 'button', and id 'missing-pane'; got: %v", fake.errors)
		}
	})

	t.Run("IDREFS with one missing", func(t *testing.T) {
		html := readFixture(t, "missing_idref.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)

		if !fake.failed {
			t.Fatal("expected failure on missing id in IDREFS list")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"aria-describedby\"") && strings.Contains(err, "\"second-missing\"") && strings.Contains(err, "input") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'aria-describedby', tag 'input', and id 'second-missing'; got: %v", fake.errors)
		}
	})

	t.Run("template isolation", func(t *testing.T) {
		html := readFixture(t, "template_isolation.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)

		if !fake.failed {
			t.Fatal("expected failure due to template isolation")
		}
		if len(fake.errors) != 2 {
			t.Fatalf("expected 2 errors, got %d: %v", len(fake.errors), fake.errors)
		}
		// Inside template referencing outside ID outer-scope must fail
		foundInside := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"href\"") && strings.Contains(err, "\"outer-scope\"") && strings.Contains(err, "a") {
				foundInside = true
				break
			}
		}
		if !foundInside {
			t.Errorf("expected error for inside referencing outer-scope; got: %v", fake.errors)
		}
		// Outside template referencing inside ID inner-scope must fail
		foundOutside := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"aria-controls\"") && strings.Contains(err, "\"inner-scope\"") && strings.Contains(err, "button") {
				foundOutside = true
				break
			}
		}
		if !foundOutside {
			t.Errorf("expected error for outside referencing inner-scope; got: %v", fake.errors)
		}
	})

	t.Run("External", func(t *testing.T) {
		html := readFixture(t, "external.html")
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html, uitest.External("remote-drawer", "remote-dialog"))

		if fake.failed {
			t.Fatalf("expected pass when referenced IDs are listed in External, got: %v", fake.errors)
		}
	})

	t.Run("correct page passes", func(t *testing.T) {
		html := uitest.Render(t, kerneltest.Field())
		uitest.CheckIDs(t, html)
	})
}

func TestCheckIDs_EdgeCases(t *testing.T) {
	t.Run("empty id attribute", func(t *testing.T) {
		html := `<div id=""></div>`
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)
		if !fake.failed {
			t.Fatal("expected failure on empty id")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"id\"") && strings.Contains(err, "div") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'id' and tag 'div'; got: %v", fake.errors)
		}
	})

	t.Run("empty single-id reference", func(t *testing.T) {
		html := `<label for="">Name</label>`
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)
		if !fake.failed {
			t.Fatal("expected failure on empty for reference")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"for\"") && strings.Contains(err, "label") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'for' and tag 'label'; got: %v", fake.errors)
		}
	})

	t.Run("empty ID-list reference", func(t *testing.T) {
		html := `<input aria-controls="   "/>`
		fake := &fakeTB{TB: t}
		uitest.CheckIDs(fake, html)
		if !fake.failed {
			t.Fatal("expected failure on empty aria-controls")
		}
		found := false
		for _, err := range fake.errors {
			if strings.Contains(err, "attribute \"aria-controls\"") && strings.Contains(err, "input") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error naming attribute 'aria-controls' and tag 'input'; got: %v", fake.errors)
		}
	})

	t.Run("ignored href and hx selectors", func(t *testing.T) {
		html := `<a href="#">Top</a>
<a href="https://example.com">Ext</a>
<button hx-target="this" hx-include="closest form" hx-indicator="closest .spinner" hx-select="table > tbody" hx-disabled-elt="this">Submit</button>`
		uitest.CheckIDs(t, html)
	})
}
