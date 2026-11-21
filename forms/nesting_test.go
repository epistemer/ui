package forms_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/formstest"
	"github.com/epistemer/ui/forms/rule"
)

type pct struct {
	*forms.Form
	Value, Max *forms.Field[int]
}

func newPct(g *forms.Form) *pct {
	return &pct{Form: g, Value: g.Int("value").Required().Is(rule.Between(0, 100)), Max: g.Int("max")}
}

type constraint struct {
	*forms.Form
	Type      *forms.Field[string]
	MinAmount *forms.Field[int]
}

func newConstraint(g *forms.Form) *constraint {
	c := &constraint{Form: g}
	c.Type = g.Text("type").Required()
	c.MinAmount = g.Int("min_amount").When(c.Type.Eq("monetary")).Required()
	return c
}

type promo struct {
	*forms.Form
	Type        *forms.Field[string]
	Percentage  *pct
	Constraints *forms.List[*constraint]
}

func newPromo() *promo {
	f := &promo{Form: forms.New()}
	f.Type = f.Text("type").Required()
	f.Percentage = f.Group("percentage", newPct)
	f.Percentage.When(f.Type.Eq("percentage"))
	f.Constraints = f.List("constraints", newConstraint).MinItems(1).MaxItems(3)
	return f
}

func TestGroupBindsUnderItsPath(t *testing.T) {
	f := newPromo()
	ok := formstest.Submit(t, bg, f.Form, "type=percentage&percentage.value=150&constraints[0].type=x")
	if ok || !errors.Is(f.Percentage.Value.Err(), rule.ErrBetween) {
		t.Fatalf("valid %v, value err %v", ok, f.Percentage.Value.Err())
	}
	if got := formstest.Errors(f.Form); !slices.Equal(got, []string{"percentage.value: errors.forms.rule.between"}) {
		t.Errorf("errors %q", got)
	}
}

func TestInactiveGroupIsCleared(t *testing.T) {
	f := newPromo()
	if !formstest.Submit(t, bg, f.Form, "type=fixed&percentage.value=150&constraints[0].type=x") {
		t.Fatalf("a stale value in an inactive group failed the form: %v", formstest.Errors(f.Form))
	}
	if f.Percentage.Active() || f.Percentage.Value.Ptr() != nil {
		t.Errorf("active %v, value %v", f.Percentage.Active(), f.Percentage.Value.Value())
	}
}

func TestListSparseIndices(t *testing.T) {
	f := newPromo()
	formstest.Submit(t, bg, f.Form, "type=fixed&constraints[0].type=monetary&constraints[0].min_amount=5&constraints[3].type=monetary")
	items := f.Constraints.Items()
	if len(items) != 2 || items[0].MinAmount.Value() != 5 {
		t.Fatalf("%d items", len(items))
	}
	if got := formstest.Errors(f.Form); !slices.Equal(got, []string{"constraints[1].min_amount: errors.forms.required"}) {
		t.Errorf("errors %q: index 3 renumbers to 1", got)
	}
	if p := items[1].MinAmount.Control().Path(); p != "constraints[1].min_amount" {
		t.Errorf("path %q", p)
	}
}

func TestListLimitsAndJunk(t *testing.T) {
	f := newPromo()
	formstest.Submit(t, bg, f.Form, "type=fixed&constraints[00].type=a&constraints[x].type=b&constraints[1]junk=c&constraints[7].bogus=d")
	if len(f.Constraints.Items()) != 0 || !errors.Is(f.Constraints.Err(), forms.ErrMinItems) {
		t.Errorf("items %d, err %v: malformed indices and undeclared members make no item", len(f.Constraints.Items()), f.Constraints.Err())
	}
	g := newPromo()
	var q []string
	for _, i := range []string{"0", "1", "2", "3", "4"} {
		q = append(q, "constraints["+i+"].type=t")
	}
	formstest.Submit(t, bg, g.Form, "type=fixed&"+strings.Join(q, "&"))
	if len(g.Constraints.Items()) != 3 || !errors.Is(g.Constraints.Err(), forms.ErrMaxItems) {
		t.Errorf("items %d, err %v", len(g.Constraints.Items()), g.Constraints.Err())
	}
}

