package forms

import (
	"encoding"
	"fmt"
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
	return newField[T](f, name, KindNumber, codec{
		parse: func(in input) (any, error) {
			var v T
			rv := reflect.ValueOf(&v).Elem()
			s := in.texts[0]
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(s, 10, rv.Type().Bits())
				if err != nil {
					return nil, err
				}
				rv.SetInt(n)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n, err := strconv.ParseUint(s, 10, rv.Type().Bits())
				if err != nil {
					return nil, err
				}
				rv.SetUint(n)
			default:
				x, err := parseFloat(s, rv.Type().Bits())
				if err != nil {
					return nil, err
				}
				rv.SetFloat(x)
			}
			return v, nil
		},
		format: func(v any) []string {
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return []string{strconv.FormatInt(rv.Int(), 10)}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				return []string{strconv.FormatUint(rv.Uint(), 10)}
			default:
				return []string{strconv.FormatFloat(rv.Float(), 'g', -1, rv.Type().Bits())}
			}
		},
	})
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
				return []string{fmt.Sprint(v)}
			}
			return []string{string(b)}
		},
	})
}
