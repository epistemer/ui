package themeschema

import "testing"

// The group is checked end to end by internal/testapp, whose generated
// settings must satisfy theme.Config with theme's defaults and checks.
func TestGroup(t *testing.T) {
	if Group() == nil {
		t.Fatal("Group returned nil")
	}
}
