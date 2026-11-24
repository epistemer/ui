package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestParseSVGRefusals(t *testing.T) {
	bad := map[string]struct {
		svg  string
		want string // expected substring in error (e.g. file or attribute)
	}{
		"script element": {
			svg:  `<svg viewBox="0 0 24 24"><script>alert(1)</script></svg>`,
			want: "script",
		},
		"onload attribute on svg": {
			svg:  `<svg viewBox="0 0 24 24" onload="alert(1)"><path d="M0 0h24"/></svg>`,
			want: "onload",
		},
		"onload attribute on path": {
			svg:  `<svg viewBox="0 0 24 24"><path d="M0 0h24" onload="alert(1)"/></svg>`,
			want: "onload",
		},
		"custom entity": {
			svg:  `<svg viewBox="0 0 24 24"><path d="M0 &xxe; 0"/></svg>`,
			want: "XML",
		},
		"doctype": {
			svg:  `<!DOCTYPE svg [<!ENTITY xxe "foo">]><svg viewBox="0 0 24 24"><path d="M0 0"/></svg>`,
			want: "directive",
		},
		"d with quote": {
			svg:  `<svg viewBox="0 0 24 24"><path d='M0 0" onclick="x'/></svg>`,
			want: "d=",
		},
		"d with brace": {
			svg:  `<svg viewBox="0 0 24 24"><path d="M0 0{expr}"/></svg>`,
			want: "d=",
		},
		"unknown element": {
			svg:  `<svg viewBox="0 0 24 24"><g><path d="M0 0"/></g></svg>`,
			want: "<g>",
		},
		"unknown attribute on rect": {
			svg:  `<svg viewBox="0 0 24 24"><rect x="1" y="1" width="10" height="10" style="fill:red"/></svg>`,
			want: "style",
		},
		"circle fill non-currentColor": {
			svg:  `<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="4" fill="red"/></svg>`,
			want: "fill",
		},
		"wrong viewBox": {
			svg:  `<svg viewBox="0 0 16 16"><path d="M0 0"/></svg>`,
			want: "viewBox",
		},
		"missing viewBox": {
			svg:  `<svg><path d="M0 0"/></svg>`,
			want: "viewBox",
		},
		"processing instruction": {
			svg:  `<?xml-stylesheet href="evil.xsl"?><svg viewBox="0 0 24 24"><path d="M0 0"/></svg>`,
			want: "processing instruction",
		},
		"points with letter": {
			svg:  `<svg viewBox="0 0 24 24"><polyline points="0 0 a 10"/></svg>`,
			want: "points",
		},
	}

	for name, tc := range bad {
		_, err := parseSVG("bad.svg", []byte(tc.svg))
		if err == nil {
			t.Errorf("%s: expected error, got nil", name)
			continue
		}
		if !strings.Contains(err.Error(), "bad.svg") || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %q should name file bad.svg and %q", name, err.Error(), tc.want)
		}
	}
}

