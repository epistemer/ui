// Package themehook lets ui/theme hand what isn't its API to ui's own
// packages: the joined stylesheet to catalogtest, the derived run-time
// tokens to the catalogue's token panel, and the switch for ui's own
// reverse selector check (visual-interface.md §1, §9.1). Not API.
package themehook

var (
	// Stylesheet returns ui's joined stylesheet. ui/theme sets it in init.
	Stylesheet func() []byte

	// Tokens returns the run-time tokens for the values given by key
	// (themeschema's key names), derived as the theme stylesheet derives
	// them, in the order it declares them. ui/theme sets it in init.
	Tokens func(values map[string]string) [][2]string

	// ReverseCheck makes catalogtest fail for every ui rule no story
	// renders. Only ui's own catalogue test sets it.
	ReverseCheck bool
)
