// Package col is the spike's stand-in for the column set models would
// generate per _view relation (server-state-design.md §6.1, S3): only what
// the schema knows.
package col

import (
	"time"

	"github.com/epistemer/i18n"
	"github.com/epistemer/models/option"
)

// Kind is a column's kind, from its SQL type.
type Kind uint8

const (
	Text Kind = iota
	Integer
	Number
	Bool
	Date
	Time
	Enum
)

// Value is a cell's plain value, nulls unwrapped: the field of its kind
// holds it, unless Null.
type Value struct {
	Null bool
	Text string
	Int  int64
	Num  float64
	Bool bool
	Time time.Time
	Enum option.SelectOption
}

// Column is one column of a view whose rows are R.
type Column[R any] struct {
	Name    string // its SQL name
	Label   i18n.Ter
	Kind    Kind
	Options []option.SelectOption // an enum's
	Get     func(R) Value
}

// Plain value constructors, as generated getters would build them.
func TextOf(s string) Value              { return Value{Text: s} }
func IntOf(n int64) Value                { return Value{Int: n} }
func TimeOf(t time.Time) Value           { return Value{Time: t, Null: t.IsZero()} }
func EnumOf(o option.SelectOption) Value { return Value{Enum: o, Null: o == nil} }
func BoolOf(b bool) Value                { return Value{Bool: b} }
