package stories

import (
	"testing"

	"github.com/epistemer/ui/catalog/catalogtest"
	"github.com/epistemer/ui/internal/themehook"
)

// TestStories runs every check on every variant of ui's own stories, and
// the reverse check on ui's CSS: every component rule is used by some
// story (visual-interface.md §9.1).
func TestStories(t *testing.T) {
	themehook.ReverseCheck = true
	defer func() { themehook.ReverseCheck = false }()
	catalogtest.Check(t, All())
}
