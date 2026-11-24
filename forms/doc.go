// Package forms declares, binds, validates and describes HTML forms.
//
// A form is a struct that embeds *Form and holds one typed handle per
// field. A constructor declares the fields; each registers itself with the
// form, in order:
//
//	type StoreForm struct {
//		*forms.Form
//		Name *forms.Field[string]
//		Kind *forms.Field[models.StoreKind]
//	}
//
//	func NewStoreForm() *StoreForm {
//		f := &StoreForm{Form: forms.New()}
//		f.Name = f.Text("name").Label(models.L10nStore_Name).Required().Is(rule.MaxLen(255))
//		f.Kind = f.Enum("kind", models.AllStoreKindValues()).Required()
//		return f
//	}
//
// A field's handle can't share a name with a method of Form: the field
// would hide the embedded method, and the constructor's f.Text("text")
// would no longer compile. Every exported method of Form is taken (Text,
// Date, Time, File, Files, Int, Float, Bool, Label, Group, List, Check,
// Fail, Valid, Err, Errors, When, Control, among others); name the handle
// after what it holds (Body, StartsOn).
//
// A handler calls Submit, then reads typed values: f.Name.Value() is a
// string, f.Kind.Value() the enum. Errors belong to the field that reported
// them; errors.T(ctx, f.Name.Err()) is the text people see.
//
// One instance serves one request; a form is not safe for concurrent use.
// Programmer mistakes panic when the form is declared; input from a user
// never panics. forms renders nothing: ui/inputs reads each node through
// Control.
//
// The contract is docs/interface.md §5 and §9 in github.com/epistemer/ui.
package forms
