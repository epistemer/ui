package theme

import (
	"fmt"
	"math"
)

// Scheme is light or dark.
type Scheme int

const (
	Light Scheme = iota
	Dark
)

func (s Scheme) String() string {
	if s == Dark {
		return "dark"
	}
	return "light"
}

// TextContrast is what derived text must reach: WCAG's 4.5:1 with a margin
// for the browser's own rounding of the colours we emit.
const TextContrast = 4.55

// SubtleShare is the fill's share of a subtle background, mixed with the
// raised surface in OKLab.
const SubtleShare = 0.14

var (
	black = RGB{0, 0, 0}
	white = RGB{1, 1, 1}
)

// Partners are an accent's colours in one scheme, with the contrast each
// derived colour reached.
type Partners struct {
	Fill, On, Hover, Active, Text, Subtle RGB
	OnMin                                 float64 // the on-colour's worst contrast over the fill, hover and active
	TextMin                               float64 // the text colour's worst contrast over the surfaces and the subtle background
}

func surface(name string, s Scheme) RGB {
	for _, t := range Surfaces {
		if t.Name == name {
			if s == Dark && t.Dark != "" {
				return MustHex(t.Dark)
			}
			return MustHex(t.Light)
		}
	}
	panic("theme: no surface " + name)
}

// Derive computes an accent's partners in scheme s from its fill
// (visual-design.md §4).
func Derive(fill RGB, s Scheme) Partners {
	raised := surface("color-surface-raised", s)
	grounds := []RGB{surface("color-surface", s), raised}

	p := Partners{Fill: fill}
	worst := func(c RGB, against []RGB) float64 {
		w := math.Inf(1)
		for _, a := range against {
			w = math.Min(w, Contrast(c, a))
		}
		return w
	}

	// On: black or white, whichever contrasts more with the fill; one of
	// them always reaches 4.58:1. Hover and active move away from it.
	away := white
	if Contrast(black, fill) >= Contrast(white, fill) {
		p.On = black
	} else {
		p.On, away = white, black
	}
	p.Hover = MustHex(MixOKLab(fill, HoverShare, away).Hex())
	p.Active = MustHex(MixOKLab(fill, ActiveShare, away).Hex())
	p.OnMin = worst(p.On, []RGB{fill, p.Hover, p.Active})

	// Subtle: a tint of the raised surface.
	p.Subtle = MixOKLab(fill, SubtleShare, raised)
	grounds = append(grounds, p.Subtle)

	// Text: the fill itself if it already reads on every ground; otherwise
	// the colour nearest to it in OKLCh lightness that does, darker in the
	// light scheme and lighter in the dark one, with chroma reduced to stay
	// in sRGB.
	if c := worst(fill, grounds); c >= TextContrast {
		p.Text, p.TextMin = fill, c
		return p
	}
	base := fill.Lab().LCh()
	target := 0.0
	if s == Dark {
		target = 1
	}
	at := func(l float64) RGB { return LCh{L: l, C: base.C, H: base.H}.RGB() }
	near, far := base.L, target // near fails, far passes (black or white always does)
	for range 50 {
		mid := (near + far) / 2
		if worst(at(mid), grounds) >= TextContrast {
			far = mid
		} else {
			near = mid
		}
	}
	p.Text = at(far)
	// Emitting hex rounds each channel; check the colour the page will get.
	p.Text = MustHex(p.Text.Hex())
	p.TextMin = worst(p.Text, grounds)
	if p.TextMin < 4.5 {
		panic(fmt.Sprintf("theme: derived text %s reaches only %.2f", p.Text.Hex(), p.TextMin))
	}
	return p
}
