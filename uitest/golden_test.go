package uitest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epistemer/ui/internal/kerneltest"
	"github.com/epistemer/ui/uitest"
)

type fakeTB struct {
	testing.TB
	errors []string
	fatals []string
	failed bool
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Errorf(format string, args ...any) {
	f.failed = true
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeTB) Fatalf(format string, args ...any) {
	f.failed = true
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
}

func (f *fakeTB) Fail() {
	f.failed = true
}

func (f *fakeTB) Failed() bool {
	return f.failed
}

func TestGoldenAt(t *testing.T) {
	tempDir := t.TempDir()
	goldenPath := filepath.Join(tempDir, "subdir", "component.golden.html")

	htmlOriginal := `<div><label for="name">Name</label><input id="name" type="text" value="Alice"></div>`
	htmlChangedAttr := `<div><label for="name">Name</label><input id="name" type="text" value="Bob"></div>`

	// 1. Missing file without -update fails with message saying to run go test -update.
	fake1 := &fakeTB{TB: t}
	uitest.GoldenAt(fake1, goldenPath, htmlOriginal, false)
	if !fake1.failed {
		t.Fatal("expected failure on missing golden file without -update")
	}
	allMessages1 := strings.Join(append(fake1.errors, fake1.fatals...), "\n")
	if !strings.Contains(allMessages1, "go test -update") {
		t.Errorf("expected failure message to say 'go test -update', got: %s", allMessages1)
	}

	// 2. With -update flag, writes file (creating directory) and passes.
	fake2 := &fakeTB{TB: t}
	uitest.GoldenAt(fake2, goldenPath, htmlOriginal, true)
	if fake2.failed {
		t.Fatalf("expected pass with -update=true, failed with: %v %v", fake2.errors, fake2.fatals)
	}
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file was not written: %v", err)
	}
	normalisedOriginal := uitest.Normalise(htmlOriginal)
	if string(data) != normalisedOriginal {
		t.Errorf("written golden mismatch\ngot:\n%s\nwant:\n%s", string(data), normalisedOriginal)
	}

	// 3. Compare existing golden file with matching content: passes.
	fake3 := &fakeTB{TB: t}
	uitest.GoldenAt(fake3, goldenPath, htmlOriginal, false)
	if fake3.failed {
		t.Errorf("expected pass when content matches golden, failed: %v %v", fake3.errors, fake3.fatals)
	}

	// 4. Compare with changed attribute: fails and shows line diff.
	fake4 := &fakeTB{TB: t}
	uitest.GoldenAt(fake4, goldenPath, htmlChangedAttr, false)
	if !fake4.failed {
		t.Fatal("expected failure on changed attribute")
	}
	if len(fake4.errors) == 0 {
		t.Fatal("expected an Errorf message with diff")
	}
	diffMsg := fake4.errors[0]

	// 5. Verify a changed attribute shows as exactly one differing line.
	var minusCount, plusCount int
	for _, line := range strings.Split(diffMsg, "\n") {
		if strings.HasPrefix(line, "- ") {
			minusCount++
		}
		if strings.HasPrefix(line, "+ ") {
			plusCount++
		}
	}
	if minusCount != 1 {
		t.Errorf("expected exactly 1 '-' line in diff, got %d in:\n%s", minusCount, diffMsg)
	}
	if plusCount != 1 {
		t.Errorf("expected exactly 1 '+' line in diff, got %d in:\n%s", plusCount, diffMsg)
	}
}

func TestGoldenReal(t *testing.T) {
	rendered := uitest.Render(t, kerneltest.Field())
	uitest.Golden(t, "field", rendered)
}
