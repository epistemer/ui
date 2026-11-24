package components

import (
	"context"
	"iter"
	"net/http"
	"strings"
	"testing"

	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/uitest"
)

// fakeFlash records what Reply adds, as handlers' Flash would before the
// redirect's header goes out, and delivers what it was given.
type fakeFlash struct {
	added []flashMsg
	give  []flashMsg
}

type flashMsg struct {
	level string
	text  string
}

func (f *fakeFlash) Add(ctx context.Context, level string, msg i18n.Ter, args ...any) {
	f.added = append(f.added, flashMsg{level, msg.T(ctx, args...)})
}

func (f *fakeFlash) Messages(ctx context.Context) iter.Seq2[string, i18n.Ter] {
	return func(yield func(string, i18n.Ter) bool) {
		for _, m := range f.give {
			if !yield(m.level, i18n.Static(m.text)) {
				return
			}
		}
	}
}

var errRefund = errors.Key("errors.shop.refund_closed", "the refund window has closed", errors.Invalid)

func withFlash(r *http.Request, f Flasher) *http.Request {
	return r.WithContext(Flash.With(r.Context(), f))
}

func TestReplyFlash(t *testing.T) {
	saved := i18n.Static("Saved")
	cases := []struct {
		name   string
		hx     bool
		answer func(*ReplySpec) error
		want   []flashMsg
		header string // Location or HX-Redirect
	}{
		{"go after a toast", true, func(a *ReplySpec) error { return a.Toast(saved).Go(ui.URL("/orders")) },
			[]flashMsg{{"info", "Saved"}}, "/orders"},
		{"go after a toast, no script", false, func(a *ReplySpec) error { return a.Toast(saved).Go(ui.URL("/orders")) },
			[]flashMsg{{"info", "Saved"}}, "/orders"},
		{"go after a change drops it", true, func(a *ReplySpec) error { return a.Toast(saved).Remove("order-1").Go(ui.URL("/orders")) },
			[]flashMsg{{"info", "Saved"}}, "/orders"},
		{"go alone", true, func(a *ReplySpec) error { return a.Go(ui.URL("/orders")) }, nil, "/orders"},
		{"fail then go", false, func(a *ReplySpec) error { return a.Toast(saved).Fail(errRefund).Go(ui.URL("/orders/1")) },
			[]flashMsg{{"error", "the refund window has closed"}}, "/orders/1"},
		{"no script: the toast crosses the 303", false, func(a *ReplySpec) error { return a.Toast(saved).Remove("order-1").Else(ui.URL("/orders")) },
			[]flashMsg{{"info", "Saved"}}, "/orders"},
		{"no script: fail's message crosses the 303", false, func(a *ReplySpec) error { return a.Fail(errRefund).Else(ui.URL("/orders/1")) },
			[]flashMsg{{"error", "the refund window has closed"}}, "/orders/1"},
		{"private errors read generic", false, func(a *ReplySpec) error { return a.Fail(errors.New("db: deadlock")).Else(ui.URL("/orders/1")) },
			nil, "/orders/1"},
		{"htmx: in place, nothing flashed", true, func(a *ReplySpec) error { return a.Toast(saved).Remove("order-1").Else(ui.URL("/orders")) },
			nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeFlash{}
			w, r := req(c.hx, "")
			r = withFlash(r, f)
			if err := c.answer(Reply(w, r)); err != nil {
				t.Fatal(err)
			}
			got := f.added
			if c.name == "private errors read generic" {
				if len(got) != 1 || got[0].level != "error" || strings.Contains(got[0].text, "deadlock") {
					t.Errorf("flashed %v", got)
				}
			} else if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) {
				t.Errorf("flashed %v, want %v", got, c.want)
			}
			loc := w.Header().Get("Location")
			if c.hx {
				loc = w.Header().Get("HX-Redirect")
			}
			if loc != c.header {
				t.Errorf("redirect %q, want %q (%v)", loc, c.header, w.Header())
			}
		})
	}
}

func TestReplyWithoutFlash(t *testing.T) {
	w, r := req(false, "")
	if err := Reply(w, r).Toast(i18n.Static("Saved")).Go(ui.URL("/orders")); err != nil || w.Code != http.StatusSeeOther {
		t.Fatalf("Go without a flash: %v, %d", err, w.Code)
	}
	// A nil Flasher is no flash.
	w, r = req(false, "")
	r = withFlash(r, nil)
	if err := Reply(w, r).Fail(errRefund).Else(ui.URL("/orders")); err != nil || w.Code != http.StatusSeeOther {
		t.Fatalf("Else with a nil flash: %v, %d", err, w.Code)
	}
}

func TestReplyNotLocalFlashesNothing(t *testing.T) {
	f := &fakeFlash{}
	w, r := req(false, "")
	r = withFlash(r, f)
	if err := Reply(w, r).Toast(i18n.Static("Saved")).Go(ui.URL("https://evil.example/")); err != ErrNotLocal || len(f.added) != 0 {
		t.Fatalf("%v, %v", err, f.added)
	}
}

func TestToastsShowFlashes(t *testing.T) {
	f := &fakeFlash{give: []flashMsg{{"success", "Saved"}, {"error", "Refused"}, {"weird", "Odd"}, {"info", " "}}}
	html := uitest.Render(t, Toasts(), uitest.Context(func(ctx context.Context) context.Context { return Flash.With(ctx, f) }))
	region := strings.Index(html, "</ui-toasts>")
	if region < 0 {
		t.Fatal(html)
	}
	after := html[region:]
	for _, want := range []string{`data-tone="success"`, "Saved", `data-tone="danger"`, "Refused", "Odd"} {
		if !strings.Contains(after, want) {
			t.Errorf("no %q after the region:\n%s", want, html)
		}
	}
	if strings.Count(after, "<ui-toast ") != 3 {
		t.Errorf("want 3 toasts (a blank one skipped):\n%s", html)
	}
	if strings.Contains(html[:region], "<ui-toast ") {
		t.Error("a flashed toast rendered inside the region: it wouldn't be announced")
	}
	// Without a flash, the region alone.
	if got := uitest.Render(t, Toasts()); strings.Contains(got, "<ui-toast ") {
		t.Error(got)
	}
}
