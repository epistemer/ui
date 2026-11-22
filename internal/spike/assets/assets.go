// Package assets holds the spike's stylesheet and script, and htmx 2.0.11
// (Zero-Clause BSD, LICENSE.htmx) for its pages.
package assets

import "embed"

//go:embed spike.css spike.js htmx.min.js LICENSE.htmx
var FS embed.FS
