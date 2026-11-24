// Command gen vendors Lucide SVG shapes (-fetch <version>) and generates
// icons/lucide.go and internal/iconlist/iconlist.go (offline, via go generate).
package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"html"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	maxUnpackedTotal = 64 << 20 // 64 MB unpacked total
	maxEntryBytes    = 4 << 20  // 4 MB per entry
	npmRegistryBase  = "https://registry.npmjs.org/lucide-static/"
)

// notMirroredPrefixes are left/right pairs that depict glyphs, digits,
// hardware or switches whose orientation does not flip in RTL (§5.1).
var notMirroredPrefixes = []string{
	"pilcrow-",
	"decimals-arrow-",
	"mouse-",
	"toggle-",
}

func main() {
	fetch := flag.String("fetch", "", "fetch and vendor lucide-static `<version>` from npm")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		fatalf("%v", err)
	}
	if *fetch != "" {
		if err := runFetch(root, *fetch); err != nil {
			fatalf("fetch %s: %v", *fetch, err)
		}
	}
	if err := runGenerate(root); err != nil {
		fatalf("generate: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "icons/internal/gen: "+format+"\n", args...)
	os.Exit(1)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "icons", "internal", "gen")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot locate repository root containing go.mod and icons/internal/gen")
		}
		dir = parent
	}
}

type npmPackageVersion struct {
	Dist struct {
		Tarball   string `json:"tarball"`
		Integrity string `json:"integrity"`
	} `json:"dist"`
}

func runFetch(root, version string) error {
	if version == "" || strings.ContainsAny(version, "/\\?# \t\r\n") {
		return fmt.Errorf("invalid version %q", version)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	metaURL := npmRegistryBase + version
	resp, err := client.Get(metaURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %s", metaURL, resp.Status)
	}
	var pkg npmPackageVersion
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&pkg); err != nil {
		return fmt.Errorf("decode registry JSON: %w", err)
	}
	if pkg.Dist.Tarball == "" || !strings.HasPrefix(pkg.Dist.Tarball, "https://") {
		return fmt.Errorf("unexpected tarball URL %q", pkg.Dist.Tarball)
	}
	wantHash, err := parseSHA512Integrity(pkg.Dist.Integrity)
	if err != nil {
		return err
	}

	tarResp, err := client.Get(pkg.Dist.Tarball)
	if err != nil {
		return err
	}
	defer tarResp.Body.Close()
	if tarResp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %s", pkg.Dist.Tarball, tarResp.Status)
	}
	tgzBytes, err := readLimited(tarResp.Body, maxUnpackedTotal, "tarball")
	if err != nil {
		return err
	}
	gotHash := sha512.Sum512(tgzBytes)
	if !bytes.Equal(gotHash[:], wantHash) {
		return fmt.Errorf("tarball sha512 mismatch against %s", pkg.Dist.Integrity)
	}

	fetched, license, err := extractTarIcons(bytes.NewReader(tgzBytes))
	if err != nil {
		return err
	}

	lucideDir := filepath.Join(root, "icons", "internal", "lucide")
	if err := os.MkdirAll(lucideDir, 0o755); err != nil {
		return err
	}

	shapesPath := filepath.Join(lucideDir, "shapes.txt")
	keptPath := filepath.Join(lucideDir, "kept.txt")

	prevShapes, err := readIconMapIfExists(shapesPath)
	if err != nil {
		return err
	}
	prevKept, err := readIconMapIfExists(keptPath)
	if err != nil {
		return err
	}

	// Any icon in the previous shapes.txt or kept.txt that the new release
	// lacks stays in kept.txt; icons present in the new release leave kept.
	nextKept := map[string]iconEntry{}
	for name, e := range prevKept {
		if _, ok := fetched[name]; !ok {
			e.Deprecated = true
			nextKept[name] = e
		}
	}
	for name, e := range prevShapes {
		if _, ok := fetched[name]; !ok {
			e.Deprecated = true
			nextKept[name] = e
		}
	}

	// Validate naming and collisions before writing vendored files.
	if _, err := buildFunctions(fetched, nextKept); err != nil {
		return err
	}

	if err := writeIconMap(shapesPath, fetched); err != nil {
		return err
	}
	if err := writeIconMap(keptPath, nextKept); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(lucideDir, "LICENSE"), license, 0o644); err != nil {
		return err
	}
	versionText := fmt.Sprintf("version: %s\nintegrity: %s\n", version, pkg.Dist.Integrity)
	if err := os.WriteFile(filepath.Join(lucideDir, "VERSION"), []byte(versionText), 0o644); err != nil {
		return err
	}
	return nil
}

