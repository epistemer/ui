package inputs

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/forms/files"
	"github.com/epistemer/ui/forms/rule"
	"github.com/epistemer/ui/internal/themehook"
	_ "github.com/epistemer/ui/theme"
	"github.com/epistemer/ui/uitest"
)

var errAddress = errors.Key("errors.shop.address", "the address is incomplete", errors.Invalid)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var b strings.Builder
	ctx := ui.WithIDs(context.Background(), ui.Sequence("t"))
	ctx = actions.Gate.With(ctx, actions.AllowAll)
	if err := c.Render(ctx, &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

type countLabeler struct {
	names map[string]string
	calls int
	err   error
}

func (l *countLabeler) LabelOf(_ context.Context, id string) (string, bool, error) {
	l.calls++
	if l.err != nil {
		return "", false, l.err
	}
	name, ok := l.names[id]
	return name, ok, nil
}

// funcLabeler has a slice in its dynamic value, so reflect.TypeOf(l).Comparable() is false.
type funcLabeler func(context.Context, string) (string, bool, error)

func (f funcLabeler) LabelOf(ctx context.Context, id string) (string, bool, error) {
	return f(ctx, id)
}

func firstControl(f *forms.Form) forms.Control {
	for c := range f.Controls() {
		return c
	}
	panic("empty form")
}

func checkAll(t *testing.T, html string) {
	t.Helper()
	uitest.CheckIDs(t, html)
	uitest.CheckNames(t, html)
	uitest.CheckSelectors(t, html, string(themehook.Stylesheet()))
}

func TestHiddenTakesAttrs(t *testing.T) {
	f := forms.New()
	f.Text("ref").Hint(forms.KindHidden)
	c := firstControl(f)
	html := render(t, Field(c).Attrs(templ.Attributes{"form": "other", "name": "stolen"}))
	if !strings.Contains(html, `form="other"`) || strings.Contains(html, "stolen") || !strings.Contains(html, `name="ref"`) {
		t.Errorf("hidden field: %s", html)
	}
}

func TestGroupErrorIsDescribed(t *testing.T) {
	f := forms.New()
	g := f.Group("address", func(g *forms.Form) *forms.Form {
		g.Text("street").Label(i18n.Static("Street"))
		g.Check(func(context.Context) error { return errAddress })
		return g
	})
	g.Label(i18n.Static("Address"))
	f.Bind(forms.FromValues(url.Values{"address.street": {"x"}}))
	f.Validate(context.Background())
	c := firstControl(f)
	html := render(t, Field(c))
	if !strings.Contains(html, `aria-describedby="field--t-1-error"`) || !strings.Contains(html, `id="field--t-1-error"`) {
		t.Errorf("group error not described: %s", html)
	}
}

func TestRemoteLabelOfFoundMissErrorAndUnconverted(t *testing.T) {
	lbl := &countLabeler{names: map[string]string{"7": "Ann Lee"}}
	search := ui.URL("/customers/search")

	// Found.
	f := forms.New()
	fld := f.Number[int64]("customer_id").Label(i18n.Static("Customer")).Remote(search, lbl)
	fld.Set(7)
	html := render(t, Field(fld.Control()))
	checkAll(t, html)
	if lbl.calls != 1 || !strings.Contains(html, `>Ann Lee</span>`) {
		t.Errorf("found: calls=%d html=%s", lbl.calls, html)
	}
	for _, attr := range []string{
		`data-ui-search="/customers/search"`,
		`hx-get="/customers/search"`,
		`hx-trigger="ui-search"`,
		`hx-params="none"`,
		`hx-target="find .ui-combobox-options"`,
		`hx-select="[data-ui-suggestions] &gt; li"`,
		`hx-swap="innerHTML"`,
		`hx-sync="this:replace"`,
		`inputmode="numeric"`,
	} {
		if !strings.Contains(html, attr) {
			t.Errorf("missing %s in:\n%s", attr, html)
		}
	}

	// Miss -> renders the id.
	f = forms.New()
	fld = f.Number[int64]("customer_id").Label(i18n.Static("Customer")).Remote(search, lbl)
	fld.Set(99)
	html = render(t, Field(fld.Control()))
	checkAll(t, html)
	if !strings.Contains(html, `>99</span>`) {
		t.Errorf("miss should show id: %s", html)
	}

	// Unconverted text -> LabelOf not called, input holds text, span is empty.
	callsBefore := lbl.calls
	f = forms.New()
	fld = f.Number[int64]("customer_id").Label(i18n.Static("Customer")).Remote(search, lbl)
	f.Bind(forms.FromValues(url.Values{"customer_id": {"abc"}}))
	f.Validate(context.Background())
	html = render(t, Field(fld.Control()))
	checkAll(t, html)
	if lbl.calls != callsBefore {
		t.Errorf("unconverted text called LabelOf: %d -> %d", callsBefore, lbl.calls)
	}
	if !strings.Contains(html, `value="abc"`) || !strings.Contains(html, `id="field--t-1-chosen"></span>`) {
		t.Errorf("unconverted text markup: %s", html)
	}

	// Error -> render returns the error.
	wantErr := stderrors.New("db down")
	lbl.err = wantErr
	f = forms.New()
	fld = f.Number[int64]("customer_id").Label(i18n.Static("Customer")).Remote(search, lbl)
	fld.Set(7)
	_, err := uitest.RenderErr(Field(fld.Control()), uitest.Context(func(ctx context.Context) context.Context {
		return actions.Gate.With(ctx, actions.AllowAll)
	}))
	if !stderrors.Is(err, wantErr) {
		t.Errorf("got err %v, want %v", err, wantErr)
	}
}

func TestWithLabelCache(t *testing.T) {
	lbl := &countLabeler{names: map[string]string{"103": "Brick 2x2 yellow"}}
	search := ui.URL("/products/search")

	f := forms.New()
	lines := f.List("lines", func(g *forms.Form) *forms.Field[int64] {
		return g.Number[int64]("product_id").Label(i18n.Static("Product")).Remote(search, lbl)
	}).Label(i18n.Static("Lines"))
	for range 3 {
		lines.Add().Set(103)
	}

	// Without cache: 3 calls.
	render(t, Field(lines.Control()))
	if lbl.calls != 3 {
		t.Errorf("without cache: got %d calls, want 3", lbl.calls)
	}

	// With cache: 1 call for repeated id.
	lbl.calls = 0
	ctx := WithLabelCache(actions.Gate.With(context.Background(), actions.AllowAll))
	ctx = ui.WithIDs(ctx, ui.Sequence("t"))
	if err := Field(lines.Control()).Render(ctx, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if lbl.calls != 1 {
		t.Errorf("with cache: got %d calls, want 1", lbl.calls)
	}

	// Non-comparable Labeler is never cached, and doesn't panic.
	fnCalls := 0
	var nonComp funcLabeler = func(_ context.Context, id string) (string, bool, error) {
		fnCalls++
		return "Name " + id, true, nil
	}
	f2 := forms.New()
	l2 := f2.List("lines", func(g *forms.Form) *forms.Field[int64] {
		return g.Number[int64]("product_id").Label(i18n.Static("Product")).Remote(search, nonComp)
	}).Label(i18n.Static("Lines"))
	l2.Add().Set(1)
	l2.Add().Set(1)
	if err := Field(l2.Control()).Render(ctx, &strings.Builder{}); err != nil {
		t.Fatal(err)
	}
	if fnCalls != 2 {
		t.Errorf("non-comparable labeler: got %d calls, want 2", fnCalls)
	}
}

func TestRemoteGateRefusedAndMissing(t *testing.T) {
	lbl := &countLabeler{names: map[string]string{"7": "Ann Lee"}}
	search := ui.URL("/customers/search")
	f := forms.New()
	fld := f.Text("customer_id").Label(i18n.Static("Customer")).Remote(search, lbl)
	fld.Set("7")

	// Refused gate -> no data-ui-search and no hx-*, chosen name still rendered.
	html := uitest.Render(t, Field(fld.Control()), uitest.Context(func(ctx context.Context) context.Context {
		return actions.Gate.With(ctx, func(ui.Link) bool { return false })
	}))
	checkAll(t, html)
	if strings.Contains(html, "data-ui-search") || strings.Contains(html, "hx-") {
		t.Errorf("refused gate should omit search and hx-*: %s", html)
	}
	if !strings.Contains(html, `>Ann Lee</span>`) {
		t.Errorf("refused gate should still show chosen name: %s", html)
	}

	// No gate in context -> panics.
	defer func() {
		if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "no actions.Gate") {
			t.Errorf("expected panic for missing gate, got %v", r)
		}
	}()
	Field(fld.Control()).Render(ui.WithIDs(context.Background(), ui.Sequence("t")), &strings.Builder{})
}

func TestUploadLostPreviewsAndGate(t *testing.T) {
	stored := []files.Stored{
		{Key: "k-1", Name: "receipt.pdf", Href: ui.URL("/files/k-1")},
		{Key: "k-2", Name: "photo.png", Href: ui.URL("/files/k-2"), Image: true},
	}
	f := forms.New()
	req := f.Text("title").Label(i18n.Static("Title")).Required()
	up := f.Upload("doc", stored[:1]).Label(i18n.Static("Document")).
		Is(rule.Files(rule.MaxSize(2 << 20)))
	ups := f.Uploads("gallery", stored).Label(i18n.Static("Gallery"))
	fl := f.File("avatar").Label(i18n.Static("Avatar"))
	_ = req

	// Build a multipart request posting a file to doc and avatar, while title is missing (form fails -> Lost() is true).
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("doc", "new.pdf")
	fw.Write([]byte("%PDF-1.4"))
	fw2, _ := mw.CreateFormFile("avatar", "pic.png")
	fw2.Write([]byte("png"))
	mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/upload", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if f.Submit(context.Background(), r) {
		t.Fatal("expected form to fail on missing title")
	}

	htmlDoc := render(t, Field(up.Control()).Previews())
	checkAll(t, htmlDoc)
	if !strings.Contains(htmlDoc, `data-ui-previews`) || !strings.Contains(htmlDoc, `ui-upload-lost`) || !strings.Contains(htmlDoc, `aria-describedby="field--t-1-lost"`) {
		t.Errorf("upload lost/previews: %s", htmlDoc)
	}
	if !strings.Contains(htmlDoc, `%s is larger than 2 MB`) {
		t.Errorf("tooLarge size not filled: %s", htmlDoc)
	}

	htmlFile := render(t, Field(fl.Control()).Previews())
	checkAll(t, htmlFile)
	if !strings.Contains(htmlFile, `ui-upload-lost`) || strings.Contains(htmlFile, `ui-upload-files`) {
		t.Errorf("file lost note or unexpected stored list: %s", htmlFile)
	}

	// Refused gate on stored files -> no thumbnail and no link, keep checkbox and label remain.
	htmlRefused := uitest.Render(t, Field(ups.Control()), uitest.Context(func(ctx context.Context) context.Context {
		return actions.Gate.With(ctx, func(ui.Link) bool { return false })
	}))
	checkAll(t, htmlRefused)
	if strings.Contains(htmlRefused, "<img") || strings.Contains(htmlRefused, "<a ") {
		t.Errorf("refused gate rendered thumbnail or link: %s", htmlRefused)
	}
	if !strings.Contains(htmlRefused, ">Keep receipt.pdf</label>") || !strings.Contains(htmlRefused, ">Keep photo.png</label>") {
		t.Errorf("refused gate missing keep labels: %s", htmlRefused)
	}
}

func TestPreviewsPanicsOnNonUpload(t *testing.T) {
	f := forms.New()
	c := f.Text("name").Control()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Previews on KindText did not panic")
		}
	}()
	Field(c).Previews()
}
