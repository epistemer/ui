package tableui

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/components"
	"github.com/epistemer/ui/internal/links"
)

// ReplyTo is an action's answer being built (§5.6): one change, an
// optional toast, and an ending.
type ReplyTo struct {
	w       http.ResponseWriter
	r       *http.Request
	toast   i18n.Ter
	change  string // "replace", "remove", "refresh", ""
	id      string
	content templ.Component
}

// Reply starts an action's answer.
func Reply(w http.ResponseWriter, r *http.Request) *ReplyTo { return &ReplyTo{w: w, r: r} }

func (a *ReplyTo) Toast(t i18n.Ter) *ReplyTo { a.toast = t; return a }

// Replace swaps c for the element with the keyed id.
func (a *ReplyTo) Replace(id string, c templ.Component) *ReplyTo {
	a.change, a.id, a.content = "replace", id, c
	return a
}

// Remove removes the element with the id.
func (a *ReplyTo) Remove(id string) *ReplyTo { a.change, a.id = "remove", id; return a }

// Refresh reloads the table's results at its current URL.
func (a *ReplyTo) Refresh(tableID string) *ReplyTo { a.change, a.id = "refresh", tableID; return a }

// ErrNotLocal is an Else or Go to an address that isn't this site's.
var ErrNotLocal = errors.New("tableui: Reply: the address isn't local")

// local reports whether l is an address on this site: a path from the
// root, with no scheme or host, no backslash, and no control character
// or white space a browser would strip into "//" (review 0014).
func local(l ui.Link) (string, bool) {
	u := string(links.Href(l))
	if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") || strings.ContainsAny(u, "\\") {
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

// Else sends the answer; a request without htmx goes to l with 303.
func (a *ReplyTo) Else(l ui.Link) error {
	u, ok := local(l)
	if !ok {
		return ErrNotLocal
	}
	if !htmx(a.r) {
		http.Redirect(a.w, a.r, u, http.StatusSeeOther)
		return nil
	}
	h := a.w.Header()
	trigger := map[string]any{"ui-action-done": true}
	var body bytes.Buffer
	switch a.change {
	case "replace":
		h.Set("HX-Retarget", "#"+a.id)
		h.Set("HX-Reswap", "outerHTML")
		if err := a.content.Render(a.r.Context(), &body); err != nil {
			return err
		}
	case "remove":
		h.Set("HX-Retarget", "#"+a.id)
		h.Set("HX-Reswap", "delete")
	case "refresh":
		h.Set("HX-Reswap", "none")
		trigger["ui-table-refresh"] = map[string]string{"id": a.id}
	default:
		h.Set("HX-Reswap", "none")
	}
	if a.toast != nil {
		if err := components.ToastSwap(components.Toast(a.toast)).Render(a.r.Context(), &body); err != nil {
			return err
		}
	}
	t, _ := json.Marshal(trigger)
	h.Set("HX-Trigger-After-Settle", string(t)) // after the swap, which would reopen the dialog
	h.Set("Content-Type", "text/html; charset=utf-8")
	_, err := a.w.Write(body.Bytes())
	return err
}

// Go sends the browser to l either way, instead of a change.
func (a *ReplyTo) Go(l ui.Link) error {
	u, ok := local(l)
	if !ok {
		return ErrNotLocal
	}
	if !htmx(a.r) {
		http.Redirect(a.w, a.r, u, http.StatusSeeOther)
		return nil
	}
	a.w.Header().Set("HX-Redirect", u)
	a.w.WriteHeader(http.StatusNoContent)
	return nil
}
