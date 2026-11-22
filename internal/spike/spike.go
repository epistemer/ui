// Package spike is D8.2's slice spike (rich-inputs-design.md §8): the toy
// shop's order form with a remote select, an upload with a stored file and
// a list of lines, and the pairings the parts meet in. The shapes the
// design gives forms (Remote, Upload) are declared here, locally, keyed by
// path; nothing in forms or inputs changes before the lock.
package spike

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/listing"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/forms"
	"github.com/epistemer/ui/theme"
)

// Prefix is where the spike is mounted.
const Prefix = "/spike"

//go:embed assets
var assets embed.FS

// link stands in for handlers' generated link.
type link string

func (l link) URL() string    { return string(l) }
func (l link) Method() string { return "GET" }

// A record people pick by name.
type record struct {
	ID   int64
	Name string
}

var customers = func() []record {
	names := []string{"Ann Lee", "Bassem Aziz", "Carla Ruiz", "Dina Farouk", "Elias Berg", "Fatma Nour", "Gus Holm", "Hana Sato", "Ilse Vos", "Jamal Odeh", "Kira Novak", "Laila Haddad", "Mona Zaki", "Nils Lund", "Omar Saleh", "Petra Kos", "Quinn Ash", "Rana Adel", "Sami Halabi", "Tara Quist", "Uma Rao", "Victor Hale", "Wafa Samir", "Xenia Pop", "Yara Fawzi", "Zeid Amin"}
	out := make([]record, len(names))
	for i, n := range names {
		out[i] = record{int64(i + 1), n}
	}
	return out
}()

var products = func() []record {
	names := []string{"Brick 2x4 red", "Brick 2x4 blue", "Brick 2x2 yellow", "Plate 1x6 green", "Wheel small", "Wheel large", "Window frame", "Door", "Roof tile", "Minifigure", "Baseplate grey", "Baseplate green", "Hinge", "Antenna", "Propeller", "Gear 8t", "Gear 24t", "Axle 4", "Axle 8", "Tyre", "Steering wheel", "Ladder", "Fence", "Flower", "Tree", "Lamp post", "Sign", "Barrel", "Crate", "Chest"}
	out := make([]record, len(names))
	for i, n := range names {
		out[i] = record{int64(101 + i), n}
	}
	return out
}()

// table is a Labeler over records, counting its calls (R10).
type table struct {
	rows  []record
	mu    sync.Mutex
	calls int
	fail  bool
}

func (t *table) LabelOf(_ context.Context, id string) (string, bool, error) {
	t.mu.Lock()
	t.calls++
	fail := t.fail
	t.mu.Unlock()
	if fail {
		return "", false, fmt.Errorf("spike: the database is down")
	}
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", false, nil // an id that can't name a row is a miss (§3.2)
	}
	for _, r := range t.rows {
		if r.ID == n {
			return r.Name, true, nil
		}
	}
	return "", false, nil
}

