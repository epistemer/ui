package theme

import "math"

// scheme is light or dark.
type scheme int

const (
	schemeLight scheme = iota
	schemeDark
)

func (s scheme) String() string {
	if s == schemeDark {
		return "dark"
	}
	return "light"
}

// textContrast is what derived text must reach: WCAG's 4.5:1 with a margin
// for the browser's own rounding of the colours we emit.
const textContrast = 4.55

// subtleShare is the fill's share of a subtle background, mixed with the
// raised surface in OKLab.
const subtleShare = 0.14

var (
	black = rgb{0, 0, 0}
	white = rgb{1, 1, 1}
)

// partners are an accent's colours in one scheme, with the contrast each
// derived colour reached.
type partners struct {
	Fill, On, Hover, Active, Text, Subtle rgb
	OnMin                                 float64 // the on-colour's worst contrast over the fill, hover and active
	TextMin                               float64 // the text colour's worst contrast over the surfaces and the subtle background
}

func surface(name string, s scheme) rgb {
	for _, t := range surfaces {
		if t.Name == name {
			if s == schemeDark && t.dark != "" {
				return mustHex(t.dark)
			}
			return mustHex(t.light)
		}
	}
	panic("theme: no surface " + name)
}

// derive computes an accent's partners in scheme s from its fill
// (visual-design.md §4).
func derive(fill rgb, s scheme) partners {
	raised := surface("color-surface-raised", s)
	grounds := []rgb{surface("color-surface", s), raised}

	fill = mustHex(fill.hex()) // measure what the page will get
	p := partners{Fill: fill}
	worst := func(c rgb, against []rgb) float64 {
		w := math.Inf(1)
		for _, a := range against {
			w = math.Min(w, contrast(c, a))
		}
		return w
	}

	// On: black or white, whichever contrasts more with the fill; one of
	// them always reaches 4.58:1. Hover and active move away from it.
	away := white
	if contrast(black, fill) >= contrast(white, fill) {
		p.On = black
	} else {
		p.On, away = white, black
	}
	p.Hover = mustHex(mixOKLab(fill, hoverShare, away).hex())
	p.Active = mustHex(mixOKLab(fill, activeShare, away).hex())
	p.OnMin = worst(p.On, []rgb{fill, p.Hover, p.Active})

	// Subtle: a tint of the raised surface.
	p.Subtle = mustHex(mixOKLab(fill, subtleShare, raised).hex())
	grounds = append(grounds, p.Subtle)

	// Text: the fill itself if it already reads on every ground; otherwise
	// the colour nearest to it in OKLCh lightness that does, darker in the
	// light scheme and lighter in the dark one, with chroma reduced to stay
	// in sRGB.
	if c := worst(fill, grounds); c >= textContrast {
		p.Text, p.TextMin = fill, c
		return p
	}
	base := fill.lab().lch()
	target := 0.0
	if s == schemeDark {
		target = 1
	}
	at := func(l float64) rgb { return lch{L: l, C: base.C, H: base.H}.rgb() }
	near, far := base.L, target // near fails, far passes (black or white always does)
	for range 50 {
		mid := (near + far) / 2
		if worst(at(mid), grounds) >= textContrast {
			far = mid
		} else {
			near = mid
		}
	}
	// Emitting hex rounds each channel, which can cost a little contrast:
	// measure the rounded colour, and step on towards black or white until
	// it holds. It always does at the end, so derive never fails on what
	// an admin picks.
	l := far
	for {
		p.Text = mustHex(at(l).hex())
		if p.TextMin = worst(p.Text, grounds); p.TextMin >= 4.5 || l == target {
			return p
		}
		if s == schemeDark {
			l = math.Min(1, l+0.005)
		} else {
			l = math.Max(0, l-0.005)
		}
	}
}
