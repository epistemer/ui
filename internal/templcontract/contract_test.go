package templcontract

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// renders a Go component that renders its block. clear says whether it
// calls templ.ClearChildren first, as invariant K3 requires.
func renders(clear bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		block := templ.GetChildren(ctx)
		if clear {
			ctx = templ.ClearChildren(ctx)
		}
		return block.Render(ctx, w)
	})
}

var errRecursed = errors.New("block rendered again: children leaked")

// TestChildrenLeakWithoutClear pins K3: a Go component that renders its
// block without clearing the children leaks that block to a generated
// component inside it that renders children but is called without a block;
// clearing stops the leak.
func TestChildrenLeakWithoutClear(t *testing.T) {
	for _, tc := range []struct {
		clear bool
		leaks bool
	}{
		{clear: false, leaks: true},
		{clear: true, leaks: false},
	} {
		runs := 0
		block := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			runs++
			if runs > 1 {
				return errRecursed
			}
			return Child().Render(ctx, w)
		})
		var buf bytes.Buffer
		err := renders(tc.clear).Render(templ.WithChildren(context.Background(), block), &buf)
		if got := errors.Is(err, errRecursed); got != tc.leaks {
			t.Errorf("clear=%v: leaked=%v (err %v, output %q), want leaked=%v", tc.clear, got, err, buf.String(), tc.leaks)
		}
		if !tc.leaks && buf.String() != "<i></i>" {
			t.Errorf("clear=%v: output %q, want %q", tc.clear, buf.String(), "<i></i>")
		}
	}
}

// TestBlockOfCallsWritesOnlyWhiteSpace pins what the slot collect pass
// relies on: a block made only of component calls, if and for writes
// nothing but white space of its own.
func TestBlockOfCallsWritesOnlyWhiteSpace(t *testing.T) {
	silent := templ.ComponentFunc(func(context.Context, io.Writer) error { return nil })
	for _, tc := range []struct {
		n     int
		items []string
	}{
		{0, nil},
		{1, nil},
		{2, []string{"a", "b", "c"}},
	} {
		var buf bytes.Buffer
		if err := OnlyCalls(silent, tc.n, tc.items).Render(context.Background(), &buf); err != nil {
			t.Fatal(err)
		}
		if s := buf.String(); strings.TrimSpace(s) != "" {
			t.Errorf("n=%d items=%v: block wrote %q", tc.n, tc.items, s)
		}
	}
}
