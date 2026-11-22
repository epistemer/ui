// Package theme is ui's design tokens and stylesheet (visual-interface.md
// §3): the token table, the stylesheet that joins ui's cascade layers, the
// ui script, and the run-time theme an admin changes through cfgx, whose
// accent colours are derived so that text on and around them stays
// readable.
//
//	t, err := theme.New("/ui", settings) // settings satisfies theme.Config
//	mux.Handle("/ui/", t)
//	…
//	<head> @t.Head() </head>
//
// The visual packages emit class names and never import theme; the CSS for
// them lives here (ADR 0004).
package theme