func TestDefaultMaxItems(t *testing.T) {
	f := forms.New()
	l := f.List("rows", func(g *forms.Form) *forms.Field[string] { return g.Text("v") })
	var q []string
	for i := range 257 {
		q = append(q, "rows["+itoa(i)+"].v=x")
	}
	formstest.Submit(t, bg, f, strings.Join(q, "&"))
	if len(l.Items()) != forms.DefaultMaxItems || !errors.Is(l.Err(), forms.ErrMaxItems) {
		t.Errorf("items %d, err %v: item 257 is refused", len(l.Items()), l.Err())
	}
}

func itoa(i int) string { return fmt.Sprint(i) }

func TestAddAndGroupChecks(t *testing.T) {
	f := newPromo()
	first := f.Constraints.Add()
	first.Type.Set("monetary")
	if len(f.Constraints.Items()) != 1 || first.Type.Value() != "monetary" {
		t.Fatal("Add")
	}
	g := forms.New()
	grp := g.Group("range", func(s *forms.Form) [2]*forms.Field[int] {
		lo, hi := s.Int("lo"), s.Int("hi")
		s.Check(func(context.Context) error {
			if lo.Value() > hi.Value() {
				return errTaken
			}
			return nil
		})
		return [2]*forms.Field[int]{lo, hi}
	})
	formstest.Submit(t, bg, g, "range.lo=5&range.hi=1")
	if got := formstest.Errors(g); !slices.Equal(got, []string{"range: errors.test.taken"}) {
		t.Errorf("group check: %q", got)
	}
	_ = grp
}

func TestControlTree(t *testing.T) {
	f := newPromo()
	f.Type.Label(i18nKey("test.promo.type", "Type")).Is(rule.MaxLen(20), rule.MinLen(2), rule.MaxLen(10))
	var kinds []string
	for c := range f.Controls() {
		kinds = append(kinds, c.Path()+":"+c.Kind().String())
	}
	if strings.Join(kinds, ",") != "type:text,percentage:group,constraints:list" {
		t.Errorf("controls %v", kinds)
	}
	var attrs []string
	for k, v := range f.Type.Control().Attrs() {
		attrs = append(attrs, k+"="+v)
	}
	if strings.Join(attrs, ",") != "maxlength=10,minlength=2" {
		t.Errorf("attrs %v: the later rule wins, in order of first appearance", attrs)
	}
	tpl, ok := f.Constraints.Control().Template()
	if !ok {
		t.Fatal("no template")
	}
	var tplPaths []string
	for c := range tpl.Children() {
		tplPaths = append(tplPaths, c.Path())
	}
	if strings.Join(tplPaths, ",") != "constraints[__i__].type,constraints[__i__].min_amount" {
		t.Errorf("template %v", tplPaths)
	}
	w, has := f.Percentage.Control().When()
	b, _ := json.Marshal(w)
	if !has || string(b) != `{"typ":"c","k":"type","op":"eq","v":"percentage"}` {
		t.Errorf("group When: %s", b)
	}
	if !f.Constraints.Control().Required() {
		t.Error("a list with MinItems is required")
	}
}

func TestControlBeforeBind(t *testing.T) {
	f := newPromo()
	f.Type.Set("percentage")
	if !f.Percentage.Control().Active() {
		t.Error("before Bind, conditions read initial values")
	}
	c := forms.New()
	color := c.Enum("color", []color{"red", "blue"})
	color.Set("blue")
	var sel []string
	for ch := range color.Control().Options() {
		if ch.Selected {
			sel = append(sel, ch.Value)
		}
	}
	if !slices.Equal(sel, []string{"blue"}) || !slices.Equal(color.Control().Text(), []string{"blue"}) {
		t.Errorf("selected %v text %v", sel, color.Control().Text())
	}
}

