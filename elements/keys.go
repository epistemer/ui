package elements

import "github.com/epistemer/i18n"

// ui's own text (§2.5, §11). The keys and their English texts are the
// contract; an app changes them through translations.
var (
	alertInfo      = i18n.Key("ui.elements.alert.info", "Information:")
	alertSuccess   = i18n.Key("ui.elements.alert.success", "Success:")
	alertWarning   = i18n.Key("ui.elements.alert.warning", "Warning:")
	alertDanger    = i18n.Key("ui.elements.alert.danger", "Error:")
	boolYes        = i18n.Key("ui.elements.bool.yes", "Yes")
	boolNo         = i18n.Key("ui.elements.bool.no", "No")
	emptyNone      = i18n.Key("ui.elements.empty.none", "None")
	spinnerLoading = i18n.Key("ui.elements.spinner.loading", "Loading")
)
