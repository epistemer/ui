package forms_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/listing"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/forms/formstest"
	"github.com/epistemer/ui/forms/rule"
)

type panicLabeler struct{}

func (panicLabeler) LabelOf(context.Context, string) (string, bool, error) {
	panic("forms must never call Labeler")
}

func TestRemoteFields(t *testing.T) {
	search := testLink("/customers/search")
	lbl := panicLabeler{}

	t.Run("supported types and Control.Remote before and after Bind", func(t *testing.T) {
		f := forms.New()
		txt := f.Text("txt").Remote(search, lbl)
		num := f.Int("num").Remote(search, lbl).Is(rule.Min(10))
		i64 := f.Number[int64]("i64").Remote(search, lbl)
		u32 := f.Number[uint32]("u32").Remote(search, lbl)
		prs := f.Parsed[code]("prs").Remote(search, lbl)
		plain := f.Text("plain")

		if _, _, _, ok := plain.Control().Remote(); ok {
			t.Error("plain text field reported Remote ok")
		}
		if _, _, _, ok := f.Control().Remote(); ok {
			t.Error("form control reported Remote ok")
		}
		var zero forms.Control
		if _, _, _, ok := zero.Remote(); ok {
			t.Error("zero control reported Remote ok")
		}

		// Before Bind, unset: id is "".
		if s, l, id, ok := num.Control().Remote(); !ok || s != search || l != lbl || id != "" {
			t.Errorf("unset Remote: search=%v l=%v id=%q ok=%v", s, l, id, ok)
		}

		// Before Bind, Set: id is formatted value.
		num.Set(7)
		prs.Set(code{"abc"})
		if _, _, id, ok := num.Control().Remote(); !ok || id != "7" {
			t.Errorf("Set int Remote: id=%q ok=%v", id, ok)
		}
		if _, _, id, ok := prs.Control().Remote(); !ok || id != "ABC" {
			t.Errorf("Set parsed Remote: id=%q ok=%v", id, ok)
		}

		// Attrs: integer remote fields have inputmode="numeric", text ones don't.
		for _, c := range []forms.Control{num.Control(), i64.Control(), u32.Control()} {
			attrs := collectAttrs(c)
			if attrs["inputmode"] != "numeric" {
				t.Errorf("%s attrs = %v, want inputmode=numeric", c.Path(), attrs)
			}
		}
		for _, c := range []forms.Control{txt.Control(), prs.Control()} {
			attrs := collectAttrs(c)
			if _, has := attrs["inputmode"]; has {
				t.Errorf("%s attrs = %v, unexpected inputmode", c.Path(), attrs)
			}
		}

		// After Bind: leading zeros format to canonical integer id; rule failure keeps id.
		formstest.Submit(t, bg, f, "txt=+uuid-1+&num=007&i64=42&u32=9&prs=egp")
		if !errors.Is(num.Err(), rule.ErrMin) {
			t.Errorf("num err = %v, want ErrMin", num.Err())
		}
		if _, _, id, _ := num.Control().Remote(); id != "7" {
			t.Errorf("num id after rule failure = %q, want 7", id)
		}
		if _, _, id, _ := txt.Control().Remote(); id != "uuid-1" {
			t.Errorf("txt id = %q, want uuid-1", id)
		}
		if _, _, id, _ := i64.Control().Remote(); id != "42" {
			t.Errorf("i64 id = %q, want 42", id)
		}
		if _, _, id, _ := u32.Control().Remote(); id != "9" {
			t.Errorf("u32 id = %q, want 9", id)
		}
		if _, _, id, _ := prs.Control().Remote(); id != "EGP" {
			t.Errorf("prs id = %q, want EGP", id)
		}
	})

	t.Run("unconverted text gives empty id while Control.Text keeps it", func(t *testing.T) {
		f := forms.New()
		num := f.Number[int64]("customer_id").Remote(search, lbl)
		prs := f.Parsed[code]("code_id").Remote(search, lbl)
		abs := f.Int("absent_id").Remote(search, lbl)
		gate := f.Checkbox("gate")
		inact := f.Int("inact_id").Remote(search, lbl).When(gate.Filled())

		formstest.Submit(t, bg, f, "customer_id=abc&code_id=toolong&inact_id=5")
		if _, _, id, ok := num.Control().Remote(); !ok || id != "" {
			t.Errorf("unconverted int: id=%q ok=%v, want empty id", id, ok)
		}
		if !slices.Equal(num.Control().Text(), []string{"abc"}) {
			t.Errorf("unconverted int Text() = %v, want [abc]", num.Control().Text())
		}
		if _, _, id, ok := prs.Control().Remote(); !ok || id != "" {
			t.Errorf("unconverted parsed: id=%q ok=%v, want empty id", id, ok)
		}
		if !slices.Equal(prs.Control().Text(), []string{"toolong"}) {
			t.Errorf("unconverted parsed Text() = %v, want [toolong]", prs.Control().Text())
		}
		if _, _, id, _ := abs.Control().Remote(); id != "" {
			t.Errorf("absent id = %q, want empty", id)
		}
		if _, _, id, _ := inact.Control().Remote(); id != "" {
			t.Errorf("inactive id = %q, want empty", id)
		}
	})
}

