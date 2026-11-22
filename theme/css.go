package theme

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// layers are ui's cascade layers, lowest first (ADR 0004). "tokens" is
// rendered from the token table; every other layer is css/<name>.css.
var layers = []string{"reset", "tokens", "base", "elements", "inputs", "components", "layout"}

//go:embed css/*.css js/*.js
var assetFS embed.FS

// bundle is ui's static assets for one release: the joined stylesheet and
// the script modules, served under one content hash (ADR 0004, 0005).
type bundle struct {
	Hash  string            // of every file, so relative module imports stay in one release
	Files map[string][]byte // "ui.css", "ui.js", other modules
}

// newBundle joins the layers and reads the script modules.
func newBundle() (*bundle, error) {
	var css strings.Builder
	css.WriteString("@layer " + layerList() + ";\n")
	for _, name := range layers {
		var body string
		if name == "tokens" {
			body = renderTokens(Defaults{})
		} else {
			b, err := assetFS.ReadFile("css/" + name + ".css")
			if err != nil {
				return nil, fmt.Errorf("theme: layer %s: %w", name, err)
			}
			body = string(b)
		}
		fmt.Fprintf(&css, "@layer ui.%s {\n%s}\n", name, body)
	}
	files := map[string][]byte{"ui.css": []byte(css.String())}
	js, err := fs.Glob(assetFS, "js/*.js")
	if err != nil {
		return nil, err
	}
	for _, p := range js {
		b, err := assetFS.ReadFile(p)
		if err != nil {
			return nil, err
		}
		files[path.Base(p)] = b
	}
	return &bundle{Hash: hashFiles(files), Files: files}, nil
}

func layerList() string {
	names := make([]string, len(layers))
	for i, n := range layers {
		names[i] = "ui." + n
	}
	return strings.Join(names, ", ")
}

func hashFiles(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		fmt.Fprintf(h, "%s\x00%d\x00", n, len(files[n]))
		h.Write(files[n])
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// renderTokens writes the token layer's declarations from c: neutral
// colours and scales from the table, accents derived from c's fills.
func renderTokens(c Config) string {
	var b strings.Builder
	b.WriteString(":root {\n\tcolor-scheme: light dark;\n")
	for _, t := range surfaces {
		fmt.Fprintf(&b, "\t--ui-%s: light-dark(%s, %s);\n", t.Name, t.light, t.dark)
	}
	b.WriteString(renderAccents(c))
	b.WriteString(renderScales(c))
	b.WriteString("}\n")
	b.WriteString(":root[data-color-scheme=\"light\"] { color-scheme: light; }\n")
	b.WriteString(":root[data-color-scheme=\"dark\"] { color-scheme: dark; }\n")
	// Every transition in ui reads a duration token, so reduced motion is
	// one rule here; a rule in ui.reset would lose to every component.
	b.WriteString("@media (prefers-reduced-motion: reduce) {\n\t:root { --ui-duration-s: 0s; --ui-duration-m: 0s; }\n}\n")
	return b.String()
}

// accentTokens lists the six tokens of every accent role, and the focus
// ring, which is the brand's text colour (3:1 or better on any surface).
// The fill, its on-colour and its shades don't depend on the scheme; text
// and subtle are measured against each scheme's surfaces.
func accentTokens(c Config) [][2]string {
	var out [][2]string
	add := func(name, value string) { out = append(out, [2]string{"--ui-" + name, value}) }
	for _, a := range accents {
		fill := mustHex(validColor(fillOf(c, a.Name), a.Fill))
		l, d := derive(fill, schemeLight), derive(fill, schemeDark)
		add("color-"+a.Name, fill.hex())
		add("color-on-"+a.Name, l.On.hex())
		add("color-"+a.Name+"-hover", l.Hover.hex())
		add("color-"+a.Name+"-active", l.Active.hex())
		add("color-"+a.Name+"-text", "light-dark("+l.Text.hex()+", "+d.Text.hex()+")")
		add("color-"+a.Name+"-subtle", "light-dark("+l.Subtle.hex()+", "+d.Subtle.hex()+")")
	}
	add("color-focus", "var(--ui-color-brand-text)")
	return out
}

// overrides lists the tokens an admin's values set, in the order the theme
// stylesheet declares them: the accents and their partners, the font
// families and the radius base. Invalid values give the defaults.
func overrides(c Config) [][2]string {
	out := accentTokens(c)
	out = append(out,
		[2]string{"--ui-font-sans", validFont(c.ThemeFontSans(), scaleDefault("font-sans"))},
		[2]string{"--ui-font-mono", validFont(c.ThemeFontMono(), scaleDefault("font-mono"))},
		[2]string{"--ui-radius", validLength(c.ThemeRadius(), scaleDefault("radius"))},
	)
	return out
}

func renderAccents(c Config) string {
	var b strings.Builder
	for _, kv := range accentTokens(c) {
		fmt.Fprintf(&b, "\t%s: %s;\n", kv[0], kv[1])
	}
	return b.String()
}

func renderScales(c Config) string {
	var b strings.Builder
	for _, t := range scales {
		v := t.light
		switch t.Name {
		case "radius":
			v = validLength(c.ThemeRadius(), t.light)
		case "font-sans":
			v = validFont(c.ThemeFontSans(), t.light)
		case "font-mono":
			v = validFont(c.ThemeFontMono(), t.light)
		}
		fmt.Fprintf(&b, "\t--ui-%s: %s;\n", t.Name, v)
	}
	b.WriteString("\t--ui-radius-s: calc(var(--ui-radius) * 0.5);\n")
	b.WriteString("\t--ui-radius-m: var(--ui-radius);\n")
	b.WriteString("\t--ui-radius-l: calc(var(--ui-radius) * 2);\n")
	b.WriteString("\t--ui-radius-full: 999rem;\n")
	return b.String()
}
