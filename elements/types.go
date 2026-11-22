package elements

// Variant is a button's look.
type Variant string

// Size is a component's size.
type Size string

// Tone is the colour role a component takes. The zero Tone is the
// component's own default: brand for Button, neutral for Badge and
// MenuItem, info for Alert and Toast (§2.3).
type Tone string

const (
	Solid   Variant = "" // the default: the tone's fill
	Outline Variant = "outline"
	Ghost   Variant = "ghost"

	Small  Size = "s"
	Medium Size = "" // the default
	Large  Size = "l"

	Neutral Tone = "neutral"
	Brand   Tone = "brand"
	Danger  Tone = "danger"
	Warning Tone = "warning"
	Success Tone = "success"
	Info    Tone = "info"
)

// toneAttr is the data-tone a component renders: none when the tone it
// resolves to is its default.
func toneAttr(t, def Tone) string {
	if t == "" || t == def {
		return ""
	}
	return string(t)
}
