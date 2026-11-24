// Package icons is ui's icon set (visual-interface.md §4): inline SVG on a
// 24-unit grid from Lucide, drawn in currentColor and sized 1.25em by ui's
// CSS by default, so an icon follows the text around it. Every icon is
// decorative (aria-hidden) unless Label is set or it is wrapped in Labelled.
// Directional Start/End icons mirror in RTL.
//
// The paths are Lucide's (https://lucide.dev), ISC licence, kept with
// this package in LICENSE.lucide (ADR 0014).
//
//go:generate go run ./internal/gen
package icons

import (
	"context"
	"fmt"
	"io"
	"maps"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/internal/attrs"
)

// Size picks a named icon size. SizeText (the zero value) follows the
// surrounding text at 1.25em; SizeS, SizeM and SizeL emit data-size.
type Size uint8

const (
	SizeText Size = iota // 1.25em, follows the text around it (the default)
	SizeS                // 1rem
	SizeM                // 1.5rem
	SizeL                // 2rem
)

// Color picks a semantic colour role. ColorCurrent (the zero value)
// inherits currentColor; the rest emit data-color.
type Color uint8

const (
	ColorCurrent Color = iota // currentColor (the default)
	ColorMuted                // --ui-color-text-muted
	ColorBrand                // --ui-color-brand-text
	ColorDanger               // --ui-color-danger-text
	ColorWarning              // --ui-color-warning-text
	ColorSuccess              // --ui-color-success-text
	ColorInfo                 // --ui-color-info-text
)

// IconSpec is an icon. It renders <svg class="ui-icon" …>. Every method
// returns a changed copy, so specs can be shared and refined safely.
type IconSpec struct {
	shapes    string
	mirror    bool
	size      Size
	color     Color
	stroke    float64
	strokeSet bool
	label     i18n.Ter
	labelSet  bool
	attrs     templ.Attributes
}

// Size sets the icon's size.
func (i IconSpec) Size(s Size) IconSpec { i.size = s; return i }

// Stroke sets the stroke width (must be > 0 and <= 4).
func (i IconSpec) Stroke(width float64) IconSpec {
	i.stroke = width
	i.strokeSet = true
	return i
}

// Color sets the icon's colour role.
func (i IconSpec) Color(c Color) IconSpec { i.color = c; return i }

// Label gives the icon an accessible name (role="img" and aria-label) in
// place of aria-hidden="true".
func (i IconSpec) Label(t i18n.Ter) IconSpec {
	i.label = t
	i.labelSet = true
	return i
}

// Attrs adds attributes to the <svg>, merged with elements.Merge's rules.
// Owned keys: viewBox, fill, stroke, stroke-width, stroke-linecap,
// stroke-linejoin, aria-hidden, role, aria-label, data-mirror, data-size,
// data-color.
func (i IconSpec) Attrs(a templ.Attributes) IconSpec {
	i.attrs = attrs.Merge("icons", i.attrs, nil, a)
	return i
}

var iconOwned = []string{
	"viewBox", "fill", "stroke", "stroke-width", "stroke-linecap",
	"stroke-linejoin", "aria-hidden", "role", "aria-label",
	"data-mirror", "data-size", "data-color",
}

// More is a row of three dots: more actions (Lucide's "ellipsis").
func More() IconSpec { return Ellipsis() }

// Warning is a warning triangle (Lucide's "triangle-alert").
func Warning() IconSpec { return TriangleAlert() }

func sizeAttr(s Size) string {
	switch s {
	case SizeText:
		return ""
	case SizeS:
		return "s"
	case SizeM:
		return "m"
	case SizeL:
		return "l"
	default:
		panic(fmt.Sprintf("icons: unknown Size %d", s))
	}
}

func colorAttr(c Color) string {
	switch c {
	case ColorCurrent:
		return ""
	case ColorMuted:
		return "muted"
	case ColorBrand:
		return "brand"
	case ColorDanger:
		return "danger"
	case ColorWarning:
		return "warning"
	case ColorSuccess:
		return "success"
	case ColorInfo:
		return "info"
	default:
		panic(fmt.Sprintf("icons: unknown Color %d", c))
	}
}

// Render writes the icon SVG to w.
func (i IconSpec) Render(ctx context.Context, w io.Writer) error {
	if i.shapes == "" {
		panic("icons: zero IconSpec (call an icon function such as icons.Check())")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	sz := sizeAttr(i.size)
	col := colorAttr(i.color)

	strokeWidth := "2"
	if i.strokeSet {
		if math.IsNaN(i.stroke) || math.IsInf(i.stroke, 0) || i.stroke <= 0 || i.stroke > 4 {
			panic(fmt.Sprintf("icons: Stroke(%v) must be > 0 and <= 4", i.stroke))
		}
		if i.stroke != 2 {
			strokeWidth = strconv.FormatFloat(i.stroke, 'f', -1, 64)
		}
	}

	var labelText string
	if i.labelSet {
		if i.label == nil {
			panic("icons: Label with a nil Ter")
		}
		labelText = i.label.T(ctx)
		if strings.TrimSpace(labelText) == "" {
			panic("icons: Label with a blank text")
		}
	}

	classVal := "ui-icon"
	var extra templ.Attributes
	if len(i.attrs) > 0 {
		own := templ.Attributes{
			"class":           "ui-icon",
			"viewBox":         "0 0 24 24",
			"fill":            "none",
			"stroke":          "currentColor",
			"stroke-width":    strokeWidth,
			"stroke-linecap":  "round",
			"stroke-linejoin": "round",
		}
		if i.mirror {
			own["data-mirror"] = true
		}
		if sz != "" {
			own["data-size"] = sz
		}
		if col != "" {
			own["data-color"] = col
		}
		if i.labelSet {
			own["role"] = "img"
			own["aria-label"] = labelText
		} else {
			own["aria-hidden"] = "true"
		}
		merged := attrs.Merge("icons", own, iconOwned, i.attrs)
		if c, ok := merged["class"].(string); ok {
			classVal = c
		}
		extra = maps.Clone(merged)
		delete(extra, "class")
		for _, k := range iconOwned {
			delete(extra, strings.ToLower(k))
		}
	}

	var b strings.Builder
	b.WriteString(`<svg class="`)
	b.WriteString(templ.EscapeString(classVal))
	b.WriteByte('"')
	if i.mirror {
		b.WriteString(` data-mirror`)
	}
	if sz != "" {
		b.WriteString(` data-size="`)
		b.WriteString(sz)
		b.WriteByte('"')
	}
	if col != "" {
		b.WriteString(` data-color="`)
		b.WriteString(col)
		b.WriteByte('"')
	}
	b.WriteString(` viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="`)
	b.WriteString(strokeWidth)
	b.WriteString(`" stroke-linecap="round" stroke-linejoin="round"`)
	if i.labelSet {
		b.WriteString(` role="img" aria-label="`)
		b.WriteString(templ.EscapeString(labelText))
		b.WriteByte('"')
	} else {
		b.WriteString(` aria-hidden="true"`)
	}
	if len(extra) > 0 {
		buf := templ.GetBuffer()
		if err := templruntime.RenderAttributes(ctx, buf, w, "svg", extra); err != nil {
			templ.ReleaseBuffer(buf)
			return err
		}
		b.Write(buf.Bytes())
		templ.ReleaseBuffer(buf)
	}
	b.WriteByte('>')
	b.WriteString(i.shapes)
	b.WriteString(`</svg>`)

	_, err := io.WriteString(w, b.String())
	return err
}
