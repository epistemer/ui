package forms_test

import (
	"context"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/models/option"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

type testLink string

func (l testLink) URL() string { return string(l) }

type testLabeler struct{}

func (testLabeler) LabelOf(context.Context, string) (string, bool, error) { return "", false, nil }

// TestDefaultKinds pins interface §5.10's table of default kinds.
func TestDefaultKinds(t *testing.T) {
	f := forms.New()
	for _, tc := range []struct {
		c    forms.Control
		want forms.Kind
	}{
		{f.Text("text").Control(), forms.KindText},
		{f.Parsed[code]("parsed").Control(), forms.KindText},
		{f.Text("multi").Hint(forms.KindMultiline).Control(), forms.KindMultiline},
		{f.Text("pw").Raw().Hint(forms.KindPassword).Control(), forms.KindPassword},
		{f.Text("hid").Hint(forms.KindHidden).Control(), forms.KindHidden},
		{f.Int("int").Control(), forms.KindNumber},
		{f.Float("float").Control(), forms.KindNumber},
		{f.Number[int64]("num").Control(), forms.KindNumber},
		{f.Checkbox("box").Control(), forms.KindCheckbox},
		{f.Bool("toggle").Control(), forms.KindToggle},
		{f.Date("date").Control(), forms.KindDate},
		{f.Time("time", "15:04").Control(), forms.KindDateTime},
		{f.Enum("enum", []color{"red"}).Control(), forms.KindSelect},
		{f.Text("opts").Options(color("red")).Control(), forms.KindSelect},
		{f.Texts("tags").Options([]option.SelectOption{color("red")}...).Control(), forms.KindMultiSelect},
		{f.File("file").Control(), forms.KindFile},
		{f.Files("files").Control(), forms.KindFiles},
		{f.List("list", func(g *forms.Form) int { return 0 }).Control(), forms.KindList},
		{f.Text("rem").Remote(testLink("/s"), testLabeler{}).Control(), forms.KindRemote},
		{f.Upload("up", nil).Control(), forms.KindUpload},
		{f.Uploads("ups", nil).Control(), forms.KindUploads},
	} {
		if got := tc.c.Kind(); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.c.Path(), got, tc.want)
		}
	}
	g := f.Group("grp", func(g *forms.Form) *forms.Form { return g })
	if g.Control().Kind() != forms.KindGroup {
		t.Error("a group is KindGroup")
	}
	want := []string{"", "text", "multiline", "password", "hidden", "number", "checkbox", "toggle", "date", "datetime", "select", "multiselect", "file", "files", "group", "list", "remote", "upload", "uploads"}
	for k := forms.Kind(0); int(k) < len(want); k++ {
		if k.String() != want[k] {
			t.Errorf("Kind(%d).String() = %q, want %q", k, k.String(), want[k])
		}
	}
	if forms.KindList != 15 || forms.KindRemote != 16 || forms.KindUpload != 17 || forms.KindUploads != 18 {
		t.Errorf("kind numbers: list=%d remote=%d upload=%d uploads=%d", forms.KindList, forms.KindRemote, forms.KindUpload, forms.KindUploads)
	}
}

func TestFieldErrorAndChoice(t *testing.T) {
	f := forms.New()
	n := f.Text("n").Is(rule.MinLen(5))
	bind(f, "n=abc")
	f.Validate(bg)
	var fe *forms.FieldError
	if !errors.As(f.Err(), &fe) || fe.Path != "n" || fe.Unwrap() != n.Err() || fe.Error() != "n: must be at least 5 characters" {
		t.Errorf("FieldError %+v", fe)
	}
	var got forms.Choice
	for ch := range forms.New().Enum("c", []color{"red"}).Control().Options() {
		got = ch
	}
	if got.Value != "red" || got.Selected || got.Label == nil {
		t.Errorf("Choice %+v", got)
	}
	if forms.IndexPlaceholder != "__i__" || forms.DefaultMaxItems != 256 {
		t.Error("constants")
	}
	satisfies[color, int32]()
}

// satisfies compiles only when E is an EnumValue and N an Integer.
func satisfies[E forms.EnumValue, N forms.Integer]() {}

func TestControlTwoState(t *testing.T) {
	f := forms.New()
	box := f.Checkbox("box")
	boxToggle := f.Checkbox("box_toggle").Hint(forms.KindToggle)
	boxHidden := f.Checkbox("box_hidden").Hint(forms.KindHidden)
	tri := f.Bool("tri")
	triBox := f.Bool("tri_box").Hint(forms.KindCheckbox)
	txt := f.Text("txt")
	txtToggle := f.Text("txt_toggle").Hint(forms.KindToggle)
	grp := f.Group("grp", func(g *forms.Form) *forms.Form { return g })
	lst := f.List("lst", func(g *forms.Form) int { return 0 })

	for _, tc := range []struct {
		name string
		c    forms.Control
		want bool
	}{
		{"checkbox", box.Control(), true},
		{"checkbox hinted toggle", boxToggle.Control(), true},
		{"checkbox hinted hidden", boxHidden.Control(), true},
		{"bool", tri.Control(), false},
		{"bool hinted checkbox", triBox.Control(), false},
		{"text", txt.Control(), false},
		{"text hinted toggle", txtToggle.Control(), false},
		{"group", grp.Control(), false},
		{"list", lst.Control(), false},
		{"root", f.Control(), false},
		{"zero", forms.Control{}, false},
	} {
		if got := tc.c.TwoState(); got != tc.want {
			t.Errorf("%s: TwoState() = %v, want %v", tc.name, got, tc.want)
		}
	}
	if boxToggle.Control().Kind() != forms.KindToggle {
		t.Errorf("boxToggle Kind() = %v, want KindToggle", boxToggle.Control().Kind())
	}
}
