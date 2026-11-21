// Package hook lets forms reach behaviour of its subpackages that is not
// part of their public API.
package hook

// DateOnly, set by package rule, returns a copy of a rule.Rule[time.Time]
// that compares calendar dates in UTC rather than instants, and leaves any
// other value unchanged. forms applies it to the rules of a Date field
// (interface §9.4).
var DateOnly func(r any) any
