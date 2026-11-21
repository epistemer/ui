package main

import "github.com/epistemer/ui/forms/rule"

var _ rule.Rule[bool] = rule.MaxLen(5)

func main() {}