func parseSHA512Integrity(integrity string) ([]byte, error) {
	b64, ok := strings.CutPrefix(integrity, "sha512-")
	if !ok {
		return nil, fmt.Errorf("unsupported integrity %q: expected sha512-<base64>", integrity)
	}
	sum, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("invalid integrity base64: %w", err)
	}
	if len(sum) != sha512.Size {
		return nil, fmt.Errorf("invalid sha512 length %d", len(sum))
	}
	return sum, nil
}

func readLimited(r io.Reader, limit int64, what string) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds %d byte limit", what, limit)
	}
	return data, nil
}

func extractTarIcons(r io.Reader) (map[string]iconEntry, []byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	// Cap total decompressed stream at maxUnpackedTotal + 1 so tar headers
	// and payloads combined cannot exceed the unpacked limit.
	tr := tar.NewReader(io.LimitReader(gz, maxUnpackedTotal+1))

	icons := map[string]iconEntry{}
	var license []byte
	var totalUnpacked int64

	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read tar entry: %w", err)
		}
		if err := validateTarEntry(hdr); err != nil {
			return nil, nil, err
		}
		if hdr.Size < 0 {
			return nil, nil, fmt.Errorf("entry %q has negative size %d", hdr.Name, hdr.Size)
		}
		totalUnpacked += hdr.Size
		if totalUnpacked > maxUnpackedTotal {
			return nil, nil, fmt.Errorf("unpacked tarball exceeds %d byte limit", maxUnpackedTotal)
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		cleanName := path.Clean(hdr.Name)
		isLicense := cleanName == "package/LICENSE"
		rel, isIconRel := strings.CutPrefix(cleanName, "package/icons/")
		isIcon := isIconRel && !strings.Contains(rel, "/") && strings.HasSuffix(rel, ".svg")
		if !isLicense && !isIcon {
			continue
		}
		if hdr.Size > maxEntryBytes {
			return nil, nil, fmt.Errorf("entry %q size %d exceeds %d byte limit", hdr.Name, hdr.Size, maxEntryBytes)
		}
		if isLicense {
			b, err := readLimited(tr, maxEntryBytes, hdr.Name)
			if err != nil {
				return nil, nil, err
			}
			license = b
			continue
		}
		b, err := readLimited(tr, maxEntryBytes, hdr.Name)
		if err != nil {
			return nil, nil, err
		}
		shapes, err := parseSVG(cleanName, b)
		if err != nil {
			return nil, nil, err
		}
		name := strings.TrimSuffix(rel, ".svg")
		if _, dup := icons[name]; dup {
			return nil, nil, fmt.Errorf("duplicate icon %q in tarball", name)
		}
		icons[name] = iconEntry{Name: name, Shapes: shapes}
	}

	if len(icons) == 0 {
		return nil, nil, errors.New("tarball contained no package/icons/*.svg entries")
	}
	if len(license) == 0 {
		return nil, nil, errors.New("tarball contained no package/LICENSE")
	}
	return icons, license, nil
}

func validateTarEntry(hdr *tar.Header) error {
	name := hdr.Name
	if name == "" || strings.ContainsRune(name, '\x00') {
		return fmt.Errorf("invalid tar entry name %q", name)
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return fmt.Errorf("tar entry %q has absolute path", name)
	}
	if len(name) >= 2 && name[1] == ':' {
		return fmt.Errorf("tar entry %q has drive prefix", name)
	}
	for _, seg := range strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." {
			return fmt.Errorf("tar entry %q contains '..'", name)
		}
	}
	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeDir: // the reader reports TypeRegA as TypeReg
	case tar.TypeSymlink, tar.TypeLink:
		return fmt.Errorf("tar entry %q is a link, which is refused", name)
	default:
		return fmt.Errorf("tar entry %q has unsupported type %q", name, hdr.Typeflag)
	}
	return nil
}

const svgNS = "http://www.w3.org/2000/svg"

