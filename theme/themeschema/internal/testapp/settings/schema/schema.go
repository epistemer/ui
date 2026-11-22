// Package schema is a toy shop's configuration, to check that what cfgx
// generates from themeschema's group satisfies theme.Config.
package schema

import (
	"github.com/epistemer/cfgx/dsl"

	"github.com/epistemer/ui/theme/themeschema"
)

// Theme is ui's fragment.
var Theme = themeschema.Group()

var store = dsl.Path("store", "Settings store").Default("shop.db")

// App is the toy shop.
var App = dsl.App("shop", "A toy shop", Theme, dsl.Group("storage", "Storage", store)).Store(store)

// Serve runs the shop.
var Serve = dsl.Command("Serve").Uses(Theme)
