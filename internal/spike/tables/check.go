package tables

import "fmt"

// Check reports what needs the whole declaration: a row label or default
// sort that isn't one of the table's columns, a column without a label.
func (t *Table[R]) Check() error {
	if t.rowLabel != "" {
		if _, ok := t.column(t.rowLabel); !ok {
			return fmt.Errorf("tables: %s: RowLabel %q isn't a column of the table", t.name, t.rowLabel)
		}
	}
	if t.sort != "" {
		if _, ok := t.column(t.sort); !ok {
			return fmt.Errorf("tables: %s: Sort %q isn't a column of the table", t.name, t.sort)
		}
	}
	if t.label == nil {
		return fmt.Errorf("tables: %s has no Label: its caption names it", t.name)
	}
	for _, c := range t.cols {
		if c.c.Label == nil {
			return fmt.Errorf("tables: %s: column %q has no label", t.name, c.c.Name)
		}
	}
	return nil
}
