package forms_test

import (
	"fmt"

	"github.com/epistemer/i18n"
)

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }

func i18nKey(key, text string) i18n.TKer { return i18n.Key(key, text) }
