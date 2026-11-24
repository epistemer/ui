package components

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/actions"
	"github.com/epistemer/ui/actions/actionstest"
)

type link struct{ url, method string }

func (l link) URL() string    { return l.url }
func (l link) Method() string { return l.method }

var errRefused = errors.Key("errors.test.refused", "not now", errors.Conflict)

var (
	aView   = actions.New(i18n.Static("View"))
	aEdit   = actions.New(i18n.Static("Edit")).Primary()
	aOpen   = actions.New(i18n.Static("Reassign")).Dialog()
	aRefund = actions.New(i18n.Static("Refund")).Danger().Confirm(i18n.Static("Sure?"))
	aPut    = actions.New(i18n.Static("Archive"))
)

func items(t *testing.T, refundable bool) []actions.Item {
	var reason error
	if !refundable {
		reason = errRefused
	}
	return actions.Resolve(actionstest.Context(t),
		aView.To(ui.URL("/o/1")), aEdit.To(ui.URL("/o/1/edit")), aOpen.To(link{"/o/1/reassign", "GET"}),
		aPut.To(link{"/o/1/archive", "PUT"}), aRefund.To(link{"/o/1/refund", "POST"}).Unless(reason))
}

func TestActionButtons(t *testing.T) {
	html := render(t, ActionButtons(items(t, true)))
	for _, want := range []string{
		`<a class="ui-button" data-tone="neutral" data-variant="outline" href="/o/1">`,
		`class="ui-button" href="/o/1/edit"`,
		`hx-get="/o/1/reassign" hx-push-url="false" hx-swap="innerHTML" hx-target="#ui-action-dialog-body"`,
		`form="ui-action-form" formaction="/o/1/archive" formmethod="post" hx-put="/o/1/archive" hx-swap="none"`,
		`data-tone="danger"`, `data-ui-confirm=`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in\n%s", want, html)
		}
	}
	html = render(t, ActionButtons(items(t, false)))
	refund := html[strings.Index(html, "Refund")-700:]
	if !strings.Contains(html, `aria-disabled="true"`) || strings.Contains(html, "/o/1/refund") || strings.Contains(refund, "data-ui-confirm") {
		t.Errorf("an unavailable item can still act:\n%s", html)
	}
	if !strings.Contains(html, "not now") {
		t.Error("the reason isn't shown")
	}
	if got := render(t, ActionButtons(nil)); strings.TrimSpace(got) != "" {
		t.Errorf("no items: %q", got)
	}
}