func TestParseSVGAroundXMLBuiltinsAndShapes(t *testing.T) {
	svg := `<!-- @license -->
<svg class="lucide lucide-demo" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M18 6 6 18" />
  <circle cx="12" cy="12" r="1" fill="currentColor" />
  <ellipse cx="12" cy="12" rx="4" ry="2.5" />
  <line x1="1" y1="2" x2="3" y2="4" />
  <polyline points="1 2, 3.5 -4" />
  <polygon points="0 0 10 10 0 10" />
  <rect x="2" y="3" width="18" height="14" rx="2" ry="2" />
</svg>`
	got, err := parseSVG("demo.svg", []byte(svg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `<path d="M18 6 6 18"></path> <circle cx="12" cy="12" r="1" fill="currentColor"></circle> <ellipse cx="12" cy="12" rx="4" ry="2.5"></ellipse> <line x1="1" y1="2" x2="3" y2="4"></line> <polyline points="1 2, 3.5 -4"></polyline> <polygon points="0 0 10 10 0 10"></polygon> <rect x="2" y="3" width="18" height="14" rx="2" ry="2"></rect>`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestNamingPairsExclusionsCollisionsAndKept(t *testing.T) {
	active := map[string]iconEntry{
		"arrow-left":           {Name: "arrow-left", Shapes: `<path d="M19 12H5"></path>`},
		"arrow-right":          {Name: "arrow-right", Shapes: `<path d="M5 12h14"></path>`},
		"arrow-left-right":     {Name: "arrow-left-right", Shapes: `<path d="M8 3 4 7"></path>`},
		"arrow-right-left":     {Name: "arrow-right-left", Shapes: `<path d="M16 3 20 7"></path>`},
		"copyleft":             {Name: "copyleft", Shapes: `<circle cx="12" cy="12" r="10"></circle>`},
		"copyright":            {Name: "copyright", Shapes: `<circle cx="12" cy="12" r="9"></circle>`},
		"pilcrow-left":         {Name: "pilcrow-left", Shapes: `<path d="M10 4v16"></path>`},
		"pilcrow-right":        {Name: "pilcrow-right", Shapes: `<path d="M14 4v16"></path>`},
		"decimals-arrow-left":  {Name: "decimals-arrow-left", Shapes: `<path d="M1 1"></path>`},
		"decimals-arrow-right": {Name: "decimals-arrow-right", Shapes: `<path d="M2 2"></path>`},
		"mouse-left":           {Name: "mouse-left", Shapes: `<path d="M3 3"></path>`},
		"mouse-right":          {Name: "mouse-right", Shapes: `<path d="M4 4"></path>`},
		"toggle-left":          {Name: "toggle-left", Shapes: `<path d="M5 5"></path>`},
		"toggle-right":         {Name: "toggle-right", Shapes: `<path d="M6 6"></path>`},
		// Identical shapes merge into one function:
		"arrow-up-01":  {Name: "arrow-up-01", Shapes: `<path d="M12 19V5"></path>`},
		"arrow-up-0-1": {Name: "arrow-up-0-1", Shapes: `<path d="M12 19V5"></path>`},
	}
	kept := map[string]iconEntry{
		"old-badge": {Name: "old-badge", Shapes: `<path d="M1 2"></path>`, Deprecated: true, Replacement: "badge"},
	}

	funcs, err := buildFunctions(active, kept)
	if err != nil {
		t.Fatalf("buildFunctions failed: %v", err)
	}
	byName := map[string]generatedFunc{}
	for _, fn := range funcs {
		byName[fn.GoName] = fn
	}

	// Paired Start/End carry Mirror=true; Left/Right carry Mirror=false.
	for _, name := range []string{"ArrowStart", "ArrowEnd", "ArrowStartEnd", "ArrowEndStart"} {
		fn, ok := byName[name]
		if !ok {
			t.Errorf("missing expected paired function %s", name)
			continue
		}
		if !fn.Mirror {
			t.Errorf("%s should have Mirror=true", name)
		}
	}
	for _, name := range []string{"ArrowLeft", "ArrowRight", "ArrowLeftRight", "ArrowRightLeft"} {
		fn, ok := byName[name]
		if !ok {
			t.Errorf("missing expected function %s", name)
			continue
		}
		if fn.Mirror {
			t.Errorf("%s should have Mirror=false", name)
		}
	}

	// copyleft/copyright is not a pair; exclusions produce no Start/End.
	for _, unexpected := range []string{
		"Copystart", "Copyend",
		"PilcrowStart", "PilcrowEnd",
		"DecimalsArrowStart", "DecimalsArrowEnd",
		"MouseStart", "MouseEnd",
		"ToggleStart", "ToggleEnd",
	} {
		if _, ok := byName[unexpected]; ok {
			t.Errorf("unexpected function %s generated", unexpected)
		}
	}

	// Identical aliases merged.
	if fn, ok := byName["ArrowUp01"]; !ok || len(fn.LucideNames) != 2 {
		t.Errorf("ArrowUp01 = %+v, want merged 2 aliases", fn)
	}

	// Kept icon is marked deprecated.
	if fn, ok := byName["OldBadge"]; !ok || !fn.Deprecated || fn.Replacement != "badge" {
		t.Errorf("OldBadge = %+v, want deprecated kept icon", fn)
	}
}

func TestNamingRefusesRealCollisionAndReservedNames(t *testing.T) {
	// Different shapes sharing a Go name must fail.
	_, err := buildFunctions(map[string]iconEntry{
		"arrow-up-01":  {Name: "arrow-up-01", Shapes: `<path d="M1 1"></path>`},
		"arrow-up-0-1": {Name: "arrow-up-0-1", Shapes: `<path d="M2 2"></path>`},
	}, nil)
	if err == nil {
		t.Error("expected collision error for differing shapes, got nil")
	}

	for _, bad := range []string{"2fa", "icon-spec", "size-s", "color-brand", "labelled", "more", "warning"} {
		if _, err := toGoName(bad); err == nil {
			t.Errorf("toGoName(%q) unexpectedly succeeded", bad)
		}
	}
}

func TestExtractTarRejectsUnsafeEntries(t *testing.T) {
	makeTar := func(hdr tar.Header, body string) []byte {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		if hdr.Typeflag == tar.TypeReg {
			hdr.Size = int64(len(body))
		}
		if err := tw.WriteHeader(&hdr); err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		tw.Close()
		gz.Close()
		return buf.Bytes()
	}

	for _, hdr := range []tar.Header{
		{Name: "../evil.svg", Typeflag: tar.TypeReg},
		{Name: "/etc/passwd", Typeflag: tar.TypeReg},
		{Name: "package/icons/a.svg", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
	} {
		if _, _, err := extractTarIcons(bytes.NewReader(makeTar(hdr, "<svg/>"))); err == nil {
			t.Errorf("tar entry %+v was accepted", hdr)
		}
	}

	// An SVG over 4 MB is refused.
	bigSVG := `<svg viewBox="0 0 24 24"><path d="` + strings.Repeat("M0 0", (maxEntryBytes/4)+10) + `"/></svg>`
	if _, _, err := extractTarIcons(bytes.NewReader(makeTar(tar.Header{Name: "package/icons/big.svg", Typeflag: tar.TypeReg}, bigSVG))); err == nil {
		t.Error("oversized SVG entry was accepted")
	}
}

func TestRecheckShapes(t *testing.T) {
	if err := recheckShapes("ok", `<path d="M5 12h14"></path>`); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		`<path d="M5 12h14" onclick="x()"></path>`,
		`<script>alert(1)</script>`,
		`<path d="M5 12h14"></path><img src=x>`,
		`<path d='M5 12h14'/>`,
	} {
		if err := recheckShapes("bad", bad); err == nil {
			t.Errorf("recheckShapes accepted %q", bad)
		}
	}
}
