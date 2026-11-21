package stores

import (
	"context"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/epistemer/ui/examples/shop/models"
)

// Service is what the pages need from the shop's service layer.
type Service interface {
	CreateStore(ctx context.Context, name string, desc *string, address string, kind models.StoreKind, logo *multipart.FileHeader) (models.Store, error)
	GetStore(ctx context.Context, id int64) (models.Store, error)
}

// Render shows a page; the app supplies it (templ in a real app).
type Render func(w http.ResponseWriter, r *http.Request, status int, f *StoreForm) error

// H holds the pages' dependencies.
type H struct {
	S      Service
	Render Render
}

// StoreCreate handles the create form's POST. The logo is mandatory here.
func (h *H) StoreCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	f := NewStoreForm()
	f.Logo.Required()
	if !f.Submit(ctx, r) {
		return h.Render(w, r, http.StatusUnprocessableEntity, f)
	}
	st, err := h.S.CreateStore(ctx, f.Name.Value(), f.Description.Ptr(), f.Address.Value(), f.Kind.Value(), f.Logo.Value())
	if err != nil {
		f.Fail(err)
		return h.Render(w, r, http.StatusUnprocessableEntity, f)
	}
	http.Redirect(w, r, "/admin/stores/"+strconv.FormatInt(st.ID, 10), http.StatusSeeOther)
	return nil
}

// StoreEdit shows the same form, filled from the model. The logo stays
// optional.
func (h *H) StoreEdit(w http.ResponseWriter, r *http.Request, id int64) error {
	st, err := h.S.GetStore(r.Context(), id)
	if err != nil {
		return err
	}
	f := NewStoreForm()
	f.Fill(st)
	return h.Render(w, r, http.StatusOK, f)
}
