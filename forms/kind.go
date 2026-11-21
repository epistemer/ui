package forms

// Kind is what a node is, for a renderer to pick a widget. The set is
// closed, so a switch over it can be checked for exhaustiveness. Each
// constructor sets a default; Field.Hint overrides it.
type Kind uint8

// The kinds.
const (
	KindText Kind = iota + 1
	KindMultiline
	KindPassword
	KindHidden
	KindNumber
	KindCheckbox
	KindToggle
	KindDate
	KindDateTime
	KindSelect
	KindMultiSelect
	KindFile
	KindFiles
	KindGroup
	KindList
)

var kindNames = [...]string{
	KindText: "text", KindMultiline: "multiline", KindPassword: "password",
	KindHidden: "hidden", KindNumber: "number", KindCheckbox: "checkbox",
	KindToggle: "toggle", KindDate: "date", KindDateTime: "datetime",
	KindSelect: "select", KindMultiSelect: "multiselect", KindFile: "file",
	KindFiles: "files", KindGroup: "group", KindList: "list",
}

// String returns the kind's name without its prefix, in lower case, such
// as "multiselect"; "" for a value outside the set.
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return ""
}
