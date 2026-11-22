package forms

import (
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"slices"
	"strconv"
	"time"

	"github.com/epistemer/models/option"
	"github.com/epistemer/ui/forms/files"
)

// Text declares a text field. White space around the value is trimmed
// unless Raw is called; an empty value is absent.
func (f *Form) Text(name string) *Field[string] {
	return newField[string](f, name, KindText, codec{
		text:   true,
		parse:  func(in input) (any, error) { return in.texts[0], nil },
		format: func(v any) []string { return []string{v.(string)} },
	})
}

// Texts declares a field of a repeated name, such as a group of checkboxes
// or a multiple select. Empty items are dropped; with none, it is absent.
func (f *Form) Texts(name string) *Field[[]string] {
	return newField[[]string](f, name, KindText, codec{
		multi:  true,
		parse:  func(in input) (any, error) { return append([]string(nil), in.texts...), nil },
		format: func(v any) []string { return append([]string(nil), v.([]string)...) },
	})
}

// Int declares an integer field. It is Field[int] so that rule.Min(0)
// infers int; for other widths use Number.
func (f *Form) Int(name string) *Field[int] {
	return newField[int](f, name, KindNumber, codec{
		integer: true,
		parse:   func(in input) (any, error) { return strconv.Atoi(in.texts[0]) },
		format:  func(v any) []string { return []string{strconv.Itoa(v.(int))} },
	})
}

// Float declares a float64 field. NaN and infinities don't convert.
func (f *Form) Float(name string) *Field[float64] {
	return newField[float64](f, name, KindNumber, codec{
		parse:  func(in input) (any, error) { return parseFloat(in.texts[0], 64) },
		format: func(v any) []string { return []string{strconv.FormatFloat(v.(float64), 'g', -1, 64)} },
	})
}

var errNotFinite = errors.New("not a finite number")

func parseFloat(s string, bits int) (float64, error) {
	v, err := strconv.ParseFloat(s, bits)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, errNotFinite
	}
	return v, nil
}

// Bool declares a three-state boolean: absent, true or false. It accepts
// what strconv.ParseBool accepts, and "on".
func (f *Form) Bool(name string) *Field[bool] {
	return newField[bool](f, name, KindToggle, codec{
		parse: func(in input) (any, error) {
			if in.texts[0] == "on" {
				return true, nil
			}
			return strconv.ParseBool(in.texts[0])
		},
		format: func(v any) []string { return []string{strconv.FormatBool(v.(bool))} },
	})
}

// Checkbox declares a two-state boolean that is never absent: true when its
// last value is non-empty and not one strconv.ParseBool reads as false.
// Required means it must be ticked.
func (f *Form) Checkbox(name string) *Field[bool] {
	h := newField[bool](f, name, KindCheckbox, codec{
		always: true,
		box:    true,
		parse: func(in input) (any, error) {
			if len(in.texts) == 0 {
				return false, nil
			}
			b, err := strconv.ParseBool(in.texts[0])
			return err != nil || b, nil
		},
		format: func(v any) []string { return []string{strconv.FormatBool(v.(bool))} },
	})
	// Never absent: unticked until Set or Bind says otherwise, before Bind
	// too, as a browser reads an unchecked box.
	h.f.present, h.f.vals, h.f.text = true, []any{false}, []string{"false"}
	return h
}

// Time declares a time field parsed with layout, in UTC.
func (f *Form) Time(name, layout string) *Field[time.Time] {
	if layout == "" {
		panic(fmt.Sprintf("forms: Time(%q, \"\"): a layout is needed", name))
	}
	return newField[time.Time](f, name, KindDateTime, timeCodec(layout))
}

// Date declares a date field, layout 2006-01-02, in UTC. Its time rules
// (rule.NotBefore, rule.NotAfter) compare calendar dates.
func (f *Form) Date(name string) *Field[time.Time] {
	h := newField[time.Time](f, name, KindDate, timeCodec(time.DateOnly))
	h.f.dateOnly = true
	return h
}

func timeCodec(layout string) codec {
	return codec{
		parse:  func(in input) (any, error) { return time.ParseInLocation(layout, in.texts[0], time.UTC) },
		format: func(v any) []string { return []string{v.(time.Time).UTC().Format(layout)} },
	}
}