func TestRedisplayThroughControl(t *testing.T) {
	f := forms.New()
	c := f.Int("c").As(seconds)
	c.Set(90_000_000_000)
	if !slices.Equal(c.Control().Text(), []string{"90"}) {
		t.Errorf("formatted initial value %v", c.Control().Text())
	}
	g := forms.New()
	n := g.Int("n")
	formstest.Submit(t, bg, g, "n=abc")
	if !slices.Equal(n.Control().Text(), []string{"abc"}) || !errors.Is(n.Control().Err(), forms.ErrInvalid) {
		t.Errorf("redisplay %v %v", n.Control().Text(), n.Control().Err())
	}
}

func TestAssertLabelled(t *testing.T) {
	f := newPromo()
	rec := &recorder{TB: t}
	formstest.AssertLabelled(rec, f.Form)
	if !slices.Contains(rec.errs, "formstest: type has no label of its own") || !slices.Contains(rec.errs, "formstest: percentage has no label of its own") {
		t.Errorf("errors %q", rec.errs)
	}
	g := forms.New()
	g.Text("name").Label(i18nKey("test.name", "Name"))
	rec2 := &recorder{TB: t}
	formstest.AssertLabelled(rec2, g)
	if len(rec2.errs) != 0 {
		t.Errorf("a labelled form: %q", rec2.errs)
	}
}

type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(format string, a ...any) { r.errs = append(r.errs, sprintf(format, a...)) }
func (r *recorder) Helper()                        {}

