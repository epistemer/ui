// Package elements is ui's smallest components (visual-interface.md §5):
// Button, Badge, Alert, Card, Avatar, Spinner, Tooltip, separators,
// Heading, and the value renderers that table cells and detail rows share.
// It also holds what every package above it shares: Merge, and the
// Variant, Size and Tone types.
//
// Components are builders (§2.1): immutable values whose methods return
// changed copies, rendered as templ components.
//
//	@elements.Button().Variant(elements.Outline).Tone(elements.Neutral) { Cancel }
package elements
