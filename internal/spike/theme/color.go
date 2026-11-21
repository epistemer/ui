package theme

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGB is a colour in sRGB, each channel in [0, 1].
type RGB struct{ R, G, B float64 }

// Lab is a colour in OKLab.
type Lab struct{ L, A, B float64 }

// LCh is a colour in OKLCh. H is in degrees; it is meaningless when C is 0.
type LCh struct{ L, C, H float64 }

// ParseHex reads #rgb or #rrggbb.
func ParseHex(s string) (RGB, error) {
	h, ok := strings.CutPrefix(s, "#")
	if !ok || len(h) != 3 && len(h) != 6 {
		return RGB{}, fmt.Errorf("theme: colour %q: want #rgb or #rrggbb", s)
	}
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("theme: colour %q: want #rgb or #rrggbb", s)
	}
	return RGB{float64(n>>16&0xff) / 255, float64(n>>8&0xff) / 255, float64(n&0xff) / 255}, nil
}

// MustHex is ParseHex for colours written in the source.
func MustHex(s string) RGB {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Hex formats c as #rrggbb, clamping each channel.
func (c RGB) Hex() string {
	b := func(v float64) int { return int(math.Round(clamp01(v) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", b(c.R), b(c.G), b(c.B))
}

func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// InGamut reports whether c is inside sRGB, allowing for rounding.
func (c RGB) InGamut() bool {
	const eps = 1e-6
	in := func(v float64) bool { return v >= -eps && v <= 1+eps }
	return in(c.R) && in(c.G) && in(c.B)
}

func toLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func fromLinear(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// Lab converts c to OKLab (Björn Ottosson's matrices, as CSS Color 4 uses).
func (c RGB) Lab() Lab {
	r, g, b := toLinear(c.R), toLinear(c.G), toLinear(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return Lab{
		L: 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		A: 1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		B: 0.0259040371*l + 0.7827717662*m - 0.8086757660*s,
	}
}

// RGB converts c to sRGB. The result may be outside the gamut.
func (c Lab) RGB() RGB {
	l := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s := c.L - 0.0894841775*c.A - 1.2914855480*c.B
	l, m, s = l*l*l, m*m*m, s*s*s
	return RGB{
		R: fromLinear(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		G: fromLinear(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		B: fromLinear(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s),
	}
}

// LCh converts c to OKLCh.
func (c Lab) LCh() LCh {
	h := math.Atan2(c.B, c.A) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return LCh{L: c.L, C: math.Hypot(c.A, c.B), H: h}
}

// Lab converts c to OKLab.
func (c LCh) Lab() Lab {
	r := c.H * math.Pi / 180
	return Lab{L: c.L, A: c.C * math.Cos(r), B: c.C * math.Sin(r)}
}

// RGB converts c to sRGB, reducing chroma until it fits the gamut, at the
// same lightness and hue. This is the gamut mapping of CSS Color 4 without
// its tolerance for clipping, so the result never lightens or darkens.
func (c LCh) RGB() RGB {
	c.L = math.Max(0, math.Min(1, c.L))
	if rgb := c.Lab().RGB(); rgb.InGamut() {
		return rgb
	}
	lo, hi := 0.0, c.C
	for range 40 {
		mid := (lo + hi) / 2
		if (LCh{c.L, mid, c.H}).Lab().RGB().InGamut() {
			lo = mid
		} else {
			hi = mid
		}
	}
	rgb := LCh{c.L, lo, c.H}.Lab().RGB()
	return RGB{clamp01(rgb.R), clamp01(rgb.G), clamp01(rgb.B)}
}

// Luminance is c's relative luminance (WCAG 2).
func (c RGB) Luminance() float64 {
	return 0.2126*toLinear(clamp01(c.R)) + 0.7152*toLinear(clamp01(c.G)) + 0.0722*toLinear(clamp01(c.B))
}

// Contrast is the WCAG 2 contrast ratio of a and b, from 1 to 21.
func Contrast(a, b RGB) float64 {
	la, lb := a.Luminance(), b.Luminance()
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// MixOKLab is CSS color-mix(in oklab, a p, b): p of a, the rest of b,
// both opaque.
func MixOKLab(a RGB, p float64, b RGB) RGB {
	x, y := a.Lab(), b.Lab()
	m := Lab{
		L: x.L*p + y.L*(1-p),
		A: x.A*p + y.A*(1-p),
		B: x.B*p + y.B*(1-p),
	}.RGB()
	// sRGB isn't convex in OKLab: a mix of two sRGB colours can land a
	// hair outside it.
	return RGB{clamp01(m.R), clamp01(m.G), clamp01(m.B)}
}

// MixOKLCh is CSS color-mix(in oklch, a p, b), with the shorter hue arc. A
// colour without chroma has a powerless hue, which CSS treats as missing:
// the other colour's hue is used.
func MixOKLCh(a RGB, p float64, b RGB) RGB {
	x, y := a.Lab().LCh(), b.Lab().LCh()
	const powerless = 1e-4
	switch {
	case x.C < powerless && y.C >= powerless:
		x.H = y.H
	case y.C < powerless && x.C >= powerless:
		y.H = x.H
	}
	d := y.H - x.H
	switch {
	case d > 180:
		d -= 360
	case d < -180:
		d += 360
	}
	h := x.H + d*(1-p)
	if h < 0 {
		h += 360
	}
	return LCh{
		L: x.L*p + y.L*(1-p),
		C: x.C*p + y.C*(1-p),
		H: math.Mod(h, 360),
	}.RGB() // gamut-mapped, as CSS does
}
