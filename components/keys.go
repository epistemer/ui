package components

import "github.com/epistemer/i18n"

// ui's own text (§2.5, §11).
var (
	dialogClose  = i18n.Key("ui.components.dialog.close", "Close")
	confirmTitle = i18n.Key("ui.components.confirm.title", "Are you sure?")
	confirmYes   = i18n.Key("ui.components.confirm.yes", "Confirm")
	confirmNo    = i18n.Key("ui.components.confirm.no", "Cancel")
	toastClose   = i18n.Key("ui.components.toast.close", "Dismiss")
)

// A toast's hidden level words are Alert's: the same keys, one translation.
var (
	toastInfo    = i18n.Key("ui.elements.alert.info", "Information:")
	toastSuccess = i18n.Key("ui.elements.alert.success", "Success:")
	toastWarning = i18n.Key("ui.elements.alert.warning", "Warning:")
	toastDanger  = i18n.Key("ui.elements.alert.danger", "Error:")
)
