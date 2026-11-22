package components

import (
	"context"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/epistemer/i18n"

	"github.com/epistemer/ui/elements"
)

// DetailListSpec is a list of labelled values (§7.6): <dl
// class="ui-detail-list">, holding DetailRow components. Rows put the label
// beside the value where the container is wide enough, above it where it
// isn't.
type DetailListSpec struct{ attrs templ.Attributes }

// DetailList starts a detail list.
func DetailList() DetailListSpec { return DetailListSpec{} }

// Attrs adds attributes to the <dl>, merged as elements.Merge does.
func (d DetailListSpec) Attrs(a templ.Attributes) DetailListSpec {
	d.attrs = elements.Merge(d.attrs, nil, a)
	return d
}

func (d DetailListSpec) Render(ctx context.Context, w io.Writer) error {
	return detailList(elements.Merge(templ.Attributes{"class": "ui-detail-list"}, nil, d.attrs)).Render(ctx, w)
}

// DetailRowSpec is one row: a label and its value, the children, usually a
// value renderer of elements.
type DetailRowSpec struct {
	label i18n.Ter
	attrs templ.Attributes
}

// DetailRow starts a row labelled label.
func DetailRow(label i18n.Ter) DetailRowSpec { return DetailRowSpec{label: label} }

// Attrs adds attributes to the row's <div>, merged as elements.Merge does.
func (d DetailRowSpec) Attrs(a templ.Attributes) DetailRowSpec {
	d.attrs = elements.Merge(d.attrs, nil, a)
	return d
}

func (d DetailRowSpec) Render(ctx context.Context, w io.Writer) error {
	if d.label == nil || strings.TrimSpace(d.label.T(ctx)) == "" {
		panic("components: DetailRow with no label: a row needs its label")
	}
	return detailRow(elements.Merge(templ.Attributes{"class": "ui-detail-row"}, nil, d.attrs), d.label).Render(ctx, w)
}