var (
	pathDataRE = regexp.MustCompile(`^[MmLlHhVvCcSsQqTtAaZz0-9.,+\- ]+$`)
	numberRE   = regexp.MustCompile(`^-?[0-9]*\.?[0-9]+$`)
	pointsRE   = regexp.MustCompile(`^[0-9.,+\- ]+$`)
	svgClassRE = regexp.MustCompile(`^[a-z0-9 -]+$`)
)

type elemRule struct {
	attrs []string // canonical attribute order
	check func(attr, val string) bool
}

func isNumber(_, val string) bool { return numberRE.MatchString(val) }

var shapeRules = map[string]elemRule{
	"path": {
		attrs: []string{"d"},
		check: func(attr, val string) bool { return attr == "d" && pathDataRE.MatchString(val) },
	},
	"circle": {
		attrs: []string{"cx", "cy", "r", "fill"},
		check: func(attr, val string) bool {
			if attr == "fill" {
				return val == "currentColor"
			}
			return numberRE.MatchString(val)
		},
	},
	"ellipse": {
		attrs: []string{"cx", "cy", "rx", "ry"},
		check: isNumber,
	},
	"line": {
		attrs: []string{"x1", "y1", "x2", "y2"},
		check: isNumber,
	},
	"polyline": {
		attrs: []string{"points"},
		check: validPoints,
	},
	"polygon": {
		attrs: []string{"points"},
		check: validPoints,
	},
	"rect": {
		attrs: []string{"x", "y", "width", "height", "rx", "ry"},
		check: isNumber,
	},
}

func validPoints(_, val string) bool {
	if !pointsRE.MatchString(val) {
		return false
	}
	fields := strings.FieldsFunc(val, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) == 0 {
		return false
	}
	for _, f := range fields {
		if !numberRE.MatchString(f) {
			return false
		}
	}
	return true
}

var svgRootAttrChecks = map[string]func(string) bool{
	"xmlns":           func(v string) bool { return v == svgNS },
	"width":           func(v string) bool { return v == "24" },
	"height":          func(v string) bool { return v == "24" },
	"viewBox":         func(v string) bool { return v == "0 0 24 24" },
	"fill":            func(v string) bool { return v == "none" },
	"stroke":          func(v string) bool { return v == "currentColor" },
	"stroke-width":    func(v string) bool { return v == "2" },
	"stroke-linecap":  func(v string) bool { return v == "round" },
	"stroke-linejoin": func(v string) bool { return v == "round" },
	"class":           func(v string) bool { return svgClassRE.MatchString(v) },
}

// parseSVG strictly parses and validates an SVG against §5.2's rules and
// returns the serialised child shapes joined by a single space.
func parseSVG(file string, data []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true

	var shapes []string
	depth := 0
	seenRoot := false

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%s: XML parse error: %w", file, err)
		}
		switch t := tok.(type) {
		case xml.Directive:
			return "", fmt.Errorf("%s: XML directive not allowed: <!%s>", file, string(t))
		case xml.ProcInst:
			return "", fmt.Errorf("%s: processing instruction <?%s?> not allowed", file, t.Target)
		case xml.Comment:
			// Lucide SVGs begin with a license comment.
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return "", fmt.Errorf("%s: unexpected text %q", file, strings.TrimSpace(string(t)))
			}
		case xml.StartElement:
			if t.Name.Space != "" && t.Name.Space != svgNS {
				return "", fmt.Errorf("%s: element %s:%s in disallowed namespace", file, t.Name.Space, t.Name.Local)
			}
			switch depth {
			case 0:
				if seenRoot {
					return "", fmt.Errorf("%s: multiple root elements", file)
				}
				if t.Name.Local != "svg" {
					return "", fmt.Errorf("%s: root element <%s>, want <svg>", file, t.Name.Local)
				}
				seenRoot = true
				if err := checkSVGRootAttrs(file, t.Attr); err != nil {
					return "", err
				}
			case 1:
				shape, err := parseShapeElem(file, t)
				if err != nil {
					return "", err
				}
				shapes = append(shapes, shape)
			default:
				return "", fmt.Errorf("%s: nested element <%s> not allowed", file, t.Name.Local)
			}
			depth++
		case xml.EndElement:
			depth--
		}
	}

	if !seenRoot {
		return "", fmt.Errorf("%s: missing <svg> root element", file)
	}
	if len(shapes) == 0 {
		return "", fmt.Errorf("%s: <svg> has no shape elements", file)
	}
	return strings.Join(shapes, " "), nil
}

