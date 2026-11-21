package forms_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/formstest"
)

func TestDeepJunkKeys(t *testing.T) {
	f := forms.New()
	l := f.List("tags", func(g *forms.Form) *forms.Field[string] { return g.Text("name") }).MaxItems(2)
	formstest.Submit(t, bg, f, "tags[0].name.junk=1&tags[1].name[0]=1&tags[2].name=a&tags[3].name=b")
	if l.Err() != nil || len(l.Items()) != 2 || l.Items()[0].Value() != "a" {
		t.Errorf("err %v, %d items: names past a declared field are junk", l.Err(), len(l.Items()))
	}
}

func TestPostWithAQueryNetHTTPRefuses(t *testing.T) {
	f := forms.New()
	name := f.Text("name")
	r := httptest.NewRequest(http.MethodPost, "/save?utm=a;b&bad=%ZZ", strings.NewReader("name=Ada"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !f.Submit(bg, r) || name.Value() != "Ada" {
		t.Errorf("valid %v, name %q, %v", f.Valid(), name.Value(), f.Err())
	}
}

func TestCheckboxBeforeBind(t *testing.T) {
	f := forms.New()
	box := f.Checkbox("agree")
	hint := f.Text("hint").When(box.Eq(false))
	if !hint.Control().Active() {
		t.Error("before Bind an unticked checkbox is Eq(false), as a browser reads it")
	}
	if strings.Join(box.Control().Text(), "") != "false" {
		t.Errorf("text %v", box.Control().Text())
	}
}

func TestUnreadableSubmitLeavesSubFormsInvalid(t *testing.T) {
	f := newPromo()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	f.Submit(bg, r)
	if f.Valid() || f.Percentage.Valid() {
		t.Errorf("root %v, group %v: nothing was validated", f.Valid(), f.Percentage.Valid())
	}
}

type relabelled struct{ v string }

func (r relabelled) Value() string    { return r.v }
func (r relabelled) Label() i18n.TKer { return labels["red"] }

func TestEnumWithForeignOptions(t *testing.T) {
	f := forms.New()
	c := f.Enum("c", []color{"red", "blue"}).Options(relabelled{"red"})
	formstest.Submit(t, bg, f, "c=red")
	if c.Err() != nil || c.Value() != "red" {
		t.Errorf("value %q, err %v", c.Value(), c.Err())
	}
}

func TestOptionsKind(t *testing.T) {
	f := forms.New()
	if k := f.Date("d").Options(relabelled{"2026-11-21"}).Control().Kind(); k != forms.KindSelect {
		t.Errorf("a date with options: %v", k)
	}
	if k := f.Text("t").Hint(forms.KindHidden).Options(relabelled{"x"}).Control().Kind(); k != forms.KindHidden {
		t.Errorf("Hint must win over Options: %v", k)
	}
}

func TestRootWhenIsANoOp(t *testing.T) {
	f := newPromo()
	f.Form.When(f.Type.Filled())
	if _, has := f.Control().When(); has || !f.Active() {
		t.Error("When on the root must have no effect")
	}
}
