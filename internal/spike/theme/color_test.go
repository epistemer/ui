package theme

import (
	"math"
	"testing"
)

func near(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestOKLabKnownValues(t *testing.T) {
	// Reference values from Björn Ottosson's OKLab post and CSS Color 4.
	cases := []struct {
		hex     string
		l, a, b float64
	}{
		{"#ffffff", 1, 0, 0},
		{"#000000", 0, 0, 0},
		{"#ff0000", 0.627955, 0.224863, 0.125846},
		{"#00ff00", 0.866440, -0.233888, 0.179498},
		{"#0000ff", 0.452014, -0.032457, -0.311528},
	}
	for _, c := range cases {
		got := MustHex(c.hex).Lab()
		if !near(got.L, c.l, 1e-4) || !near(got.A, c.a, 1e-4) || !near(got.B, c.b, 1e-4) {
			t.Errorf("%s: OKLab %+v, want {%v %v %v}", c.hex, got, c.l, c.a, c.b)
		}
		if back := got.RGB().Hex(); back != c.hex {
			t.Errorf("%s: round trip gave %s", c.hex, back)
		}
	}
}

func TestContrast(t *testing.T) {
	if c := Contrast(black, white); !near(c, 21, 1e-9) {
		t.Errorf("black on white: %v, want 21", c)
	}
	// #767676 on white is the classic 4.54:1.
	if c := Contrast(MustHex("#767676"), white); !near(c, 4.54, 0.01) {
		t.Errorf("#767676 on white: %v, want 4.54", c)
	}
}

func TestGamutMappingKeepsLightnessAndHue(t *testing.T) {
	in := LCh{L: 0.7, C: 0.4, H: 145} // far outside sRGB
	got := in.RGB()
	if !got.InGamut() {
		t.Fatalf("%v is out of gamut", got)
	}
	back := got.Lab().LCh()
	if !near(back.L, in.L, 2e-3) || !near(back.H, in.H, 1) || back.C >= in.C {
		t.Errorf("mapped to L %.4f C %.4f H %.1f; want L %.4f, H %.1f and less chroma", back.L, back.C, back.H, in.L, in.H)
	}
}

func TestMixesMatchCSS(t *testing.T) {
	// Mixing with an achromatic colour in OKLCh keeps the other's hue (its
	// hue is powerless, so CSS treats it as missing).
	lime := MustHex("#b4f601")
	m := MixOKLCh(lime, 0.5, white).Lab().LCh()
	if !near(m.H, lime.Lab().LCh().H, 0.5) {
		t.Errorf("oklch mix with white moved the hue to %.1f", m.H)
	}
	// Halfway in OKLab is halfway in L, a and b.
	x := MixOKLab(MustHex("#ff0000"), 0.5, MustHex("#0000ff")).Lab()
	want := Lab{(0.627955 + 0.452014) / 2, (0.224863 - 0.032457) / 2, (0.125846 - 0.311528) / 2}
	if !near(x.L, want.L, 2e-3) || !near(x.A, want.A, 2e-3) || !near(x.B, want.B, 2e-3) {
		t.Errorf("oklab mix %+v, want %+v", x, want)
	}
}

// TestShadesMixInOKLab records why ui mixes shades in OKLab: a near-grey
// with a little chroma has a real hue, and OKLCh interpolates towards it.
// Mixing the lime brand into a dark, slightly blue surface in OKLCh turns
// a subtle tint visibly blue; OKLab keeps it lime.
func TestShadesMixInOKLab(t *testing.T) {
	lime, raised := MustHex("#b4f601"), MustHex("#1a1d23")
	h := lime.Lab().LCh().H
	inLCh := MixOKLCh(lime, SubtleShare, raised).Lab().LCh().H
	inLab := MixOKLab(lime, SubtleShare, raised).Lab().LCh().H
	t.Logf("lime hue %.1f; subtle in oklch %.1f, in oklab %.1f", h, inLCh, inLab)
	if math.Abs(inLCh-h) < 60 {
		t.Errorf("expected OKLCh to drag the hue far from %.1f, got %.1f", h, inLCh)
	}
	if math.Abs(inLab-h) > math.Abs(inLCh-h) {
		t.Errorf("OKLab moved the hue more (%.1f) than OKLCh (%.1f)", inLab, inLCh)
	}
}