func TestNestingPanics(t *testing.T) {
	f := newPromo()
	itemField := f.Constraints.Add().Type
	for name, fn := range map[string]func(){
		"Bind on a group":        func() { f.Percentage.Bind(forms.FromValues(nil)) },
		"Validate on a group":    func() { f.Percentage.Validate(bg) },
		"group reads its member": func() { f.Percentage.When(f.Percentage.Value.Filled()) },
		"into a list item":       func() { f.Text("x").When(itemField.Filled()) },
		"MinItems above max":     func() { newPromo().Constraints.MinItems(4) },
		"MaxItems below min":     func() { newPromo().Constraints.MaxItems(0) },
		"name clash":             func() { g := forms.New(); g.Text("a"); g.Group("a", newPct) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

func TestJunkIndicesDontUseMaxItems(t *testing.T) {
	f := forms.New()
	l := f.List("items", func(g *forms.Form) *forms.Field[string] { return g.Text("name") }).MaxItems(2)
	formstest.Submit(t, bg, f, "items[0].bogus=1&items[1]=bare&items[2].name=a&items[3].name=b&items[01].name=c&items[1000000].name=d")
	got := l.Items()
	if l.Err() != nil || len(got) != 2 || got[0].Value() != "a" || got[1].Value() != "b" {
		t.Errorf("err %v, %d items: junk indices must neither build items nor count toward MaxItems", l.Err(), len(got))
	}
}

func TestNestedLists(t *testing.T) {
	type inner struct{ V *forms.Field[int] }
	f := forms.New()
	outer := f.List("a", func(g *forms.Form) *forms.List[inner] {
		return g.List("b", func(h *forms.Form) inner { return inner{h.Int("c").Is(rule.Min(1))} })
	})
	formstest.Submit(t, bg, f, "a[3].b[5].c=0&a[3].b[9].c=2")
	items := outer.Items()
	if len(items) != 1 || len(items[0].Items()) != 2 {
		t.Fatalf("items %d", len(items))
	}
	if got := formstest.Errors(f); !slices.Equal(got, []string{"a[0].b[0].c: errors.forms.rule.min"}) {
		t.Errorf("errors %q", got)
	}
}

func TestBuildCannotTouchABoundForm(t *testing.T) {
	f := forms.New()
	outer := f.Text("outer")
	f.List("rows", func(g *forms.Form) *forms.Field[string] {
		outer.Required() // a build that reaches outside its item
		return g.Text("v")
	})
	defer func() {
		if recover() == nil {
			t.Error("a list's build changed a bound form without a panic")
		}
	}()
	formstest.Submit(t, bg, f, "rows[0].v=x")
}

func TestInactiveGroupDropsItsError(t *testing.T) {
	f := newPromo()
	f.Percentage.Fail(errTaken)
	if !formstest.Submit(t, bg, f.Form, "type=fixed&constraints[0].type=x") {
		t.Errorf("an inactive group's error survived: %q", formstest.Errors(f.Form))
	}
}

func TestCheckThatFailsAFieldStopsTheRest(t *testing.T) {
	f := forms.New()
	a := f.Text("a")
	ran := 0
	f.Check(func(context.Context) error { a.Fail(errTaken); return nil })
	f.Check(func(context.Context) error { ran++; return nil })
	formstest.Submit(t, bg, f, "a=x")
	if ran != 0 || f.Valid() {
		t.Errorf("a later check ran %d times after one failed a field", ran)
	}
}

func TestTemplateRowIsActiveWhileListHidden(t *testing.T) {
	f := forms.New()
	gate := f.Checkbox("gate")
	l := f.List("rows", func(g *forms.Form) *forms.Field[string] { return g.Text("v") }).When(gate.Filled())
	if l.Control().Active() {
		t.Fatal("the list should be hidden")
	}
	tpl, _ := l.Control().Template()
	for c := range tpl.Children() {
		if !c.Active() {
			t.Errorf("%s in the template row reports inactive", c.Path())
		}
	}
}

func TestAssertLabelledLooksIntoEmptyLists(t *testing.T) {
	f := forms.New()
	f.List("rows", func(g *forms.Form) *forms.Field[string] { return g.Text("v") }).Label(i18nKey("test.rows", "Rows"))
	rec := &recorder{TB: t}
	formstest.AssertLabelled(rec, f)
	if !slices.Equal(rec.errs, []string{"formstest: rows[__i__].v has no label of its own"}) {
		t.Errorf("errors %q", rec.errs)
	}
	g := forms.New()
	l := g.List("rows", func(h *forms.Form) *forms.Field[string] { return h.Text("v") }).Label(i18nKey("test.rows", "Rows"))
	l.Add()
	l.Add()
	rec2 := &recorder{TB: t}
	formstest.AssertLabelled(rec2, g)
	if len(rec2.errs) != 1 {
		t.Errorf("a list with two items reports its member %d times; want once", len(rec2.errs))
	}
}

func TestListItemsRefuseWhenAndLabel(t *testing.T) {
	for name, fn := range map[string]func(){
		"When on an item": func() {
			f := forms.New()
			gate := f.Text("gate")
			f.List("rows", func(g *forms.Form) int { g.When(gate.Filled()); return 0 }).Add()
		},
		"Label on an item": func() {
			forms.New().List("rows", func(g *forms.Form) int { g.Label(i18nKey("x.y", "Y")); return 0 }).Add()
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			fn()
		})
	}
}

func TestZeroControlAndNilItems(t *testing.T) {
	var c forms.Control
	if c.Kind() != 0 || c.Path() != "" || c.Active() || c.Err() != nil || c.Label() == nil {
		t.Error("a zero Control must answer with zero values")
	}
	if _, has := c.When(); has {
		t.Error("zero Control has a condition")
	}
	f := forms.New()
	l := f.List("rows", func(g *forms.Form) error { g.Text("v"); return nil })
	if v := l.Add(); v != nil || len(l.Items()) != 1 {
		t.Error("a nil interface item")
	}
}

func TestMinItemsAboveDefaultMax(t *testing.T) {
	forms.New().List("rows", func(g *forms.Form) int { return 0 }).MinItems(300).MaxItems(500)
}
