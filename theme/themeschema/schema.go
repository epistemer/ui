// Package themeschema declares ui/theme's settings for a cfgx app, so every
// app declares them the same way (visual-interface.md §3.4):
//
//	var Theme = themeschema.Group()
//	var Serve = dsl.Command("Serve").Uses(Theme /* , … */)
//
// The generated settings then satisfy theme.Config. Every key is mutable,
// so an admin changes the theme in the running app; the app's cfgx App
// needs a store. The group is pinned with GoName("Theme"), and no key name
// holds an initialism, so the getter names don't depend on the app's
// Initialisms. Defaults come from theme.Defaults and checks from theme's
// Check functions, so the token table stays the one source.
//
// It is a module of its own, so cfgx never enters ui's go.mod.
package themeschema

import (
	"github.com/epistemer/cfgx/dsl"

	"github.com/epistemer/ui/theme"
)

// Group returns the cfgx group "theme".
func Group() *dsl.GroupDef {
	d := theme.Defaults{}
	color := func(name, desc, def string) dsl.Member {
		return dsl.String(name, desc).Default(def).Validate(theme.CheckColor).Mutable()
	}
	return dsl.Group("theme", "Theme",
		color("brand_color", "Brand colour", d.ThemeBrandColor()),
		color("danger_color", "Danger colour", d.ThemeDangerColor()),
		color("warning_color", "Warning colour", d.ThemeWarningColor()),
		color("success_color", "Success colour", d.ThemeSuccessColor()),
		color("info_color", "Information colour", d.ThemeInfoColor()),
		dsl.String("font_sans", "Text font").Default(d.ThemeFontSans()).Validate(theme.CheckFont).Mutable(),
		dsl.String("font_mono", "Code font").Default(d.ThemeFontMono()).Validate(theme.CheckFont).Mutable(),
		dsl.String("radius", "Corner radius").Default(d.ThemeRadius()).Validate(theme.CheckRadius).Mutable(),
	).GoName("Theme")
}