func collectAttrs(c forms.Control) map[string]string {
	m := map[string]string{}
	for k, v := range c.Attrs() {
		m[k] = v
	}
	return m
}

type multipartBuilder struct {
	buf bytes.Buffer
	w   *multipart.Writer
}

func newMultipart() *multipartBuilder {
	b := &multipartBuilder{}
	b.w = multipart.NewWriter(&b.buf)
	return b
}

func (b *multipartBuilder) field(name, val string) *multipartBuilder {
	if err := b.w.WriteField(name, val); err != nil {
		panic(err)
	}
	return b
}

func (b *multipartBuilder) file(name, filename, content string) *multipartBuilder {
	fw, err := b.w.CreateFormFile(name, filename)
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		panic(err)
	}
	return b
}

func (b *multipartBuilder) request() *http.Request {
	if err := b.w.Close(); err != nil {
		panic(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/", &b.buf)
	r.Header.Set("Content-Type", b.w.FormDataContentType())
	return r
}

func TestUploadAndUploads(t *testing.T) {
	s1 := files.Stored{Key: "k1", Name: "one.pdf", Href: testLink("/files/k1")}
	s2 := files.Stored{Key: "k2", Name: "two.png", Href: testLink("/files/k2"), Image: true}

	t.Run("before Bind value is a copy of stored and Stored yields true", func(t *testing.T) {
		callerSlice := []files.Stored{s1, s2}
		f := forms.New()
		emptyUp := f.Upload("empty_up", nil)
		fullUp := f.Upload("full_up", []files.Stored{s1})
		ups := f.Uploads("ups", callerSlice)

		// Mutate caller's slice after declaration.
		callerSlice[0].Name = "mutated"

		if _, ok := emptyUp.Get(); ok {
			t.Error("Upload with no stored files should be absent before Bind")
		}
		if v, ok := fullUp.Get(); !ok || len(v.Kept) != 1 || v.Kept[0].Key != "k1" {
			t.Errorf("Upload with 1 stored file before Bind: %+v, ok=%v", v, ok)
		}
		v, ok := ups.Get()
		if !ok || len(v.Kept) != 2 || v.Kept[0].Name != "one.pdf" || v.Kept[1].Key != "k2" {
			t.Errorf("Uploads before Bind: %+v, ok=%v", v, ok)
		}
		if ups.Control().Text() != nil || fullUp.Control().Text() != nil {
			t.Error("Control.Text() of an upload field must be nil")
		}

		var gotStored []string
		for s, kept := range ups.Control().Stored() {
			if !kept {
				t.Errorf("before Bind %s kept = false", s.Key)
			}
			gotStored = append(gotStored, s.Key+":"+s.Name)
		}
		if !slices.Equal(gotStored, []string{"k1:one.pdf", "k2:two.png"}) {
			t.Errorf("Stored() before Bind = %v", gotStored)
		}

		// Break early in Stored() iterator and check non-upload controls return empty non-nil iterator.
		count := 0
		for range ups.Control().Stored() {
			count++
			break
		}
		if count != 1 {
			t.Errorf("break in Stored() ran %d times", count)
		}
		for _, c := range []forms.Control{emptyUp.Control(), f.Text("t").Control(), f.Control(), {}} {
			seq := c.Stored()
			if seq == nil {
				t.Errorf("%s Stored() is nil", c.Path())
			}
			for range seq {
				t.Errorf("%s Stored() yielded an item", c.Path())
			}
		}
	})

	t.Run("Uploads binding drops unknown keys, deduplicates, preserves stored order", func(t *testing.T) {
		f := forms.New()
		ups := f.Uploads("docs", []files.Stored{s1, s2}).Required().
			Is(rule.MinFiles(2), rule.MaxFiles(3), rule.Files(rule.MaxSize(1024), rule.ContentType("text/plain")))

		attrs := collectAttrs(ups.Control())
		if attrs["data-ui-max-size"] != "1024" || attrs["accept"] != "text/plain" {
			t.Errorf("upload attrs = %v", attrs)
		}

		// Post k2 before k1, repeat k2, add unknown k99, and upload one new file + one empty part.
		r := newMultipart().
			field("docs.keep", "k2").
			field("docs.keep", "k99").
			field("docs.keep", "k1").
			field("docs.keep", "k2").
			file("docs", "added.txt", "hello").
			file("docs", "", "").
			request()

		if !f.Submit(bg, r) {
			t.Fatalf("Submit failed: %v", f.Err())
		}
		val, ok := ups.Get()
		if !ok || len(val.Kept) != 2 || val.Kept[0].Key != "k1" || val.Kept[1].Key != "k2" {
			t.Errorf("Kept = %+v, ok=%v (want [k1 k2] in stored order)", val.Kept, ok)
		}
		if len(val.Added) != 1 || val.Added[0].Filename != "added.txt" {
			t.Errorf("Added = %+v", val.Added)
		}
	})

	t.Run("Upload single-file takes last posted file and empties Kept while Control.Stored reports posted keep", func(t *testing.T) {
		f := forms.New()
		up := f.Upload("receipt", []files.Stored{s1}).Required()
		other := f.Text("other").Required() // will fail so we inspect redisplay state

		r := newMultipart().
			field("receipt.keep", "k1").
			file("receipt", "first.txt", "111").
			file("receipt", "second.txt", "222").
			request()

		if f.Submit(bg, r) {
			t.Fatal("expected other field to fail")
		}
		_ = other
		val, ok := up.Get()
		if !ok {
			t.Fatal("receipt should be present")
		}
		if len(val.Kept) != 0 {
			t.Errorf("Added file must empty Kept in value, got %+v", val.Kept)
		}
		if len(val.Added) != 1 || val.Added[0].Filename != "second.txt" {
			t.Errorf("Upload must take the last posted file, got %+v", val.Added)
		}
		// Control.Stored still reports the posted keep state (true for k1).
		var states []bool
		for _, kept := range up.Control().Stored() {
			states = append(states, kept)
		}
		if !slices.Equal(states, []bool{true}) {
			t.Errorf("Control.Stored() keep states = %v, want [true]", states)
		}
	})

	t.Run("inactive conditional upload resets keep state so all stored files re-render kept", func(t *testing.T) {
		f := forms.New()
		gate := f.Checkbox("gate")
		other := f.Text("other").Required()
		ups := f.Uploads("docs", []files.Stored{s1, s2}).When(gate.Filled())
		_ = other

		// gate is unticked, so docs is inactive and posts no docs.keep; other fails so the form re-renders.
		if formstest.Submit(t, bg, f, "") {
			t.Fatal("expected other field to fail")
		}
		if _, ok := ups.Get(); ok {
			t.Error("inactive upload must have no value")
		}
		if ups.Control().Lost() {
			t.Error("inactive upload must not report Lost")
		}
		var states []bool
		for _, kept := range ups.Control().Stored() {
			states = append(states, kept)
		}
		if !slices.Equal(states, []bool{true, true}) {
			t.Errorf("inactive upload Control.Stored() keep states = %v, want [true true]", states)
		}
	})

	t.Run("presence and Required with kept only, added only, or neither", func(t *testing.T) {
		// Kept only satisfies Required.
		f1 := forms.New()
		u1 := f1.Upload("u", []files.Stored{s1}).Required()
		if !formstest.Submit(t, bg, f1, "u.keep=k1") {
			t.Errorf("kept file should satisfy Required: %v", f1.Err())
		}
		if len(u1.Value().Kept) != 1 || len(u1.Value().Added) != 0 {
			t.Errorf("u1 value = %+v", u1.Value())
		}

		// Added only satisfies Required.
		f2 := forms.New()
		u2 := f2.Upload("u", []files.Stored{s1}).Required()
		r2 := newMultipart().file("u", "new.txt", "abc").request()
		if !f2.Submit(bg, r2) {
			t.Errorf("added file should satisfy Required: %v", f2.Err())
		}
		if len(u2.Value().Kept) != 0 || len(u2.Value().Added) != 1 {
			t.Errorf("u2 value = %+v", u2.Value())
		}
		for _, kept := range u2.Control().Stored() {
			if kept {
				t.Error("u.keep was not posted; Control.Stored keep state should be false")
			}
		}

		// Neither kept nor added is absent and fails Required.
		f3 := forms.New()
		u3 := f3.Uploads("u", []files.Stored{s1, s2}).Required()
		if formstest.Submit(t, bg, f3, "u.keep=unknown") {
			t.Error("expected Required failure when no valid keep key and no added file")
		}
		if _, ok := u3.Get(); ok || !errors.Is(u3.Err(), forms.ErrRequired) {
			t.Errorf("u3 present=%v err=%v", ok, u3.Err())
		}
	})
}

func TestControlLost(t *testing.T) {
	s1 := files.Stored{Key: "k1", Name: "one.pdf"}

	t.Run("all four file kinds report Lost when posted files and form fails elsewhere", func(t *testing.T) {
		f := forms.New()
		name := f.Text("name").Required()
		fl := f.File("fl")
		fls := f.Files("fls")
		up := f.Upload("up", []files.Stored{s1})
		ups := f.Uploads("ups", nil)
		keptOnly := f.Upload("kept_only", []files.Stored{s1})
		inList := f.List("items", func(g *forms.Form) *forms.Field[*multipart.FileHeader] {
			return g.File("photo")
		})

		// Before Bind, Lost is false.
		if fl.Control().Lost() || up.Control().Lost() {
			t.Error("Lost() must be false before Bind")
		}

		r := newMultipart().
			file("fl", "a.txt", "a").
			file("fls", "b.txt", "b").
			file("up", "c.txt", "c").
			file("ups", "d.txt", "d").
			field("kept_only.keep", "k1").
			file("items[0].photo", "e.txt", "e").
			request()

		// Missing required "name" fails the form.
		if f.Submit(bg, r) {
			t.Fatal("expected Submit to fail on name")
		}
		_ = name
		for _, c := range []forms.Control{
			fl.Control(),
			fls.Control(),
			up.Control(),
			ups.Control(),
			inList.Items()[0].Control(),
		} {
			if !c.Lost() {
				t.Errorf("%s: Lost() = false, want true", c.Path())
			}
		}
		if keptOnly.Control().Lost() {
			t.Error("kept_only had no added file; Lost() must be false")
		}
		if f.Control().Lost() || name.Control().Lost() || (forms.Control{}).Lost() {
			t.Error("non-file control reported Lost()")
		}
	})

	t.Run("Lost is false when form succeeds, and turns true if Fail is called after", func(t *testing.T) {
		f := forms.New()
		up := f.Upload("up", nil)
		other := f.Text("other")
		r := newMultipart().file("up", "a.txt", "a").field("other", "ok").request()
		if !f.Submit(bg, r) {
			t.Fatalf("unexpected error: %v", f.Err())
		}
		if up.Control().Lost() {
			t.Error("Lost() must be false when form is valid")
		}
		// Fail on another field or on the root form makes Lost() true dynamically.
		other.Fail(errTaken)
		if !up.Control().Lost() {
			t.Error("Lost() must be true after another field Fails")
		}

		f2 := forms.New()
		up2 := f2.File("up")
		r2 := newMultipart().file("up", "a.txt", "a").request()
		if !f2.Submit(bg, r2) {
			t.Fatal(f2.Err())
		}
		f2.Fail(errTaken)
		if !up2.Control().Lost() {
			t.Error("Lost() must be true after root Form.Fail")
		}
	})

	t.Run("Lost is false when the file field itself has an error or is inactive", func(t *testing.T) {
		f := forms.New()
		f.Text("name").Required()
		badFile := f.File("bad").Is(rule.MaxSize(1))
		gate := f.Checkbox("gate")
		inactiveFile := f.Upload("inact", nil).When(gate.Filled())

		r := newMultipart().
			file("bad", "big.txt", "too big").
			file("inact", "hidden.txt", "x").
			request()

		f.Submit(bg, r)
		if badFile.Control().Lost() {
			t.Error("a file field with its own error must not report Lost()")
		}
		if inactiveFile.Control().Lost() {
			t.Error("an inactive file field must not report Lost()")
		}
	})
}

func TestListBoundsAndBlankItems(t *testing.T) {
	t.Run("Bounds", func(t *testing.T) {
		f := forms.New()
		def := f.List("def", func(g *forms.Form) *forms.Field[string] { return g.Text("v") })
		bounded := f.List("bounded", func(g *forms.Form) *forms.Field[string] { return g.Text("v") }).MinItems(2).MaxItems(5)
		highMin := f.List("high", func(g *forms.Form) *forms.Field[string] { return g.Text("v") }).MinItems(300)
		txt := f.Text("txt")

		if min, max := def.Control().Bounds(); min != 0 || max != forms.DefaultMaxItems {
			t.Errorf("default Bounds = (%d, %d), want (0, %d)", min, max, forms.DefaultMaxItems)
		}
		if min, max := bounded.Control().Bounds(); min != 2 || max != 5 {
			t.Errorf("bounded Bounds = (%d, %d), want (2, 5)", min, max)
		}
		if min, max := highMin.Control().Bounds(); min != 300 || max != 300 {
			t.Errorf("highMin Bounds = (%d, %d), want (300, 300)", min, max)
		}
		for _, c := range []forms.Control{txt.Control(), f.Control(), {}} {
			if min, max := c.Bounds(); min != 0 || max != 0 {
				t.Errorf("%s Bounds = (%d, %d), want (0, 0)", c.Path(), min, max)
			}
		}
	})

	t.Run("MinItems blank items on a new list", func(t *testing.T) {
		f := forms.New()
		l := f.List("lines", func(g *forms.Form) *forms.Field[string] {
			return g.Text("sku").Label(i18n.Static("SKU"))
		}).Label(i18n.Static("Lines")).MinItems(2)

		var childPaths []string
		for item := range l.Control().Children() {
			for field := range item.Children() {
				childPaths = append(childPaths, field.Path())
			}
		}
		if !slices.Equal(childPaths, []string{"lines[0].sku", "lines[1].sku"}) {
			t.Errorf("blank Children paths = %v, want [lines[0].sku lines[1].sku]", childPaths)
		}
		// Items() stays empty.
		if len(l.Items()) != 0 {
			t.Errorf("Items() len = %d, want 0", len(l.Items()))
		}
		// AssertLabelled still checks the list's members once.
		formstest.AssertLabelled(t, f)

		// Add appends at index 0 and replaces the blank items in Control().Children().
		added := l.Add()
		added.Set("A-1")
		if len(l.Items()) != 1 || l.Items()[0].Value() != "A-1" {
			t.Fatalf("after Add: len=%d", len(l.Items()))
		}
		var afterAdd []string
		for item := range l.Control().Children() {
			for field := range item.Children() {
				afterAdd = append(afterAdd, field.Path()+"="+strings.Join(field.Text(), ""))
			}
		}
		if !slices.Equal(afterAdd, []string{"lines[0].sku=A-1"}) {
			t.Errorf("Children after Add = %v, want [lines[0].sku=A-1]", afterAdd)
		}

		// After Bind with no items posted, Control().Children() is empty and Validate reports ErrMinItems.
		f2 := forms.New()
		l2 := f2.List("lines", func(g *forms.Form) *forms.Field[string] { return g.Text("sku") }).MinItems(2)
		if formstest.Submit(t, bg, f2, "") {
			t.Fatal("expected ErrMinItems")
		}
		if !errors.Is(l2.Err(), forms.ErrMinItems) {
			t.Errorf("l2.Err() = %v, want ErrMinItems", l2.Err())
		}
		for range l2.Control().Children() {
			t.Error("bound empty list must yield no children")
		}
	})

	t.Run("row posting nothing when empty is dropped on bind before rules run", func(t *testing.T) {
		type row struct {
			Box  *forms.Field[bool]
			Tags *forms.Field[[]string]
		}
		f := forms.New()
		l := f.List("rows", func(g *forms.Form) row {
			return row{
				Box:  g.Checkbox("box").Required(),
				Tags: g.Texts("tags").Options(color("red")).Required(),
			}
		})
		// Row 0 posts nothing; row 1 posts box=true&tags=red.
		if !formstest.Submit(t, bg, f, "rows[1].box=true&rows[1].tags=red") {
			t.Errorf("unexpected error: %v", f.Err())
		}
		if len(l.Items()) != 1 {
			t.Errorf("len(Items()) = %d, want 1", len(l.Items()))
		}
	})
}

func TestSubmitOversizedBody(t *testing.T) {
	t.Run("multipart MaxBytesError wrapped by ErrUnreadable", func(t *testing.T) {
		r := newMultipart().
			file("doc", "huge.bin", strings.Repeat("x", 4096)).
			request()
		w := httptest.NewRecorder()
		r.Body = http.MaxBytesReader(w, r.Body, 256)

		f := forms.New()
		f.Upload("doc", nil)
		if f.Submit(bg, r) {
			t.Fatal("expected Submit to fail on oversized multipart body")
		}
		if !errors.Is(f.Err(), forms.ErrUnreadable) {
			t.Errorf("Err() = %v, want ErrUnreadable", f.Err())
		}
		var big *http.MaxBytesError
		if !errors.As(f.Err(), &big) || big.Limit != 256 {
			t.Errorf("errors.As MaxBytesError = %v (limit %d) from %v", big != nil, func() int64 {
				if big != nil {
					return big.Limit
				}
				return 0
			}(), f.Err())
		}
	})

	t.Run("urlencoded MaxBytesError wrapped by ErrUnreadable", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name="+strings.Repeat("a", 1024)))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		r.Body = http.MaxBytesReader(w, r.Body, 64)

		f := forms.New()
		f.Text("name")
		if f.Submit(bg, r) {
			t.Fatal("expected Submit to fail on oversized urlencoded body")
		}
		if !errors.Is(f.Err(), forms.ErrUnreadable) {
			t.Errorf("Err() = %v, want ErrUnreadable", f.Err())
		}
		var big *http.MaxBytesError
		if !errors.As(f.Err(), &big) || big.Limit != 64 {
			t.Errorf("errors.As MaxBytesError failed on %v", f.Err())
		}
	})
}

func TestU1Panics(t *testing.T) {
	search := testLink("/search")
	lbl := testLabeler{}
	s1 := files.Stored{Key: "k1", Name: "one.pdf"}
	s2 := files.Stored{Key: "k2", Name: "two.pdf"}

	bound := forms.New()
	boundText := bound.Text("t")
	boundUp := bound.Upload("u", nil)
	bound.Bind(forms.FromValues(nil))

	var nilUploadPtr *files.Upload
	var nilLink interface{ URL() string }
	var nilLabeler listing.Labeler

	// File and Files inside a List item are legal.
	fLegal := forms.New()
	fLegal.List("legal", func(g *forms.Form) int {
		g.File("f")
		g.Files("fs")
		return 0
	})

	for _, tc := range []struct {
		name string
		fn   func()
		want string
	}{
		{
			"Upload with len(stored) > 1",
			func() { forms.New().Upload("doc", []files.Stored{s1, s2}) },
			`forms: Upload "doc" takes at most one stored file, got 2; use Uploads`,
		},
		{
			"Upload with empty Key",
			func() { forms.New().Upload("doc", []files.Stored{{Key: "", Name: "a.pdf"}}) },
			`forms: Upload "doc": stored file 0 has an empty Key`,
		},
		{
			"Uploads with empty Key",
			func() { forms.New().Uploads("docs", []files.Stored{s1, {Key: "", Name: "b.pdf"}}) },
			`forms: Uploads "docs": stored file 1 has an empty Key`,
		},
		{
			"Upload with empty Name",
			func() { forms.New().Upload("doc", []files.Stored{{Key: "k1", Name: ""}}) },
			`forms: Upload "doc": stored file "k1" has an empty Name`,
		},
		{
			"Uploads with repeated Key",
			func() { forms.New().Uploads("docs", []files.Stored{s1, {Key: "k1", Name: "dup.pdf"}}) },
			`forms: Uploads "docs" lists the stored file key "k1" twice`,
		},
		{
			"Upload inside a List item",
			func() {
				forms.New().List("rows", func(g *forms.Form) any {
					return g.Upload("doc", nil)
				})
			},
			`forms: Upload "doc" inside a list item; use File or Files`,
		},
		{
			"Uploads inside a Group inside a List item",
			func() {
				forms.New().List("rows", func(g *forms.Form) any {
					return g.Group("grp", func(h *forms.Form) any {
						return h.Uploads("docs", nil)
					})
				})
			},
			`forms: Uploads "docs" inside a list item; use File or Files`,
		},
		{
			"Set on Upload",
			func() { forms.New().Upload("doc", nil).Set(files.Upload{}) },
			`forms: Set on upload field "doc"; pass stored files to Upload or Uploads`,
		},
		{
			"SetPtr on Uploads",
			func() { forms.New().Uploads("docs", nil).SetPtr(nilUploadPtr) },
			`forms: SetPtr on upload field "docs"; pass stored files to Upload or Uploads`,
		},
		{
			"Options on Upload",
			func() { forms.New().Upload("doc", nil).Options(color("red")) },
			`forms: Options on upload field "doc"`,
		},
		{
			"Options on Uploads",
			func() { forms.New().Uploads("docs", nil).Options(color("red")) },
			`forms: Options on upload field "docs"`,
		},
		{
			"Options on File",
			func() { forms.New().File("f").Options(color("red")) },
			`forms: Options on file field "f"`,
		},
		{
			"Options on Files",
			func() { forms.New().Files("fs").Options(color("red")) },
			`forms: Options on file field "fs"`,
		},
		{
			"Hint on Upload",
			func() { forms.New().Upload("doc", nil).Hint(forms.KindFile) },
			`forms: Hint on upload field "doc"`,
		},
		{
			"Hint on Uploads",
			func() { forms.New().Uploads("docs", nil).Hint(forms.KindFiles) },
			`forms: Hint on upload field "docs"`,
		},
		{
			"Hint with KindRemote",
			func() { forms.New().Text("a").Hint(forms.KindRemote) },
			`forms: Hint(16) on field "a": not a hintable field kind`,
		},
		{
			"Hint with KindUpload",
			func() { forms.New().Text("a").Hint(forms.KindUpload) },
			`forms: Hint(17) on field "a": not a hintable field kind`,
		},
		{
			"Hint with KindUploads",
			func() { forms.New().Text("a").Hint(forms.KindUploads) },
			`forms: Hint(18) on field "a": not a hintable field kind`,
		},
		{
			"Remote with nil search link",
			func() { forms.New().Text("a").Remote(nilLink, lbl) },
			`forms: Remote on field "a" with a nil search link`,
		},
		{
			"Remote with nil Labeler",
			func() { forms.New().Text("a").Remote(search, nilLabeler) },
			`forms: Remote on field "a" with a nil Labeler`,
		},
		{
			"Remote after Hint",
			func() { forms.New().Text("a").Hint(forms.KindText).Remote(search, lbl) },
			`forms: Remote on field "a" after Hint`,
		},
		{
			"Hint after Remote",
			func() { forms.New().Text("a").Remote(search, lbl).Hint(forms.KindText) },
			`forms: Hint on remote field "a"`,
		},
		{
			"Remote after Options",
			func() { forms.New().Text("a").Options(color("red")).Remote(search, lbl) },
			`forms: Remote on field "a" after Options`,
		},
		{
			"Options after Remote",
			func() { forms.New().Text("a").Remote(search, lbl).Options(color("red")) },
			`forms: Options on remote field "a"`,
		},
		{
			"Remote on Float",
			func() { forms.New().Float("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Number[float64]",
			func() { forms.New().Number[float64]("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Texts",
			func() { forms.New().Texts("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Bool",
			func() { forms.New().Bool("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Checkbox",
			func() { forms.New().Checkbox("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Date",
			func() { forms.New().Date("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Time",
			func() { forms.New().Time("a", "15:04").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Enum",
			func() { forms.New().Enum("a", []color{"red"}).Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on File",
			func() { forms.New().File("a").Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote on Upload",
			func() { forms.New().Upload("a", nil).Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote twice",
			func() { forms.New().Int("a").Remote(search, lbl).Remote(search, lbl) },
			`forms: Remote on field "a": requires a Text, Parsed or integer Number field`,
		},
		{
			"Remote after Bind",
			func() { boundText.Remote(search, lbl) },
			`forms: Remote on field "t" after Bind`,
		},
		{
			"Set on Upload after Bind",
			func() { boundUp.Set(files.Upload{}) },
			`forms: Set on field "u" after Bind`,
		},
		{
			"Eq on Upload",
			func() { forms.New().Upload("u", nil).Eq(files.Upload{}) },
			`forms: Eq on field "u", whose value is a list or a file: compare a single value`,
		},
		{
			"In on Uploads",
			func() { forms.New().Uploads("u", nil).In(files.Upload{}) },
			`forms: In on field "u", whose value is a list or a file: compare a single value`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("did not panic; want %q", tc.want)
				}
				if got, ok := r.(string); !ok || got != tc.want {
					t.Errorf("panic = %q, want %q", r, tc.want)
				}
			}()
			tc.fn()
		})
	}
}
