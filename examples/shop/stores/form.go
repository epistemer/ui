// Package stores is the shop's admin pages for stores: one form serves
// create and edit (ui/forms design §11.1).
package stores

import (
	"mime/multipart"

	"github.com/epistemer/ui/examples/shop/models"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

// StoreForm is the store form.
type StoreForm struct {
	*forms.Form
	Name        *forms.Field[string]
	Address     *forms.Field[string]
	Kind        *forms.Field[models.StoreKind]
	Description *forms.Field[string]
	Logo        *forms.Field[*multipart.FileHeader]
}

// NewStoreForm declares the store form.
func NewStoreForm() *StoreForm {
	f := &StoreForm{Form: forms.New()}
	f.Name = f.Text("name").
		Label(models.L10nStore_Name).
		Required().
		Is(rule.MinLen(3), rule.MaxLen(255))
	f.Address = f.Text("address").
		Label(models.L10nStore_Address).
		Required().
		Is(rule.MaxLen(255))
	f.Kind = f.Enum("kind", models.AllStoreKindValues()).
		Label(models.L10nStore_Kind).
		Required()
	f.Description = f.Text("description").
		Label(models.L10nStore_Description).
		Hint(forms.KindMultiline).
		Is(rule.MaxLen(255))
	f.Logo = f.File("logo").
		Label(models.L10nStore_Logo).
		Is(rule.MaxSize(5 << 20))
	return f
}

// Fill gives the form a store's values, for the edit page.
func (f *StoreForm) Fill(st models.Store) {
	f.Name.Set(st.Name)
	f.Address.Set(st.Address)
	f.Kind.Set(st.Kind)
	f.Description.SetPtr(st.Description)
}
