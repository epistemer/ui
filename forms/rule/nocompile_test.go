package rule_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile-time safety test in short mode")
	}

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get working directory: %v", err)
	}

	nocompileDir := filepath.Join(dir, "testdata", "nocompile")
	entries, err := os.ReadDir(nocompileDir)
	if err != nil {
		t.Fatalf("could not read %s: %v", nocompileDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			relPath := "./testdata/nocompile/" + entry.Name()
			cmd := exec.Command("go", "build", "-o", os.DevNull, relPath)
			cmd.Dir = dir
			src, err := os.ReadFile(filepath.Join(nocompileDir, entry.Name(), "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			want := wantError(t, string(src))
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("%s compiled; expected a build error", relPath)
			}
			if !strings.Contains(string(out), want) {
				t.Errorf("%s failed for another reason; want %q in:\n%s", relPath, want, out)
			}
		})
	}
}

// wantError reads the case's expected compiler message from its first
// line, written as: // want: <text>
func wantError(t *testing.T, src string) string {
	t.Helper()
	first, _, _ := strings.Cut(src, "\n")
	want, ok := strings.CutPrefix(first, "// want: ")
	if !ok || want == "" {
		t.Fatalf("a nocompile case must start with // want: <compiler message>")
	}
	return want
}