func (t *table) search(q string, limit int) []record {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []record
	for _, r := range t.rows {
		if q == "" || strings.Contains(strings.ToLower(r.Name), q) || strconv.FormatInt(r.ID, 10) == q {
			out = append(out, r)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}

var (
	customerTable = &table{rows: customers}
	productTable  = &table{rows: products}
)

// remote is what Field[T].Remote would hold (R1).
type remote struct {
	search ui.Link
	l      listing.Labeler
}

// remotes are the spike's remote fields, by path with indices as __i__.
var remotes = map[string]remote{
	"customer_id":                           {link(Prefix + "/search/customers"), customerTable},
	"lines[__i__].product_id":               {link(Prefix + "/search/products"), productTable},
	"lines[__i__].extras[__i__].product_id": {link(Prefix + "/search/products"), productTable},
	"assignee_id":                           {link(Prefix + "/search/customers"), customerTable},
	"filter.customer_id":                    {link(Prefix + "/search/customers"), customerTable},
	"refused_id":                            {link(Prefix + "/search/refused"), customerTable},
}

var index = regexp.MustCompile(`\[\d+\]`)

func remoteOf(path string) (remote, bool) {
	r, ok := remotes[index.ReplaceAllString(path, "[__i__]")]
	return r, ok
}

// lists are each list's bounds, which forms' Control doesn't expose
// (finding: the contract needs them).
var lists = map[string][2]int{
	"lines":               {1, 5},
	"lines[__i__].extras": {0, 3},
}

func boundsOf(path string) [2]int {
	return lists[index.ReplaceAllString(path, "[__i__]")]
}

// storedFile is forms.StoredFile (R16).
type storedFile struct {
	Key, Name string
	Href      ui.Link
	Image     bool
}

// upload is what Form.Upload would hold, bound by hand from the request.
type upload struct {
	path   string
	stored []storedFile
	keep   map[string]bool // posted keep state, by key
	added  int             // files posted
	lost   bool            // posted files in a form that failed
}

func (u *upload) bind(r *http.Request) {
	u.keep = map[string]bool{}
	var keys []string
	if r.MultipartForm != nil {
		keys = r.MultipartForm.Value[u.path+".keep"]
		u.added = len(r.MultipartForm.File[u.path])
	} else {
		keys = r.PostForm[u.path+".keep"]
	}
	for _, k := range keys {
		if slices.ContainsFunc(u.stored, func(s storedFile) bool { return s.Key == k }) {
			u.keep[k] = true // a key not given is dropped; a repeat counts once
		}
	}
}

func (u *upload) kept(key string) bool { return u.keep == nil || u.keep[key] }

type order struct {
	f        *forms.Form
	customer *forms.Field[int64]
	lines    *forms.List[line]
	receipt  *upload
}

type line struct {
	product *forms.Field[int64]
	qty     *forms.Field[int]
	photo   *forms.Field[*multipart.FileHeader]
}

func orderForm() *order {
	f := forms.New()
	o := &order{f: f}
	o.customer = f.Number[int64]("customer_id").Label(i18n.Static("Customer")).Required()
	o.lines = f.List("lines", func(g *forms.Form) line {
		l := line{
			product: g.Number[int64]("product_id").Label(i18n.Static("Product")).Required(),
			qty:     g.Int("qty").Label(i18n.Static("Quantity")).Required(),
		}
		l.photo = g.File("photo").Label(i18n.Static("Photo")).When(l.qty.Filled())
		g.List("extras", func(e *forms.Form) int {
			e.Number[int64]("product_id").Label(i18n.Static("Extra product")).Required()
			return 0
		}).Label(i18n.Static("Extras")).MaxItems(3)
		return l
	})
	o.lines.Label(i18n.Static("Lines")).MinItems(1).MaxItems(5)
	o.receipt = &upload{path: "receipt", stored: []storedFile{{Key: "k-81", Name: "receipt.pdf", Href: link(Prefix + "/files/k-81")}, {Key: "k-82", Name: "box.png", Href: link(Prefix + "/files/k-82"), Image: true}}}
	return o
}

// App serves the spike.
type App struct {
	Theme   *theme.Theme
	Scripts templ.Component
}

func gate(l ui.Link) bool { return !strings.HasSuffix(l.URL(), "/refused") }

// Mount registers the spike's routes.
func (a *App) Mount(mux *http.ServeMux) {
	h := func(f func(http.ResponseWriter, *http.Request) error) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(actions.Gate.With(r.Context(), gate))
			if err := f(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}
	mux.Handle("GET "+Prefix+"/assets/", http.StripPrefix(Prefix, http.FileServerFS(assets)))
	mux.Handle("GET "+Prefix+"/order", h(a.order))
	mux.Handle("POST "+Prefix+"/order", h(a.order))
	mux.Handle("GET "+Prefix+"/assign", h(a.assign))
	mux.Handle("POST "+Prefix+"/assign", h(a.assign))
	mux.Handle("GET "+Prefix+"/table", h(a.table))
	mux.Handle("GET "+Prefix+"/search/customers", h(a.search(customerTable)))
	mux.Handle("GET "+Prefix+"/search/products", h(a.search(productTable)))
	mux.Handle("GET "+Prefix+"/search/expired", h(func(w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, Prefix+"/signin", http.StatusSeeOther)
		return nil
	}))
	mux.Handle("GET "+Prefix+"/signin", h(func(w http.ResponseWriter, r *http.Request) error {
		return a.page(w, r, http.StatusOK, i18n.Static("Sign in"), signinPage())
	}))
	mux.Handle("GET "+Prefix+"/files/{key}", h(func(w http.ResponseWriter, r *http.Request) error {
		if r.PathValue("key") == "k-82" {
			w.Header().Set("Content-Type", "image/svg+xml")
			_, err := w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><rect width="64" height="64" fill="#c33"/></svg>`))
			return err
		}
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte("a stored file"))
		return err
	}))
}

func (a *App) page(w http.ResponseWriter, r *http.Request, status int, title i18n.Ter, body templ.Component) error {
	return write(w, r, status, pageTmpl(a.Theme, a.Scripts, title, body))
}

func write(w http.ResponseWriter, r *http.Request, status int, c templ.Component) error {
	var b bytes.Buffer
	ctx := withMemo(r.Context())
	if err := c.Render(ctx, &b); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(b.Bytes())
	return err
}

// SuggestionLimit is how many suggestions an answer shows (R6).
const SuggestionLimit = 20

func (a *App) search(t *table) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		rows := t.search(r.URL.Query().Get("q"), SuggestionLimit+1)
		opts := make([]suggestion, 0, len(rows))
		for _, c := range rows {
			opts = append(opts, suggestion{strconv.FormatInt(c.ID, 10), i18n.Static(c.Name)})
		}
		return write(w, r, http.StatusOK, suggestions(opts))
	}
}

// suggestion is a ui type, not models' option.SelectOption, whose label
// is an i18n.TKer: a row's name is data and has no key (finding).
type suggestion struct {
	Value string
	Label i18n.Ter
}

func (a *App) order(w http.ResponseWriter, r *http.Request) error {
	o := orderForm()
	status := http.StatusOK
	var saved string
	if r.Method == http.MethodPost {
		// Parsed first, so forms and the hand-bound upload read one body.
		if err := r.ParseMultipartForm(32 << 20); err != nil && err != http.ErrNotMultipart {
			return err
		}
		ok := o.f.Submit(r.Context(), r)
		o.receipt.bind(r)
		if ok {
			var kept []string
			for _, s := range o.receipt.stored {
				if o.receipt.kept(s.Key) {
					kept = append(kept, s.Key)
				}
			}
			saved = fmt.Sprintf("Saved: customer %d, %d lines, receipt kept %v, %d added.", o.customer.Value(), len(o.lines.Items()), kept, o.receipt.added)
		} else {
			status = http.StatusUnprocessableEntity
			o.receipt.lost = o.receipt.added > 0
		}
	} else {
		o.customer.Set(7)
		l := o.lines.Add()
		l.product.Set(103)
		l.qty.Set(2)
	}
	return a.page(w, r, status, i18n.Static("New order"), orderPage(o, saved))
}

var assignAction = actions.New(i18n.Static("Assign a customer")).Dialog()

func (a *App) assign(w http.ResponseWriter, r *http.Request) error {
	f := forms.New()
	who := f.Number[int64]("assignee_id").Label(i18n.Static("Customer")).Required()
	status := http.StatusOK
	if r.Method == http.MethodPost {
		if f.Submit(r.Context(), r) {
			return components.Reply(w, r).Toast(i18n.Static(fmt.Sprintf("Assigned to %d.", who.Value()))).Else(link(Prefix + "/order"))
		}
		status = http.StatusUnprocessableEntity
	}
	body := assignBody(f)
	if components.InActionDialog(r) {
		return write(w, r, status, body)
	}
	return a.page(w, r, status, i18n.Static("Assign"), body)
}

func (a *App) table(w http.ResponseWriter, r *http.Request) error {
	f := forms.New()
	g := f.Group("filter", func(g *forms.Form) *forms.Field[int64] {
		return g.Number[int64]("customer_id").Label(i18n.Static("Customer"))
	})
	refused := f.Number[int64]("refused_id").Label(i18n.Static("Customer (search refused)"))
	f.Bind(forms.FromValues(r.URL.Query()))
	f.Validate(r.Context())
	_ = g
	_ = refused
	items := actions.Resolve(r.Context(), assignAction.To(link(Prefix+"/assign")))
	return a.page(w, r, http.StatusOK, i18n.Static("Filters"), tablePage(f, items))
}

// memo is R10's per-request memo of LabelOf, by Labeler and id.
type memoKey struct {
	l  listing.Labeler
	id string
}

type memoVal struct {
	name string
	ok   bool
	err  error
}

type memo struct {
	mu sync.Mutex
	m  map[memoKey]memoVal
}

var memos = ui.NewProvider[*memo](nil)

func withMemo(ctx context.Context) context.Context {
	if memos.Value(ctx) != nil {
		return ctx
	}
	return memos.With(ctx, &memo{m: map[memoKey]memoVal{}})
}

func labelOf(ctx context.Context, l listing.Labeler, id string) (string, bool, error) {
	m := memos.Value(ctx)
	if m == nil {
		return l.LabelOf(ctx, id)
	}
	m.mu.Lock()
	v, hit := m.m[memoKey{l, id}]
	m.mu.Unlock()
	if hit {
		return v.name, v.ok, v.err
	}
	name, found, err := l.LabelOf(ctx, id)
	m.mu.Lock()
	m.m[memoKey{l, id}] = memoVal{name, found, err}
	m.mu.Unlock()
	return name, found, err
}
