package forms_test

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"time"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/rule"
)

type userRole string

const (
	roleAdmin  userRole = "admin"
	roleMember userRole = "member"
)

var (
	roleAdminLabel  = i18n.Key("role.admin", "admin")
	roleMemberLabel = i18n.Key("role.member", "member")
)

func (r userRole) Value() string { return string(r) }
func (r userRole) Label() i18n.TKer {
	if r == roleAdmin {
		return roleAdminLabel
	}
	return roleMemberLabel
}

type UserForm struct {
	*forms.Form
	Name  *forms.Field[string]
	Role  *forms.Field[userRole]
	Score *forms.Field[int]
}

func NewUserForm() *UserForm {
	f := &UserForm{Form: forms.New()}
	f.Name = f.Text("name").Required().Is(rule.MinLen(3))
	f.Role = f.Enum("role", []userRole{roleAdmin, roleMember}).Required()
	f.Score = f.Int("score").Required().Is(rule.Min(0))
	return f
}

func ExampleForm() {
	f := NewUserForm()
	f.Bind(forms.FromValues(url.Values{
		"name":  []string{"Bo"},
		"role":  []string{"admin"},
		"score": []string{"10"},
	}))

	ctx := context.Background()
	if !f.Validate(ctx) {
		fmt.Printf("valid: %v\n", f.Valid())
		fmt.Printf("role: %s\n", f.Role.Value())
		fmt.Printf("score: %d\n", f.Score.Value())
		for fe := range f.Errors() {
			fmt.Printf("%s: %s\n", fe.Path, errors.T(ctx, fe.Err))
		}
	}

	// Output:
	// valid: false
	// role: admin
	// score: 10
	// name: must be at least 3 characters
}

func ExampleField_When() {
	f := forms.New()
	subscribe := f.Checkbox("subscribe")
	email := f.Text("email").When(subscribe.Filled()).Required()

	// email was submitted, but subscribe was unticked (false).
	f.Bind(forms.FromValues(url.Values{
		"subscribe": []string{"false"},
		"email":     []string{"stale@example.com"},
	}))

	ctx := context.Background()
	f.Validate(ctx)

	fmt.Printf("valid: %v\n", f.Valid())
	fmt.Printf("active: %v\n", email.Control().Active())
	fmt.Printf("email: %q\n", email.Value())
	fmt.Printf("err: %v\n", email.Err())

	// Output:
	// valid: true
	// active: false
	// email: ""
	// err: <nil>
}

type ListItemForm struct {
	*forms.Form
	Name *forms.Field[string]
	Qty  *forms.Field[int]
}

func newListItemForm(g *forms.Form) *ListItemForm {
	return &ListItemForm{
		Form: g,
		Name: g.Text("name").Required(),
		Qty:  g.Int("qty").Required().Is(rule.Min(1)),
	}
}

func ExampleForm_List() {
	f := forms.New()
	items := f.List("items", newListItemForm)

	// Sparse indices 2 and 7 submitted from dynamic inputs:
	f.Bind(forms.FromValues(url.Values{
		"items[2].name": []string{"Pen"},
		"items[2].qty":  []string{"3"},
		"items[7].name": []string{"Notebook"},
		"items[7].qty":  []string{"0"},
	}))

	ctx := context.Background()
	f.Validate(ctx)

	for i, it := range items.Items() {
		fmt.Printf("item %d: %s\n", i, it.Name.Value())
	}
	for fe := range f.Errors() {
		fmt.Printf("%s: %s\n", fe.Path, errors.T(ctx, fe.Err))
	}

	// Output:
	// item 0: Pen
	// item 1: Notebook
	// items[1].qty: must be at least 1
}

func ExampleField_As() {
	seconds := forms.Conv[int, time.Duration]{
		Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Second, nil },
		Format: func(d time.Duration) int { return int(d / time.Second) },
	}

	f := forms.New()
	cooldown := f.Int("cooldown").Is(rule.Min(0)).As(seconds)

	f.Bind(forms.FromValues(url.Values{
		"cooldown": []string{"90"},
	}))

	f.Validate(context.Background())
	fmt.Printf("cooldown: %s\n", cooldown.Value())

	// Output:
	// cooldown: 1m30s
}

func ExampleCond_MarshalJSON() {
	f := forms.New()
	roleField := f.Text("role")
	cond := roleField.Eq("admin")

	wire, err := json.Marshal(cond)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(wire))

	// Output:
	// {"typ":"c","k":"role","op":"eq","v":"admin"}
}

type AuthorForm struct {
	*forms.Form
	Name *forms.Field[string]
	Bio  *forms.Field[string]
}

func newAuthorForm(g *forms.Form) *AuthorForm {
	return &AuthorForm{
		Form: g,
		Name: g.Text("name").Required(),
		Bio:  g.Text("bio"),
	}
}

func ExampleControl() {
	f := forms.New()
	f.Text("title").Required()
	f.Checkbox("agree").Required()
	f.Group("author", newAuthorForm)

	var walk func(iter.Seq[forms.Control])
	walk = func(seq iter.Seq[forms.Control]) {
		for c := range seq {
			fmt.Printf("%s: kind=%s required=%v\n", c.Path(), c.Kind(), c.Required())
			walk(c.Children())
		}
	}

	walk(f.Controls())

	// Output:
	// title: kind=text required=true
	// agree: kind=checkbox required=true
	// author: kind=group required=false
	// author.name: kind=text required=true
	// author.bio: kind=text required=false
}
