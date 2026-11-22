package elements

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
	"github.com/epistemer/models/option"
)

// The value renderers (§5.4) render a value by kind, the same way in a
// table cell (D7) and a detail row. Each renders Empty instead of a value
// that isn't there: a nil Ter (but Image's alt), one whose text is empty
// or white space, a nil option, the zero time, an empty href or src.

func blank(ctx context.Context, t i18n.Ter) bool {
	return t == nil || strings.TrimSpace(t.T(ctx)) == ""
}

// Text renders a text in <span class="ui-text">.
func Text(t i18n.Ter) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if blank(ctx, t) {
			return emptyValue().Render(ctx, w)
		}
		return textValue("ui-text", t.T(ctx)).Render(ctx, w)
	})
}

// Bool renders a tick or a cross with a hidden Yes or No.
func Bool(v bool) templ.Component { return boolValue(v) }

// Date renders a date as <time class="ui-date">, in the time's own
// location, as i18n.StaticDate writes it.
func Date(t time.Time) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if t.IsZero() {
			return emptyValue().Render(ctx, w)
		}
		return timeValue(t.Format(time.DateOnly), i18n.StaticDate(t).T(ctx)).Render(ctx, w)
	})
}

// DateTime renders a date and time as <time class="ui-date">, as
// 2006-01-02 15:04 in the time's own location.
func DateTime(t time.Time) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if t.IsZero() {
			return emptyValue().Render(ctx, w)
		}
		return timeValue(t.Format(time.RFC3339), t.Format("2006-01-02 15:04")).Render(ctx, w)
	})
}

// Enum renders an option's label in a neutral Badge.
func Enum(o option.SelectOption) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if o == nil {
			return emptyValue().Render(ctx, w)
		}
		label := o.Label()
		if blank(ctx, label) {
			return emptyValue().Render(ctx, w)
		}
		return enumValue(label.T(ctx)).Render(ctx, w)
	})
}

// Link renders <a class="ui-link">.
func Link(href templ.SafeURL, text i18n.Ter) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if strings.TrimSpace(string(href)) == "" || blank(ctx, text) {
			return emptyValue().Render(ctx, w)
		}
		return linkValue(href, text.T(ctx)).Render(ctx, w)
	})
}

// Image renders <img class="ui-image">. A nil alt renders alt=""
// (decorative), never no alt.
func Image(src templ.SafeURL, alt i18n.Ter) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if strings.TrimSpace(string(src)) == "" {
			return emptyValue().Render(ctx, w)
		}
		a := ""
		if alt != nil {
			a = alt.T(ctx)
		}
		return imageValue(src, a).Render(ctx, w)
	})
}

// Amount renders an amount already formatted (ui imports no decimal
// library) in <bdi class="ui-amount">, isolated so a currency stays in
// place in RTL, with tabular numerals.
func Amount(text i18n.Ter) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if blank(ctx, text) {
			return emptyValue().Render(ctx, w)
		}
		return amountValue(text.T(ctx)).Render(ctx, w)
	})
}

// Empty renders a dash with a hidden "None".
func Empty() templ.Component { return emptyValue() }
