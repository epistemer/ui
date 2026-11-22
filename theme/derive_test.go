package theme

import (
	"math"
	"testing"
)

func TestNeutralTokensReadOnEverySurface(t *testing.T) {
	for _, s := range []scheme{schemeLight, schemeDark} {
		for _, text := range []string{"color-text", "color-text-muted"} {
			for _, ground := range []string{"color-surface", "color-surface-raised"} {
				if c := contrast(surface(text, s), surface(ground, s)); c < 4.5 {
					t.Errorf("%s: %s on %s is %.2f:1", s, text, ground, c)
				}
			}
		}
		// A control's edge is non-text content that identifies it (WCAG
		// 1.4.11): 3:1 against both surfaces.
		for _, ground := range []string{"color-surface", "color-surface-raised"} {
			if c := contrast(surface("color-border-strong", s), surface(ground, s)); c < 3 {
				t.Errorf("%s: strong border on %s is %.2f:1", s, ground, c)
			}
		}
		// Decorative borders must still be visible against both surfaces.
		for _, ground := range []string{"color-surface", "color-surface-raised"} {
			if c := contrast(surface("color-border", s), surface(ground, s)); c < 1.2 {
				t.Errorf("%s: border on %s is %.2f:1", s, ground, c)
			}
		}
	}
}

// fills covers the defaults, zp's lime brand, and the extremes an admin
// might pick.
var fills = []string{
	"#2f5bd3", "#c62828", "#f2b01e", "#1e8a4c", "#0f7c9e", // defaults
	"#b4f601", // a lime that only works as a fill
	"#ffffff", "#000000", "#808080", "#ffff00", "#0000ff", "#ff00ff",
	"#f6f7f9", "#0f1115", // the surfaces themselves
}

func TestDerivedPartnersReachContrast(t *testing.T) {
	for _, hex := range fills {
		fill := mustHex(hex)
		for _, s := range []scheme{schemeLight, schemeDark} {
			p := derive(fill, s)
			if p.TextMin < 4.5 {
				t.Errorf("%s %s: text %s reaches %.2f:1", hex, s, p.Text.hex(), p.TextMin)
			}
			if p.OnMin < 4.5 {
				t.Errorf("%s %s: on-colour %s reaches %.2f:1 over the fill and its shades", hex, s, p.On.hex(), p.OnMin)
			}
			for _, ground := range []string{"color-surface", "color-surface-raised"} {
				if c := contrast(p.Text, surface(ground, s)); c < 3 {
					t.Errorf("%s %s: text as a focus ring on %s is %.2f:1", hex, s, ground, c)
				}
			}
		}
	}
}

func TestDefaultAccentsCarryText(t *testing.T) {
	for _, a := range accents {
		for _, s := range []scheme{schemeLight, schemeDark} {
			if p := derive(mustHex(a.Fill), s); p.OnMin < 4.5 {
				t.Errorf("%s %s: on-colour %s reaches %.2f:1", a.Name, s, p.On.hex(), p.OnMin)
			}
		}
	}
}

func TestDerivedTextStaysNearTheFill(t *testing.T) {
	// The lime brand: unreadable as text on white, so the text colour is
	// a darker lime, not a different colour.
	lime := mustHex("#b4f601")
	p := derive(lime, schemeLight)
	if p.Text == lime {
		t.Fatal("lime on white was accepted as text")
	}
	if p.On != black {
		t.Errorf("on-lime is %s, want black", p.On.hex())
	}
	fh, th := lime.lab().lch().H, p.Text.lab().lch().H
	if d := math.Abs(fh - th); d > 8 {
		t.Errorf("text hue %.1f drifted %.1f° from the fill's %.1f", th, d, fh)
	}
	t.Logf("lime: text %s (%.2f:1), subtle %s", p.Text.hex(), p.TextMin, p.Subtle.hex())
}

// TestDeriveGrid sweeps sRGB on a coarse grid: whatever an admin picks,
// every derived colour holds its contrast and derive returns.
func TestDeriveGrid(t *testing.T) {
	const step = 15 // 18 values per channel, 5832 colours
	for r := 0; r <= 255; r += step {
		for g := 0; g <= 255; g += step {
			for b := 0; b <= 255; b += step {
				fill := rgb{float64(r) / 255, float64(g) / 255, float64(b) / 255}
				for _, s := range []scheme{schemeLight, schemeDark} {
					p := derive(fill, s)
					if p.TextMin < 4.5 || p.OnMin < 4.5 {
						t.Fatalf("%s %s: text %.2f:1, on %.2f:1", fill.hex(), s, p.TextMin, p.OnMin)
					}
					if c := contrast(p.Text, p.Subtle); c < 4.5 {
						t.Fatalf("%s %s: text on subtle %.2f:1", fill.hex(), s, c)
					}
				}
			}
		}
	}
}
