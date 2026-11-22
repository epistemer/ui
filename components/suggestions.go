package components

import (
	"github.com/a-h/templ"
	"github.com/epistemer/i18n"
)

// Suggestion is one answer of a search: an id and its name (R5).
type Suggestion struct {
	Value string
	Label i18n.Ter
}

// SuggestionLimit is how many suggestions an answer shows. A handler
// fetches one more, so Suggestions can tell there were more.
const SuggestionLimit = 20

// Suggestions renders a search's answer, which the combobox hx-selects:
// <ul data-ui-suggestions> of <li role="option" aria-selected="false"
// data-value> with the label as text, the first SuggestionLimit, then
// <li data-ui-more hidden> when there were more. No ids: the script gives
// them, unique per combobox, after the swap.
func Suggestions(s ...Suggestion) templ.Component {
	return suggestions(s)
}
