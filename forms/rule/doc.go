// Package rule holds the typed validation rules of github.com/epistemer/ui/forms.
//
// A rule is a value of type Rule[T]. A field of type T accepts only
// Rule[T], so a rule for the wrong type does not compile:
//
//	f.Text("name").Is(rule.MinLen(3), rule.MaxLen(255))
//	f.Bool("active").Is(rule.MaxLen(5)) // does not compile
//
// Every catalogue rule reports a declared error of its own (ErrMaxLen, …),
// translatable through github.com/epistemer/i18n; Err replaces it. Rules
// never see an absent value: forms skips them for a field that wasn't
// submitted.
//
// The package does not import forms: Check runs rules on any value.
package rule
