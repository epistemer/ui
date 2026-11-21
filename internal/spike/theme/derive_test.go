package theme

import (
	"math"
	"testing"
)

func TestNeutralTokensReadOnEverySurface(t *testing.T) {
	for _, s := range []Scheme{Light, Dark} {
		for _, text := range []string{"color-text", "color-text-muted"} {
			for _, ground := range []string{"color-surface", "color-surface-raised"} {
				if c := Contrast(surface(text, s), surface(ground, s)); c < 4.5 {
					t.Errorf("%s: %s on %s is %.2f:1", s, text, ground, c)
				}
			}
		}
		// A control's edge is non-text content that identifies it (WCAG
		// 1.4.11): 3:1 against both surfaces.
		for _, ground := range []string{"color-surface", "color-surface-raised"} {
			if c := Contrast(surface("color-border-strong", s), surface(ground, s)); c < 3 {
				t.Errorf("%s: strong border on %s is %.2f:1", s, ground, c)
			}
		}
		// Decorative borders must still be visible against both surfaces.
		for _, ground := range []string{"color-surface", "color-surface-raised"} {
			if c := Contrast(surface("color-border", s), surface(ground, s)); c < 1.2 {
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
		fill := MustHex(hex)
		for _, s := range []Scheme{Light, Dark} {
			p := Derive(fill, s)
			if p.TextMin < 4.5 {
				t.Errorf("%s %s: text %s reaches %.2f:1", hex, s, p.Text.Hex(), p.TextMin)
			}
			if p.OnMin < 4.5 {
				t.Errorf("%s %s: on-colour %s reaches %.2f:1 over the fill and its shades", hex, s, p.On.Hex(), p.OnMin)
			}
			for _, ground := range []string{"color-surface", "color-surface-raised"} {
				if c := Contrast(p.Text, surface(ground, s)); c < 3 {
					t.Errorf("%s %s: text as a focus ring on %s is %.2f:1", hex, s, ground, c)
				}
			}
		}
	}
}

func TestDefaultAccentsCarryText(t *testing.T) {
	for _, a := range Accents {
		for _, s := range []Scheme{Light, Dark} {
			if p := Derive(MustHex(a.Fill), s); p.OnMin < 4.5 {
				t.Errorf("%s %s: on-colour %s reaches %.2f:1", a.Name, s, p.On.Hex(), p.OnMin)
			}
		}
	}
}

func TestDerivedTextStaysNearTheFill(t *testing.T) {
	// The lime brand: unreadable as text on white, so the text colour is
	// a darker lime, not a different colour.
	lime := MustHex("#b4f601")
	p := Derive(lime, Light)
	if p.Text == lime {
		t.Fatal("lime on white was accepted as text")
	}
	if p.On != black {
		t.Errorf("on-lime is %s, want black", p.On.Hex())
	}
	fh, th := lime.Lab().LCh().H, p.Text.Lab().LCh().H
	if d := math.Abs(fh - th); d > 8 {
		t.Errorf("text hue %.1f drifted %.1f° from the fill's %.1f", th, d, fh)
	}
	t.Logf("lime: text %s (%.2f:1), subtle %s", p.Text.Hex(), p.TextMin, p.Subtle.Hex())
}

// TestDeriveGrid sweeps sRGB on a coarse grid: whatever an admin picks,
// every derived colour holds its contrast and Derive returns.
func TestDeriveGrid(t *testing.T) {
	const step = 15 // 18 values per channel, 5832 colours
	for r := 0; r <= 255; r += step {
		for g := 0; g <= 255; g += step {
			for b := 0; b <= 255; b += step {
				fill := RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}
				for _, s := range []Scheme{Light, Dark} {
					p := Derive(fill, s)
					if p.TextMin < 4.5 || p.OnMin < 4.5 {
						t.Fatalf("%s %s: text %.2f:1, on %.2f:1", fill.Hex(), s, p.TextMin, p.OnMin)
					}
					if c := Contrast(p.Text, p.Subtle); c < 4.5 {
						t.Fatalf("%s %s: text on subtle %.2f:1", fill.Hex(), s, c)
					}
				}
			}
		}
	}
}
