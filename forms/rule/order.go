package rule

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"strconv"
)

// Min checks that val >= v. It reports ErrMin with fmt.Sprint(v).
// It implies min when T's underlying kind is an integer or float.
func Min[T cmp.Ordered](v T) Rule[T] {
	r := Rule[T]{
		check: func(_ context.Context, val T) error {
			if val < v {
				return ErrMin.With(fmt.Sprint(v))
			}
			return nil
		},
	}
	if s, ok := formatBound(v); ok {
		r = r.with("min", s)
	}
	return r
}

// Max checks that val <= v. It reports ErrMax with fmt.Sprint(v).
// It implies max when T's underlying kind is an integer or float.
func Max[T cmp.Ordered](v T) Rule[T] {
	r := Rule[T]{
		check: func(_ context.Context, val T) error {
			if val > v {
				return ErrMax.With(fmt.Sprint(v))
			}
			return nil
		},
	}
	if s, ok := formatBound(v); ok {
		r = r.with("max", s)
	}
	return r
}

// Between checks that lo <= val <= hi. It panics if lo > hi.
// It reports ErrBetween with fmt.Sprint(lo) and fmt.Sprint(hi).
// It implies min and max when T's underlying kind is an integer or float.
func Between[T cmp.Ordered](lo, hi T) Rule[T] {
	if lo > hi {
		panic(fmt.Sprintf("rule: Between: lo (%v) > hi (%v)", lo, hi))
	}
	r := Rule[T]{
		check: func(_ context.Context, val T) error {
			if val < lo || val > hi {
				return ErrBetween.With(fmt.Sprint(lo), fmt.Sprint(hi))
			}
			return nil
		},
	}
	if sLo, ok := formatBound(lo); ok {
		if sHi, ok := formatBound(hi); ok {
			r = r.with("min", sLo).with("max", sHi)
		}
	}
	return r
}

// AtLeast checks that val.Cmp(v) >= 0. It reports ErrMin with fmt.Sprint(v).
func AtLeast[T interface{ Cmp(T) int }](v T) Rule[T] {
	return Rule[T]{
		check: func(_ context.Context, val T) error {
			if val.Cmp(v) < 0 {
				return ErrMin.With(fmt.Sprint(v))
			}
			return nil
		},
	}
}

// AtMost checks that val.Cmp(v) <= 0. It reports ErrMax with fmt.Sprint(v).
func AtMost[T interface{ Cmp(T) int }](v T) Rule[T] {
	return Rule[T]{
		check: func(_ context.Context, val T) error {
			if val.Cmp(v) > 0 {
				return ErrMax.With(fmt.Sprint(v))
			}
			return nil
		},
	}
}

// OneOf checks that val is equal to one of vs.
func OneOf[T comparable](vs ...T) Rule[T] {
	allowed := slices.Clone(vs)
	return Rule[T]{
		check: func(_ context.Context, val T) error {
			if !slices.Contains(allowed, val) {
				return ErrOneOf
			}
			return nil
		},
	}
}

func formatBound[T any](v T) (string, bool) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), true
	case reflect.Float32:
		return strconv.FormatFloat(val.Float(), 'g', -1, 32), true
	case reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, 64), true
	default:
		return "", false
	}
}