func checkSVGRootAttrs(file string, attrs []xml.Attr) error {
	seen := map[string]bool{}
	for _, a := range attrs {
		if a.Name.Space != "" {
			return fmt.Errorf("%s: <svg> attribute %s:%s not allowed", file, a.Name.Space, a.Name.Local)
		}
		name := a.Name.Local
		if seen[name] {
			return fmt.Errorf("%s: duplicate <svg> attribute %q", file, name)
		}
		seen[name] = true
		check, ok := svgRootAttrChecks[name]
		if !ok {
			return fmt.Errorf("%s: unknown <svg> attribute %q", file, name)
		}
		if !check(a.Value) {
			return fmt.Errorf("%s: invalid <svg> attribute %s=%q", file, name, a.Value)
		}
	}
	if !seen["viewBox"] {
		return fmt.Errorf("%s: <svg> missing required attribute \"viewBox\"", file)
	}
	return nil
}

func parseShapeElem(file string, el xml.StartElement) (string, error) {
	rule, ok := shapeRules[el.Name.Local]
	if !ok {
		return "", fmt.Errorf("%s: disallowed element <%s>", file, el.Name.Local)
	}
	vals := map[string]string{}
	for _, a := range el.Attr {
		if a.Name.Space != "" {
			return "", fmt.Errorf("%s: <%s> attribute %s:%s not allowed", file, el.Name.Local, a.Name.Space, a.Name.Local)
		}
		name := a.Name.Local
		if _, dup := vals[name]; dup {
			return "", fmt.Errorf("%s: <%s> duplicate attribute %q", file, el.Name.Local, name)
		}
		if !slices.Contains(rule.attrs, name) {
			return "", fmt.Errorf("%s: <%s> disallowed attribute %q", file, el.Name.Local, name)
		}
		if !rule.check(name, a.Value) {
			return "", fmt.Errorf("%s: <%s> invalid attribute %s=%q", file, el.Name.Local, name, a.Value)
		}
		vals[name] = a.Value
	}
	if len(vals) == 0 {
		return "", fmt.Errorf("%s: <%s> has no attributes", file, el.Name.Local)
	}

	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(el.Name.Local)
	for _, name := range rule.attrs {
		if v, ok := vals[name]; ok {
			b.WriteByte(' ')
			b.WriteString(name)
			b.WriteString(`="`)
			b.WriteString(html.EscapeString(v))
			b.WriteByte('"')
		}
	}
	b.WriteString("></")
	b.WriteString(el.Name.Local)
	b.WriteByte('>')
	return b.String(), nil
}

type iconEntry struct {
	Name        string
	Shapes      string
	Deprecated  bool
	Replacement string // optional, from 3rd column of kept.txt
}

