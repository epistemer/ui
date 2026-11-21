// want: cannot use rule.Min(0) (value of struct type rule.Rule[int]) as rule.Rule[int32] value
package main

import "github.com/epistemer/ui/forms/rule"

// A bound's type comes from its argument, not from the field's.
var _ rule.Rule[int32] = rule.Min(0)

func main() {}
