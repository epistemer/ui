package formstest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/formstest"
)

type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(format string, a ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, a...))
}

func (r *recorder) Helper() {}

func TestAssertLabelled(t *testing.T) {
	// A form holding a hidden field without a label: no failure
	f := forms.New()
	f.Text("token").Hint(forms.KindHidden)
	rec := &recorder{TB: t}
	formstest.AssertLabelled(rec, f)
	if len(rec.errs) != 0 {
		t.Errorf("hidden field without label failed AssertLabelled: %v", rec.errs)
	}

	// A text field without a label still fails
	f.Text("title")
	rec2 := &recorder{TB: t}
	formstest.AssertLabelled(rec2, f)
	if len(rec2.errs) != 1 || !strings.Contains(rec2.errs[0], "title") {
		t.Errorf("want one failure, for title (the hidden field skipped): %q", rec2.errs)
	}
}
