package components

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/errors"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
	"github.com/epistemer/ui/internal/links"
)

// ErrNotLocal is Reply's Else or Go given an address that isn't this
// site's. A private error, for logs: nothing was written.
var ErrNotLocal = errors.New("components: Reply: the address isn't local")

type change int

const (
	noChange change = iota
	replaceChange
	removeChange
	refreshChange
	failChange
)

// ReplySpec is an action's answer being built (server-state-interface.md
// §4.4): one change, an optional toast, and an ending, Else or Go.
type ReplySpec struct {
	w       http.ResponseWriter
	r       *http.Request
	toast   i18n.Ter
	change  change
	id      string
	content templ.Component
	err     error
}

// Reply starts the answer to the request r on w.
func Reply(w http.ResponseWriter, r *http.Request) *ReplySpec { return &ReplySpec{w: w, r: r} }

// Toast shows text with the answer, through ToastSwap.
func (a *ReplySpec) Toast(t i18n.Ter) *ReplySpec { a.toast = t; return a }

func (a *ReplySpec) set(c change, id string) *ReplySpec {
	if a.change != noChange {
		panic("components: Reply with two changes: an answer makes one; refresh the table when several rows changed")
	}
	if c != failChange && strings.TrimSpace(id) == "" {
		panic("components: Reply with a blank id")
	}
	a.change, a.id = c, id
	return a
}

// Replace swaps c for the element with the keyed id; a table's row
// travels as the answer's own content.
func (a *ReplySpec) Replace(id string, c templ.Component) *ReplySpec {
	a.set(replaceChange, id)
	a.content = c
	return a
}

// Remove removes the element with the id.
func (a *ReplySpec) Remove(id string) *ReplySpec { return a.set(removeChange, id) }

// Refresh reloads the table's results at its current address.
func (a *ReplySpec) Refresh(tableID string) *ReplySpec { return a.set(refreshChange, tableID) }

// Fail answers a refusal: with htmx, 422 and a danger toast with err's
// text for people, and nothing changes; without, the redirect to Else,
// with that text in the request's Flash (A9). Fail(nil) panics.
func (a *ReplySpec) Fail(err error) *ReplySpec {
	if err == nil {
		panic("components: Reply.Fail(nil)")
	}
	a.set(failChange, "")
	a.err = err
	return a
}

// local is l's address if it is this site's: one leading slash, no
// backslash, white space or control character, no scheme or host.
func local(l ui.Link) (string, bool) {
	u := string(links.Href(l))
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") || strings.ContainsRune(u, '\\') {
		return u, false
	}
	for _, c := range u {
		if c <= ' ' || c == 0x7f {
			return u, false
		}
	}
	p, err := url.Parse(u)
	return u, err == nil && p.Scheme == "" && p.Host == "" && !strings.HasPrefix(p.Path, "//")
}

func htmx(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// carry hands what the answer would have shown to the request's Flash,
// before a redirect drops it: Fail's message as an error, else the toast.
// Without a Flash it is dropped (A9, Q5).
func (a *ReplySpec) carry() {
	ctx := a.r.Context()
	f := Flash.Value(ctx)
	if f == nil {
		return
	}
	switch {
	case a.change == failChange:
		f.Add(ctx, levelError, reasonTer{a.err})
	case a.toast != nil:
		f.Add(ctx, levelInfo, a.toast)
	}
}

// Else sends the answer. A request without htmx is redirected to l with
// 303, and the toast (or Fail's message) goes to the request's Flash. An
// address that isn't local returns ErrNotLocal, and nothing is written.
func (a *ReplySpec) Else(l ui.Link) error {
	u, ok := local(l)
	if !ok {
		return ErrNotLocal
	}
	if !htmx(a.r) {
		a.carry()
		http.Redirect(a.w, a.r, u, http.StatusSeeOther)
		return nil
	}
	ctx := a.r.Context()
	h := a.w.Header()
	events := map[string]any{}
	status := http.StatusOK
	var body bytes.Buffer
	switch a.change {
	case replaceChange:
		h.Set("HX-Retarget", "#"+a.id)
		h.Set("HX-Reswap", "outerHTML")
		if err := a.content.Render(ctx, &body); err != nil {
			return err
		}
		events["ui-action-done"] = true
	case removeChange:
		h.Set("HX-Retarget", "#"+a.id)
		h.Set("HX-Reswap", "delete")
		events["ui-action-done"] = true
	case refreshChange:
		h.Set("HX-Reswap", "none")
		events["ui-action-done"] = true
		events["ui-table-refresh"] = map[string]string{"id": a.id}
	case failChange:
		h.Set("HX-Reswap", "none")
		status = http.StatusUnprocessableEntity
		if err := ToastSwap(Toast(reasonTer{a.err}).Tone(elements.Danger)).Render(ctx, &body); err != nil {
			return err
		}
	default:
		h.Set("HX-Reswap", "none")
		events["ui-action-done"] = true
	}
	if a.toast != nil && a.change != failChange {
		if err := ToastSwap(Toast(a.toast)).Render(ctx, &body); err != nil {
			return err
		}
	}
	if len(events) > 0 {
		// After settling: before the swap, the dialog would close and the
		// swap step reopen it (the spike's finding 4).
		t, _ := json.Marshal(events)
		h.Set("HX-Trigger-After-Settle", string(t))
	}
	h.Set("Content-Type", "text/html; charset=utf-8")
	a.w.WriteHeader(status)
	_, err := a.w.Write(body.Bytes())
	return err
}

// Go sends the browser to l, with htmx or without. A change set before it
// is dropped (the page it goes to shows the result); the toast, or Fail's
// message, goes to the request's Flash (A9). An address that isn't local
// returns ErrNotLocal, and nothing is written or flashed.
func (a *ReplySpec) Go(l ui.Link) error {
	u, ok := local(l)
	if !ok {
		return ErrNotLocal
	}
	a.carry()
	if !htmx(a.r) {
		http.Redirect(a.w, a.r, u, http.StatusSeeOther)
		return nil
	}
	a.w.Header().Set("HX-Redirect", u)
	a.w.WriteHeader(http.StatusNoContent)
	return nil
}
