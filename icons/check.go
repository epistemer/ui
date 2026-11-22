package icons

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

func checkLabelled(ctx context.Context, icon templ.Component, text i18n.Ter) {
	if icon == nil {
		panic("icons: Labelled with a nil icon")
	}
	if text == nil || strings.TrimSpace(text.T(ctx)) == "" {
		panic("icons: Labelled with no text: a labelled icon needs its name")
	}
}
