package stores_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/i18n/i18ntest"
	"github.com/epistemer/ui/examples/shop/models"
	"github.com/epistemer/ui/examples/shop/stores"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

var errNameTaken = errors.Key("errors.shop.store_name_taken", "a store already has this name", errors.Conflict)

type fakeService struct {
	created models.Store
	fail    error
}

func (s *fakeService) CreateStore(_ context.Context, name string, desc *string, address string, kind models.StoreKind, logo *multipart.FileHeader) (models.Store, error) {
	if s.fail != nil {
		return models.Store{}, s.fail
	}
	s.created = models.Store{ID: 7, Name: name, Description: desc, Address: address, Kind: kind, LogoURL: logo.Filename}
	return s.created, nil
}

func (s *fakeService) GetStore(context.Context, int64) (models.Store, error) {
	desc := "by the sea"
	return models.Store{ID: 7, Name: "Harbour", Address: "1 Quay", Kind: models.StoreKindKiosk, Description: &desc}, nil
}

type rendered struct {
	status int
	form   *stores.StoreForm
}

func setup() (*stores.H, *fakeService, *rendered) {
	s, out := &fakeService{}, &rendered{}
	h := &stores.H{S: s, Render: func(w http.ResponseWriter, _ *http.Request, status int, f *stores.StoreForm) error {
		out.status, out.form = status, f
		w.WriteHeader(status)
		return nil
	}}
	return h, s, out
}

func post(fields map[string]string, logo bool) *http.Request {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	if logo {
		fw, _ := w.CreateFormFile("logo", "logo.png")
		fw.Write([]byte("\x89PNG\r\n\x1a\n"))
	}
	w.Close()
	r := httptest.NewRequest(http.MethodPost, "/admin/stores", &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

func TestCreate(t *testing.T) {
	h, s, _ := setup()
	rec := httptest.NewRecorder()
	err := h.StoreCreate(rec, post(map[string]string{"name": " Harbour ", "address": "1 Quay", "kind": "kiosk", "description": ""}, true))
	if err != nil || rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/stores/7" {
		t.Fatalf("err %v, status %d, location %q", err, rec.Code, rec.Header().Get("Location"))
	}
	if s.created.Name != "Harbour" || s.created.Description != nil || s.created.Kind != models.StoreKindKiosk || s.created.LogoURL != "logo.png" {
		t.Errorf("created %+v", s.created)
	}
}

func TestCreateInvalid(t *testing.T) {
	h, _, out := setup()
	h.StoreCreate(httptest.NewRecorder(), post(map[string]string{"name": "Ha", "kind": "castle"}, false))
	if out.status != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", out.status)
	}
	f := out.form
	for _, tc := range []struct {
		err  error
		want error
	}{
		{f.Name.Err(), rule.ErrMinLen},
		{f.Address.Err(), forms.ErrRequired},
		{f.Kind.Err(), forms.ErrNotAnOption},
		{f.Logo.Err(), forms.ErrRequired},
		{f.Description.Err(), nil},
	} {
		if tc.want == nil && tc.err != nil || tc.want != nil && !errors.Is(tc.err, tc.want) {
			t.Errorf("got %v, want %v", tc.err, tc.want)
		}
	}
}

func TestCreateServiceError(t *testing.T) {
	h, s, out := setup()
	s.fail = errNameTaken
	h.StoreCreate(httptest.NewRecorder(), post(map[string]string{"name": "Harbour", "address": "1 Quay", "kind": "shop"}, true))
	if out.form.Valid() || !errors.Is(out.form.Err(), errNameTaken) {
		t.Errorf("valid %v, err %v", out.form.Valid(), out.form.Err())
	}
}

func TestEdit(t *testing.T) {
	h, _, out := setup()
	if err := h.StoreEdit(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/admin/stores/7/edit", nil), 7); err != nil {
		t.Fatal(err)
	}
	f := out.form
	if f.Name.Value() != "Harbour" || f.Kind.Value() != models.StoreKindKiosk || *f.Description.Ptr() != "by the sea" || f.Logo.Ptr() != nil {
		t.Errorf("filled %q %q %v", f.Name.Value(), f.Kind.Value(), f.Logo.Ptr())
	}
}

func TestErrorsInTwoLocales(t *testing.T) {
	tr := i18ntest.New(t, i18n.Locale{Code: "en", NativeName: "English"})
	if err := tr.CreateLocale(i18n.Locale{Code: "ar", NativeName: "العربية"}); err != nil {
		t.Fatal(err)
	}
	if err := tr.Set("ar", "errors.forms.required", "مطلوب"); err != nil {
		t.Fatal(err)
	}
	f := stores.NewStoreForm()
	f.Bind(forms.FromValues(nil))
	f.Validate(context.Background())
	en, ar := i18ntest.Context(t, tr, "en"), i18ntest.Context(t, tr, "ar")
	if got := errors.T(en, f.Name.Err()); got != "is required" {
		t.Errorf("en: %q", got)
	}
	if got := errors.T(ar, f.Name.Err()); got != "مطلوب" {
		t.Errorf("ar: %q", got)
	}
	var all []string
	for p := range errors.All(f.Err()) {
		all = append(all, p.K())
	}
	if strings.Join(all, ",") != "errors.forms.required,errors.forms.required,errors.forms.required" {
		t.Errorf("All: %v", all)
	}
}