func readIconMapIfExists(p string) (map[string]iconEntry, error) {
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]iconEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]iconEntry{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || len(parts) > 3 {
			return nil, fmt.Errorf("%s:%d: expected <name>\\t<shapes>[\\t<replacement>]", p, lineNo)
		}
		name, shapes := parts[0], parts[1]
		if name == "" || shapes == "" {
			return nil, fmt.Errorf("%s:%d: empty name or shapes", p, lineNo)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate icon %q", p, lineNo, name)
		}
		e := iconEntry{Name: name, Shapes: shapes}
		if len(parts) == 3 {
			e.Replacement = parts[2]
		}
		out[name] = e
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func writeIconMap(p string, m map[string]iconEntry) error {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	slices.Sort(names)

	var b strings.Builder
	for _, n := range names {
		e := m[n]
		b.WriteString(e.Name)
		b.WriteByte('\t')
		b.WriteString(e.Shapes)
		if e.Replacement != "" {
			b.WriteByte('\t')
			b.WriteString(e.Replacement)
		}
		b.WriteByte('\n')
	}
	return os.WriteFile(p, []byte(b.String()), 0o644)
}

type generatedFunc struct {
	GoName      string
	LucideNames []string
	Shapes      string
	Mirror      bool
	Deprecated  bool
	Replacement string
}

func runGenerate(root string) error {
	lucideDir := filepath.Join(root, "icons", "internal", "lucide")
	active, err := readIconMapIfExists(filepath.Join(lucideDir, "shapes.txt"))
	if err != nil {
		return err
	}
	if len(active) == 0 {
		return errors.New("icons/internal/lucide/shapes.txt is empty or missing")
	}
	kept, err := readIconMapIfExists(filepath.Join(lucideDir, "kept.txt"))
	if err != nil {
		return err
	}
	for name := range kept {
		e := kept[name]
		e.Deprecated = true
		kept[name] = e
	}
	// The vendored files are checked again, as the release was: a hand edit
	// of shapes.txt or kept.txt must not reach the rendered SVG unchecked.
	for _, m := range []map[string]iconEntry{active, kept} {
		for name, e := range m {
			if err := recheckShapes(name, e.Shapes); err != nil {
				return err
			}
		}
	}

	funcs, err := buildFunctions(active, kept)
	if err != nil {
		return err
	}

	if err := writeLucideGo(filepath.Join(root, "icons", "lucide.go"), funcs); err != nil {
		return err
	}
	if err := writeIconListGo(filepath.Join(root, "internal", "iconlist", "iconlist.go"), funcs); err != nil {
		return err
	}
	return nil
}

func buildFunctions(active, kept map[string]iconEntry) ([]generatedFunc, error) {
	all := map[string]iconEntry{}
	for name, e := range kept {
		all[name] = e
	}
	for name, e := range active {
		if _, inKept := kept[name]; inKept {
			return nil, fmt.Errorf("icon %q appears in both shapes.txt and kept.txt", name)
		}
		all[name] = e
	}

	byGoName := map[string]*generatedFunc{}
	add := func(goName, lucideName, shapes string, mirror, deprecated bool, replacement string) error {
		if cur, exists := byGoName[goName]; exists {
			if cur.Shapes != shapes || cur.Mirror != mirror {
				return fmt.Errorf("name collision on %s: %v vs %q", goName, cur.LucideNames, lucideName)
			}
			if !slices.Contains(cur.LucideNames, lucideName) {
				cur.LucideNames = append(cur.LucideNames, lucideName)
				slices.Sort(cur.LucideNames)
			}
			if !deprecated {
				cur.Deprecated = false
				cur.Replacement = ""
			}
			return nil
		}
		byGoName[goName] = &generatedFunc{
			GoName:      goName,
			LucideNames: []string{lucideName},
			Shapes:      shapes,
			Mirror:      mirror,
			Deprecated:  deprecated,
			Replacement: replacement,
		}
		return nil
	}

	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	slices.Sort(names)

	for _, name := range names {
		e := all[name]
		goName, err := toGoName(name)
		if err != nil {
			return nil, fmt.Errorf("icon %q: %w", name, err)
		}
		if err := add(goName, name, e.Shapes, false, e.Deprecated, e.Replacement); err != nil {
			return nil, err
		}

		if seName, ok := pairedStartEndName(name, all); ok {
			seGoName, err := toGoName(seName)
			if err != nil {
				return nil, fmt.Errorf("paired icon %q (%s): %w", name, seName, err)
			}
			if err := add(seGoName, name, e.Shapes, true, e.Deprecated, e.Replacement); err != nil {
				return nil, err
			}
		}
	}

	out := make([]generatedFunc, 0, len(byGoName))
	for _, fn := range byGoName {
		out = append(out, *fn)
	}
	slices.SortFunc(out, func(a, b generatedFunc) int {
		return strings.Compare(a.GoName, b.GoName)
	})
	return out, nil
}

func pairedStartEndName(name string, all map[string]iconEntry) (string, bool) {
	for _, prefix := range notMirroredPrefixes {
		if strings.HasPrefix(name, prefix) {
			return "", false
		}
	}
	words := strings.Split(name, "-")
	hasLR := false
	swapped := make([]string, len(words))
	startEnd := make([]string, len(words))
	for i, w := range words {
		switch w {
		case "left":
			hasLR = true
			swapped[i] = "right"
			startEnd[i] = "start"
		case "right":
			hasLR = true
			swapped[i] = "left"
			startEnd[i] = "end"
		default:
			swapped[i] = w
			startEnd[i] = w
		}
	}
	if !hasLR {
		return "", false
	}
	partner := strings.Join(swapped, "-")
	if partner == name {
		return "", false
	}
	if _, ok := all[partner]; !ok {
		return "", false
	}
	return strings.Join(startEnd, "-"), true
}

func toGoName(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty icon name")
	}
	parts := strings.Split(name, "-")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			return "", errors.New("empty word in icon name")
		}
		for i := 0; i < len(p); i++ {
			c := p[i]
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				return "", fmt.Errorf("unexpected character %q in icon name", c)
			}
		}
		if p[0] >= 'a' && p[0] <= 'z' {
			b.WriteByte(p[0] - 'a' + 'A')
			b.WriteString(p[1:])
		} else {
			b.WriteString(p)
		}
	}
	id := b.String()
	if id == "" || (id[0] >= '0' && id[0] <= '9') {
		return "", fmt.Errorf("identifier %q starts with a digit", id)
	}
	if token.IsKeyword(id) {
		return "", fmt.Errorf("identifier %q collides with Go keyword", id)
	}
	switch {
	case id == "IconSpec", id == "Labelled", id == "More", id == "Warning",
		strings.HasPrefix(id, "Size"), strings.HasPrefix(id, "Color"):
		return "", fmt.Errorf("identifier %q collides with package icons identifier", id)
	}
	return id, nil
}

