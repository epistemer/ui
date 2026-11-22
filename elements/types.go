package elements

import "fmt"

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

func checkVariant(component string, v Variant) {
	switch v {
	case Solid, Outline, Ghost:
	default:
		panic(fmt.Sprintf("elements: %s with variant %q: Solid, Outline or Ghost", component, v))
	}
}

func checkSize(component string, s Size) {
	switch s {
	case Small, Medium, Large:
	default:
		panic(fmt.Sprintf("elements: %s with size %q: Small, Medium or Large", component, s))
	}
}

func checkTone(component string, t Tone) {
	switch t {
	case "", Neutral, Brand, Danger, Warning, Success, Info:
	default:
		panic(fmt.Sprintf("elements: %s with tone %q: Neutral, Brand, Danger, Warning, Success or Info", component, t))
	}
}

// toneAttr is the data-tone a component renders: none when the tone it
// resolves to is its default.
func toneAttr(t, def Tone) string {
	if t == "" || t == def {
		return ""
	}
	return string(t)
}
