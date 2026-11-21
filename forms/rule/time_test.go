package rule_test

import (
	"context"
	"strings"
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
	plus3 := time.FixedZone("+03", 3*3600)
	for _, tc := range []struct {
		name      string
		r         rule.Rule[time.Time]
		key, want string
	}{
		{"NotBefore", rule.NotBefore(time.Date(2026, 11, 21, 12, 30, 0, 0, time.UTC)), "min", "2026-11-21T12:30"},
		{"NotAfter", rule.NotAfter(time.Date(2026, 11, 21, 12, 30, 0, 0, time.UTC)), "max", "2026-11-21T12:30"},
		{"seconds kept", rule.NotBefore(time.Date(2026, 11, 21, 12, 30, 15, 0, time.UTC)), "min", "2026-11-21T12:30:15"},
		{"in UTC", rule.NotBefore(time.Date(2026, 11, 22, 1, 0, 0, 0, plus3)), "min", "2026-11-21T22:00"},
		{"date only", hook.DateOnly(rule.NotAfter(time.Date(2026, 11, 21, 12, 30, 0, 0, time.UTC))).(rule.Rule[time.Time]), "max", "2026-11-21"},
	} {
		n := 0
		for k, v := range tc.r.Attrs() {
			n++
			if k != tc.key || v != tc.want {
				t.Errorf("%s: got %s=%q, want %s=%q", tc.name, k, v, tc.key, tc.want)
			}
		}
		if n != 1 {
			t.Errorf("%s: %d attributes, want 1", tc.name, n)
		}
	}
}

func TestTimeErrorShowsTheUTCDate(t *testing.T) {
	plus3 := time.FixedZone("+03", 3*3600)
	bound := time.Date(2026, 11, 22, 1, 0, 0, 0, plus3) // 2026-11-21 22:00 UTC
	r := hook.DateOnly(rule.NotBefore(bound)).(rule.Rule[time.Time])
	err := rule.Check(context.Background(), time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), r)
	if !errors.Is(err, rule.ErrNotBefore) || !strings.Contains(errors.T(context.Background(), err), "2026-11-21") {
		t.Errorf("got %v (%q), want the bound's UTC date 2026-11-21", err, errors.T(context.Background(), err))
	}
}

func TestDateOnlyKeepsErr(t *testing.T) {
	r := hook.DateOnly(rule.NotBefore(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)).Err(errCustom)).(rule.Rule[time.Time])
	if err := rule.Check(context.Background(), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), r); !errors.Is(err, errCustom) {
		t.Errorf("got %v, want errCustom", err)
	}
}