func writeLucideGo(dst string, funcs []generatedFunc) error {
	var b strings.Builder
	b.WriteString("// Code generated by icons/internal/gen; DO NOT EDIT.\n\n")
	b.WriteString("package icons\n\n")

	// Emit one unexported constant per unique shape string.
	constNames := map[string]string{}
	for _, fn := range funcs {
		if _, ok := constNames[fn.Shapes]; ok {
			continue
		}
		cName := "shapes" + fn.GoName
		constNames[fn.Shapes] = cName
		b.WriteString("const ")
		b.WriteString(cName)
		b.WriteString(" = ")
		b.WriteString(strconv.Quote(fn.Shapes))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')

	for _, fn := range funcs {
		cName := constNames[fn.Shapes]
		b.WriteString("// ")
		b.WriteString(fn.GoName)
		b.WriteString(" is Lucide's ")
		for i, ln := range fn.LucideNames {
			if i > 0 {
				b.WriteString(" / ")
			}
			b.WriteString(strconv.Quote(ln))
		}
		if fn.Mirror {
			b.WriteString(" icon, mirrored in RTL.\n")
		} else {
			b.WriteString(" icon.\n")
		}
		if fn.Deprecated {
			if fn.Replacement != "" {
				rep := fn.Replacement
				if goRep, err := toGoName(fn.Replacement); err == nil {
					rep = goRep
				}
				fmt.Fprintf(&b, "//\n// Deprecated: use %s instead.\n", rep)
			} else {
				b.WriteString("//\n// Deprecated: removed from Lucide; kept for compatibility.\n")
			}
		}
		if fn.Mirror {
			fmt.Fprintf(&b, "func %s() IconSpec { return IconSpec{shapes: %s, mirror: true} }\n\n", fn.GoName, cName)
		} else {
			fmt.Fprintf(&b, "func %s() IconSpec { return IconSpec{shapes: %s} }\n\n", fn.GoName, cName)
		}
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("format lucide.go: %w", err)
	}
	return os.WriteFile(dst, formatted, 0o644)
}

func writeIconListGo(dst string, funcs []generatedFunc) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("// Code generated by icons/internal/gen; DO NOT EDIT.\n\n")
	b.WriteString("package iconlist\n\n")
	b.WriteString("import \"github.com/epistemer/ui/icons\"\n\n")
	b.WriteString("// Icon names one generated icon in package icons.\n")
	b.WriteString("type Icon struct {\n\tName string\n\tSpec func() icons.IconSpec\n}\n\n")
	b.WriteString("// All holds every generated icon in alphabetical order by Go name.\n")
	b.WriteString("var All = []Icon{\n")
	for _, fn := range funcs {
		fmt.Fprintf(&b, "\t{Name: %s, Spec: icons.%s},\n", strconv.Quote(fn.GoName), fn.GoName)
	}
	b.WriteString("}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("format iconlist.go: %w", err)
	}
	return os.WriteFile(dst, formatted, 0o644)
}

// recheckShapes parses a vendored icon's shapes inside a bare root, with
// the same checks as a release's file, and requires that they serialise
// back to exactly what is stored.
func recheckShapes(name, shapes string) error {
	doc := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">` + shapes + `</svg>`
	got, err := parseSVG(name, []byte(doc))
	if err != nil {
		return fmt.Errorf("vendored icon %w", err)
	}
	if got != shapes {
		return fmt.Errorf("vendored icon %s: shapes don't round-trip: %q", name, shapes)
	}
	return nil
}
