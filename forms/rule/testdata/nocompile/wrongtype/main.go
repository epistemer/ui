// want: cannot use rule.MaxLen(5) (value of struct type rule.Rule[string]) as rule.Rule[bool] value
package main

import "github.com/epistemer/ui/forms/rule"

var _ rule.Rule[bool] = rule.MaxLen(5)

func main() {}
