package forms

import (
	"fmt"
	"iter"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
)

// Source is what a form binds from: submitted values and files by name.
type Source interface {
	Values(name string) []string
	Files(name string) []*multipart.FileHeader
	Names() iter.Seq[string] // every submitted name; lists need it
}

// FromRequest reads r once. A POST, PUT or PATCH body is read, urlencoded
// or multipart (with net/http's 32 MB of memory before files spill to
// disk), without the query; a GET or HEAD reads the query; any other
// method is an error. A body of
// another content type is an error. net/http removes multipart temporary
// files after the handler returns; limiting the body's size is the app's
// (http.MaxBytesReader).
func FromRequest(r *http.Request) (Source, error) {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	case http.MethodGet, http.MethodHead:
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			return nil, fmt.Errorf("forms: query: %w", err)
		}
		return FromValues(q), nil
	default:
		return nil, fmt.Errorf("forms: a form can't be read from a %s request", r.Method)
	}
	ct, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("forms: content type %q: %w", r.Header.Get("Content-Type"), err)
	}
	switch ct {
	case "application/x-www-form-urlencoded":
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		return FromValues(r.PostForm), nil
	case "multipart/form-data":
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		return &source{values: r.MultipartForm.Value, files: r.MultipartForm.File}, nil
	}
	return nil, fmt.Errorf("forms: a form can't be read from %q", ct)
}

// FromValues returns a source of v, for tests and query-string filters.
func FromValues(v url.Values) Source { return &source{values: v} }

type source struct {
	values map[string][]string
	files  map[string][]*multipart.FileHeader
}

func (s *source) Values(name string) []string               { return s.values[name] }
func (s *source) Files(name string) []*multipart.FileHeader { return s.files[name] }

func (s *source) Names() iter.Seq[string] {
	return func(yield func(string) bool) {
		for n := range maps.Keys(s.values) {
			if !yield(n) {
				return
			}
		}
		for n := range maps.Keys(s.files) {
			if _, dup := s.values[n]; !dup && !yield(n) {
				return
			}
		}
	}
}
