package forms

import (
	"fmt"
	"io"
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
// (at most 10 MB, as net/http allows) or multipart (with net/http's 32 MB of
// memory before files spill to disk), without the query, which isn't even
// parsed, so a query net/http would refuse doesn't spoil a body; a GET or
// HEAD reads the query; any other method is an error. The multipart form is
// left on r, so net/http removes its temporary files after the handler
// returns. Limiting the body's size is the app's (http.MaxBytesReader).
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
		const limit = 10 << 20
		body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
		if err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		if len(body) > limit {
			return nil, fmt.Errorf("forms: an urlencoded body over %d bytes", limit)
		}
		v, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		return FromValues(v), nil
	case "multipart/form-data":
		mr, err := r.MultipartReader()
		if err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		form, err := mr.ReadForm(32 << 20)
		if err != nil {
			return nil, fmt.Errorf("forms: %w", err)
		}
		r.MultipartForm = form // net/http's server removes its files
		return &source{values: form.Value, files: form.File}, nil
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