func TestActionMenu(t *testing.T) {
	html := render(t, ActionMenu(i18n.Static("Actions"), items(t, false)).ID("row-1-actions").LabelledBy("row-1-label"))
	for _, want := range []string{
		`aria-labelledby="row-1-actions row-1-label"`, `id="row-1-actions"`, `id="row-1-actions-menu"`,
		`data-ui-unavailable`, `class="ui-separator"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s", want)
		}
	}
	// Danger items last, after the separator.
	if strings.Index(html, "Refund") < strings.Index(html, "ui-separator") {
		t.Error("the danger item isn't last")
	}
	if got := render(t, ActionMenu(i18n.Static("Actions"), nil)); strings.TrimSpace(got) != "" {
		t.Errorf("no items: %q", got)
	}
	panicsWith(t, "a menu with a spaced id", "keyed id", Menu(i18n.Static("x")).ID("a b"))
	panicsWith(t, "an unavailable item with no reason", "blank Unavailable", MenuItem().Unavailable(i18n.Static(" ")))
}

func req(hx bool, target string) (*httptest.ResponseRecorder, *http.Request) {
	r := httptest.NewRequest("POST", "/o/1/refund", nil)
	if hx {
		r.Header.Set("HX-Request", "true")
	}
	if target != "" {
		r.Header.Set("HX-Target", target)
	}
	return httptest.NewRecorder(), r
}

func TestReply(t *testing.T) {
	back := ui.URL("/orders")
	w, r := req(false, "")
	if Reply(w, r).Toast(i18n.Static("Done")).Remove("row-1").Else(back); w.Code != 303 || w.Header().Get("Location") != "/orders" {
		t.Errorf("without htmx: %d %v", w.Code, w.Header())
	}
	for _, c := range []struct {
		name   string
		reply  func(*ReplySpec) *ReplySpec
		status int
		hdr    map[string]string
		body   []string
	}{
		{"replace", func(a *ReplySpec) *ReplySpec {
			return a.Replace("row-1", templ.Raw(`<tr id="row-1"><td>x</td></tr>`)).Toast(i18n.Static("Saved"))
		}, 200, map[string]string{"HX-Retarget": "#row-1", "HX-Reswap": "outerHTML", "HX-Trigger-After-Settle": `{"ui-action-done":true}`}, []string{`<tr id="row-1">`, `hx-swap-oob`, "Saved"}},
		{"remove", func(a *ReplySpec) *ReplySpec { return a.Remove("row-1") }, 200, map[string]string{"HX-Retarget": "#row-1", "HX-Reswap": "delete"}, nil},
		{"refresh", func(a *ReplySpec) *ReplySpec { return a.Refresh("orders") }, 200, map[string]string{"HX-Reswap": "none", "HX-Trigger-After-Settle": `{"ui-action-done":true,"ui-table-refresh":{"id":"orders"}}`}, nil},
		{"fail", func(a *ReplySpec) *ReplySpec { return a.Toast(i18n.Static("Saved")).Fail(errRefused) }, 422, map[string]string{"HX-Reswap": "none", "HX-Trigger-After-Settle": ""}, []string{`data-tone="danger"`, "not now"}},
		{"nothing", func(a *ReplySpec) *ReplySpec { return a }, 200, map[string]string{"HX-Reswap": "none"}, nil},
	} {
		w, r := req(true, "")
		if err := c.reply(Reply(w, r)).Else(back); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if w.Code != c.status {
			t.Errorf("%s: status %d", c.name, w.Code)
		}
		for k, v := range c.hdr {
			if got := w.Header().Get(k); got != v {
				t.Errorf("%s: %s = %q, want %q", c.name, k, got, v)
			}
		}
		for _, b := range c.body {
			if !strings.Contains(w.Body.String(), b) {
				t.Errorf("%s: body lacks %s", c.name, b)
			}
		}
		if c.name == "fail" && strings.Contains(w.Body.String(), "Saved") {
			t.Error("fail kept the success toast")
		}
	}
	w, r = req(true, "")
	if Reply(w, r).Go(ui.URL("/orders/2")); w.Header().Get("HX-Redirect") != "/orders/2" {
		t.Errorf("go: %v", w.Header())
	}
	for _, u := range []string{"https://evil.example/", "//evil.example", "/\\evil.example", "/\t/evil.example", "/ /x"} {
		w, r := req(true, "")
		if err := Reply(w, r).Else(ui.URL(u)); err != ErrNotLocal || w.Body.Len() != 0 {
			t.Errorf("Else(%q): %v", u, err)
		}
	}
	for name, f := range map[string]func(*ReplySpec){
		"two changes":    func(a *ReplySpec) { a.Remove("a").Refresh("t") },
		"Fail(nil)":      func(a *ReplySpec) { a.Fail(nil) },
		"a blank id":     func(a *ReplySpec) { a.Remove(" ") },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s didn't panic", name)
				}
			}()
			w, r := req(true, "")
			f(Reply(w, r))
		}()
	}
}

func TestInActionDialog(t *testing.T) {
	_, r := req(true, "ui-action-dialog-body")
	_, plain := req(false, "ui-action-dialog-body")
	_, other := req(true, "orders")
	if !InActionDialog(r) || InActionDialog(plain) || InActionDialog(other) {
		t.Error("InActionDialog")
	}
	_ = context.Background
}

func TestUnavailableItemInert(t *testing.T) {
	html := render(t, MenuItem().Href(ui.URL("/x")).Submit("f").
		Attrs(templ.Attributes{"data-hx-post": "/x", "HX-Get": "/y", "commandfor": "d", "command": "show-modal", "popovertarget": "p", "data-ui-confirm": "q", "data-keep": "1"}).
		Unavailable(i18n.Static("not now")))
	for _, gone := range []string{"href", "data-hx-post", "hx-get", "HX-Get", "commandfor", "show-modal", "popovertarget", "data-ui-confirm", `type="submit"`, `form="f"`} {
		if strings.Contains(html, gone) {
			t.Errorf("an unavailable item keeps %s:\n%s", gone, html)
		}
	}
	if !strings.Contains(html, `data-keep="1"`) {
		t.Error("an unavailable item lost an attribute it may keep")
	}
}
