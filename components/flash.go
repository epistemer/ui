package components

import (
	"context"
	"io"
	"iter"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui"
	"github.com/epistemer/ui/elements"
)

// Flasher carries messages to the next page across a redirect, and gives
// the ones this request delivers (amendment A9, confab-1-interface.md §6).
// handlers/middleware's Flash is one; ui imports no handlers. Levels are
// "info", "success", "warning" and "error".
type Flasher interface {
	Add(ctx context.Context, level string, msg i18n.Ter, args ...any)
	Messages(ctx context.Context) iter.Seq2[string, i18n.Ter]
}

// Flash is the request's Flasher, set once per request as actions.Gate is,
// inside handlers' Flash.Middleware: ctx = components.Flash.With(ctx, fl).
// Without one, Reply drops a toast it would carry across a redirect, and
// Toasts shows no flashed messages.
var Flash = ui.NewProvider[Flasher](nil)

// The flash's levels. Reply's toast has no tone, so it is info; Fail's
// message is an error.
const (
	levelInfo    = "info"
	levelSuccess = "success"
	levelWarning = "warning"
	levelError   = "error"
)

// toneOf is a level's toast tone; an unknown level reads as info, which
// has no tone.
func toneOf(level string) elements.Tone {
	switch level {
	case levelSuccess:
		return elements.Success
	case levelWarning:
		return elements.Warning
	case levelError:
		return elements.Danger
	}
	return ""
}

// flashed renders the messages the request's flash delivers, as toasts.
// They render after the region, so each moves itself in when it connects
// and is announced as one that arrives by htmx.
func flashed() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		f := Flash.Value(ctx)
		if f == nil {
			return nil
		}
		for level, msg := range f.Messages(ctx) {
			if msg == nil || strings.TrimSpace(msg.T(ctx)) == "" {
				continue
			}
			if err := Toast(msg).Tone(toneOf(level)).Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}
