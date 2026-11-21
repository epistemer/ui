package rule

import (
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms/internal/hook"
)

// NotBefore fails for a time before t. It reports ErrNotBefore with t as a
// date, and implies min.
func NotBefore(t time.Time) Rule[time.Time] {
	return Rule[time.Time]{time: &timeBound{t: t}}.with("min", t.UTC().Format(dateTimeLocal))
}

// NotAfter fails for a time after t. It reports ErrNotAfter with t as a
// date, and implies max.
func NotAfter(t time.Time) Rule[time.Time] {
	return Rule[time.Time]{time: &timeBound{t: t, after: true}}.with("max", t.UTC().Format(dateTimeLocal))
}

// dateTimeLocal is the value format of <input type="datetime-local">.
// forms rewrites it to a date for a Date field.
const dateTimeLocal = "2006-01-02T15:04"

type timeBound struct {
	t        time.Time
	after    bool // NotAfter; else NotBefore
	dateOnly bool
}

func (b *timeBound) check(v time.Time) error {
	bound := b.t
	if b.dateOnly {
		v, bound = date(v), date(bound)
	}
	if b.after {
		if v.After(bound) {
			return ErrNotAfter.With(i18n.StaticDate(b.t))
		}
		return nil
	}
	if v.Before(bound) {
		return ErrNotBefore.With(i18n.StaticDate(b.t))
	}
	return nil
}

func date(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func init() {
	hook.DateOnly = func(r any) any {
		tr, ok := r.(Rule[time.Time])
		if !ok || tr.time == nil {
			return r
		}
		b := *tr.time
		b.dateOnly = true
		tr.time = &b
		return tr
	}
}
