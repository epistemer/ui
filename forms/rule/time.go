package rule

import (
	"context"
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms/internal/hook"
)

// NotBefore fails for a time before t. It reports ErrNotBefore with t's
// date, and implies min.
func NotBefore(t time.Time) Rule[time.Time] {
	return (&timeBound{t: t}).rule()
}

// NotAfter fails for a time after t. It reports ErrNotAfter with t's date,
// and implies max.
func NotAfter(t time.Time) Rule[time.Time] {
	return (&timeBound{t: t, after: true}).rule()
}

type timeBound struct {
	t        time.Time
	after    bool // NotAfter; else NotBefore
	dateOnly bool // compare calendar dates in UTC: a Date field
}

func (b *timeBound) rule() Rule[time.Time] {
	name := "min"
	if b.after {
		name = "max"
	}
	r := Rule[time.Time]{
		check: func(_ context.Context, v time.Time) error { return b.check(v) },
		time:  b,
	}
	return r.with(name, b.attr())
}

// attr is the bound as an <input> value: a date for a Date field, else a
// datetime-local value, with seconds only when the bound has them.
func (b *timeBound) attr() string {
	u := b.t.UTC()
	switch {
	case b.dateOnly:
		return u.Format(time.DateOnly)
	case u.Second() != 0 || u.Nanosecond() != 0:
		return u.Format("2006-01-02T15:04:05")
	default:
		return u.Format("2006-01-02T15:04")
	}
}

func (b *timeBound) check(v time.Time) error {
	bound := b.t.UTC()
	if b.dateOnly {
		v, bound = date(v), date(bound)
	}
	if b.after {
		if v.After(bound) {
			return ErrNotAfter.With(i18n.StaticDate(bound))
		}
		return nil
	}
	if v.Before(bound) {
		return ErrNotBefore.With(i18n.StaticDate(bound))
	}
	return nil
}

func date(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// The hook is set here, so it is set wherever forms can see a rule: forms
// imports this package.
func init() {
	hook.DateOnly = func(r any) any {
		tr, ok := r.(Rule[time.Time])
		if !ok || tr.time == nil {
			return r
		}
		b := *tr.time
		b.dateOnly = true
		d := b.rule()
		d.err = tr.err
		return d
	}
}
