package rule_test

import (
	"os"
	"os/exec"
	"path/filepath"
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
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("%s compiled successfully; expected a build error\nOutput:\n%s", relPath, out)
			}
		})
	}
}
