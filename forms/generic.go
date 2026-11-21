package forms

import (
	"encoding"
	"reflect"
	"strconv"
)

// Integer is the set of integer types Number accepts.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Float is the set of floating-point types Number accepts.
type Float interface{ ~float32 | ~float64 }

// Number declares a number field of type T: f.Number[int32]("depth"). A
// value out of T's range, NaN and infinities don't convert. Rules name the
// type too: rule.Min[int32](1).
func (f *Form) Number[T Integer | Float](name string) *Field[T] {
	t := reflect.TypeFor[T]()
	bits := t.Bits()
	var c codec
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		c.parse = func(in input) (any, error) {
			n, err := strconv.ParseInt(in.texts[0], 10, bits)
			return T(n), err
		}
		c.format = func(v any) []string { return []string{strconv.FormatInt(int64(v.(T)), 10)} }
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		c.parse = func(in input) (any, error) {
			n, err := strconv.ParseUint(in.texts[0], 10, bits)
			return T(n), err
		}
		c.format = func(v any) []string { return []string{strconv.FormatUint(uint64(v.(T)), 10)} }
	default:
		c.parse = func(in input) (any, error) {
			x, err := parseFloat(in.texts[0], bits)
			return T(x), err
		}
		c.format = func(v any) []string { return []string{strconv.FormatFloat(float64(v.(T)), 'g', -1, bits)} }
	}
	return newField[T](f, name, KindNumber, c)
}

// Parsed declares a field of any type with text marshalling, such as a
// decimal, without forms importing it: f.Parsed[decimal.Decimal]("price").
// An app usually wraps it once:
//
//	func Decimal(f *forms.Form, name string) *forms.Field[decimal.Decimal] {
//		return f.Parsed[decimal.Decimal](name)
//	}
func (f *Form) Parsed[T any, PT interface {
	*T
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string) *Field[T] {
	return newField[T](f, name, KindText, codec{
		parse: func(in input) (any, error) {
			var v T
			if err := PT(&v).UnmarshalText([]byte(in.texts[0])); err != nil {
				return nil, err
			}
			return v, nil
		},
		format: func(v any) []string {
			t := v.(T)
			b, err := PT(&t).MarshalText()
			if err != nil {
				return nil // shown empty rather than as Go's own printing
			}
			return []string{string(b)}
		},
	})
}
