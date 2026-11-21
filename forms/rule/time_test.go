package rule_test

import (
	"context"
	"testing"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms/internal/hook"
	"github.com/epistemer/ui/forms/rule"
)

func TestTimeBounds(t *testing.T) {
	noon := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	midnight := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()
	dateOnly := func(r rule.Rule[time.Time]) rule.Rule[time.Time] { return hook.DateOnly(r).(rule.Rule[time.Time]) }
	for _, tc := range []struct {
		name string
		r    rule.Rule[time.Time]
		v    time.Time
		want error
	}{
		{"before the bound", rule.NotBefore(noon), noon.Add(-time.Hour), rule.ErrNotBefore},
		{"at the bound", rule.NotBefore(noon), noon, nil},
		{"after a NotAfter bound", rule.NotAfter(noon), noon.Add(time.Second), rule.ErrNotAfter},
		{"today's date against now, instants", rule.NotBefore(noon), midnight, rule.ErrNotBefore},
		{"today's date against now, date only", dateOnly(rule.NotBefore(noon)), midnight, nil},
		{"yesterday, date only", dateOnly(rule.NotBefore(noon)), midnight.Add(-time.Hour), rule.ErrNotBefore},
		{"NotAfter date only", dateOnly(rule.NotAfter(midnight)), noon, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := rule.Check(ctx, tc.v, tc.r)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
	other := rule.New(func(context.Context, string) error { return nil })
	if _, ok := hook.DateOnly(other).(rule.Rule[string]); !ok {
		t.Error("DateOnly must return other values unchanged")
	}
}

func TestTimeAttrs(t *testing.T) {
	noon := time.Date(2026, 11, 21, 12, 30, 0, 0, time.UTC)
	for k, v := range rule.NotBefore(noon).Attrs() {
		if k != "min" || v != "2026-11-21T12:30" {
			t.Errorf("got %s=%q", k, v)
		}
	}
}
