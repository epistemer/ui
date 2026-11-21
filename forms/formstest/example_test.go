package formstest_test

import (
	"context"
	"fmt"
	"net/url"

	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/formstest"
	"github.com/epistemer/ui/forms/rule"
)

func ExampleErrors() {
	f := forms.New()
	f.Text("title").Required()
	f.Int("age").Is(rule.Min(18))

	f.Bind(forms.FromValues(url.Values{
		"age": []string{"15"},
	}))
	f.Validate(context.Background())

	for _, errStr := range formstest.Errors(f) {
		fmt.Println(errStr)
	}

	// Output:
	// title: errors.forms.required
	// age: errors.forms.rule.min
}