// File declares a single file upload; a part with no file name is no file.
func (f *Form) File(name string) *Field[*multipart.FileHeader] {
	return newField[*multipart.FileHeader](f, name, KindFile, codec{
		files:  true,
		parse:  func(in input) (any, error) { return in.files[0], nil },
		format: func(any) []string { return nil },
	})
}

// Files declares a multiple file upload; parts with no file name are
// dropped.
func (f *Form) Files(name string) *Field[[]*multipart.FileHeader] {
	return newField[[]*multipart.FileHeader](f, name, KindFiles, codec{
		files:  true,
		multi:  true,
		parse:  func(in input) (any, error) { return append([]*multipart.FileHeader(nil), in.files...), nil },
		format: func(any) []string { return nil },
	})
}

// Upload declares a single-file upload field (KindUpload) with the file
// the record already has (at most one). Kept files post as <path>.keep=<key>
// and a new file under <path>; an added file replaces the stored one in the
// value while the control still reports the posted keep state. It panics in
// a list item, with len(stored) > 1, or on an empty or repeated Key or an
// empty Name.
func (f *Form) Upload(name string, stored []files.Stored) *Field[files.Upload] {
	if len(stored) > 1 {
		panic(fmt.Sprintf("forms: Upload %q takes at most one stored file, got %d; use Uploads", name, len(stored)))
	}
	return f.newUpload("Upload", name, KindUpload, false, stored)
}

// Uploads declares a multiple-file upload field (KindUploads) with the
// files the record already has. Kept files post as <path>.keep=<key> and
// new files under <path>. It panics in a list item, or on an empty or
// repeated Key or an empty Name.
func (f *Form) Uploads(name string, stored []files.Stored) *Field[files.Upload] {
	return f.newUpload("Uploads", name, KindUploads, true, stored)
}

func (f *Form) newUpload(what, name string, kind Kind, multi bool, stored []files.Stored) *Field[files.Upload] {
	if f.inListItem() {
		panic(fmt.Sprintf("forms: %s %q inside a list item; use File or Files", what, name))
	}
	seen := make(map[string]bool, len(stored))
	for i, s := range stored {
		if s.Key == "" {
			panic(fmt.Sprintf("forms: %s %q: stored file %d has an empty Key", what, name, i))
		}
		if s.Name == "" {
			panic(fmt.Sprintf("forms: %s %q: stored file %q has an empty Name", what, name, s.Key))
		}
		if seen[s.Key] {
			panic(fmt.Sprintf("forms: %s %q lists the stored file key %q twice", what, name, s.Key))
		}
		seen[s.Key] = true
	}
	h := newField[files.Upload](f, name, kind, codec{
		files:  true,
		upload: true,
		multi:  multi,
		format: func(any) []string { return nil },
	})
	fd := h.f
	fd.stored = slices.Clone(stored)
	if len(fd.stored) > 0 {
		fd.present = true
		fd.vals = []any{files.Upload{Kept: slices.Clone(fd.stored)}}
	}
	return h
}

// EnumValue is what Enum takes: a string type whose values are choices, as
// every enum models generates is.
type EnumValue interface {
	~string
	option.SelectOption
}

// Enum declares a field whose value is one of values, compared by Value().
// It panics on no values or two with one Value().
func (f *Form) Enum[E EnumValue](name string, values []E) *Field[E] {
	if len(values) == 0 {
		panic(fmt.Sprintf("forms: Enum %q has no values", name))
	}
	opts := make([]option.SelectOption, len(values))
	seen := map[string]bool{}
	for i, v := range values {
		if seen[v.Value()] {
			panic(fmt.Sprintf("forms: Enum %q lists the value %q twice", name, v.Value()))
		}
		seen[v.Value()] = true
		opts[i] = v
	}
	h := newField[E](f, name, KindSelect, codec{
		format: func(v any) []string { return []string{v.(E).Value()} },
	})
	fd := h.f
	// Parse against the choices as they stand at Bind, so Options can
	// replace them; a value outside them never reaches a rule. An enum's
	// value is its text: E is a string type.
	fd.base.parse = func(in input) (any, error) {
		if !fd.isOption(in.texts[0]) {
			return nil, ErrNotAnOption
		}
		return E(in.texts[0]), nil
	}
	fd.options, fd.optionsSet = opts, true
	return h
}
