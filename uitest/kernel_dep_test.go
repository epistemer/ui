package uitest_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find module root containing go.mod")
		}
		dir = parent
	}
}

func TestKernelDoesNotDependOnXNet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dependency test under -short")
	}

	root := findModuleRoot(t)
	cmd := exec.Command("go", "list", "-deps", "github.com/epistemer/ui")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps failed: %v\noutput:\n%s", err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		pkg := strings.TrimSpace(line)
		if strings.HasPrefix(pkg, "golang.org/x/net") {
			t.Errorf("kernel github.com/epistemer/ui links %s; kernel must not link golang.org/x/net", pkg)
		}
	}
}
