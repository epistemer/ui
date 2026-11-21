package ui_test

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/kerneltest"
)

func render(t testing.TB, ctx context.Context, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestProvider(t *testing.T) {
	bg := context.Background()
	for _, tc := range []struct {
		name string
		ctx  context.Context
		c    templ.Component
		want string
	}{
		{"default", bg, kerneltest.Price(), "<b>USD</b>"},
		{"nesting", bg, kerneltest.Nested(), "<b>USD</b><b>EGP</b><b>EUR</b><b>EGP</b><b>USD</b>"},
		{"three components down", bg, kerneltest.Deep(), "<div><p><b>JPY</b></p></div>"},
		{"With from Go", kerneltest.Currency.With(bg, "GBP"), kerneltest.Price(), "<b>GBP</b>"},
		{"Provide inside With", kerneltest.Currency.With(bg, "GBP"), kerneltest.Nested(), "<b>GBP</b><b>EGP</b><b>EUR</b><b>EGP</b><b>GBP</b>"},
		{"providers of one type are apart", bg, kerneltest.Labelled(), "<b>USD</b><s>sale</s>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := render(t, tc.ctx, tc.c); got != tc.want {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
}

// TestProvideClearsChildren is M1.2's done-when: a component that renders
// its children, called without a block inside Provide, renders nothing.
func TestProvideClearsChildren(t *testing.T) {
	if got := render(t, context.Background(), kerneltest.ChildInProvide()); got != "<i></i>" {
		t.Errorf("got %q, want %q", got, "<i></i>")
	}
}

func TestProviderZeroSizedTypes(t *testing.T) {
	a, b := ui.NewProvider(struct{}{}), ui.NewProvider(struct{}{})
	if a == b {
		t.Fatal("two providers share one handle")
	}
	type marker struct{}
	p := ui.NewProvider[*marker](nil)
	q := ui.NewProvider[*marker](nil)
	m := &marker{}
	if q.Value(p.With(context.Background(), m)) != nil {
		t.Error("a value given to one provider reached another of the same type")
	}
}

func TestProviderConcurrentRenders(t *testing.T) {
	var wg sync.WaitGroup
	for _, cur := range []string{"AAA", "BBB", "CCC", "DDD"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := kerneltest.Currency.Provide(cur)
			for range 200 {
				var buf bytes.Buffer
				ctx := templ.WithChildren(context.Background(), kerneltest.Price())
				if err := c.Render(ctx, &buf); err != nil {
					t.Error(err)
					return
				}
				if got := buf.String(); got != "<b>"+cur+"</b>" {
					t.Errorf("got %q, want %s", got, cur)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestProvideRenderErrorReturned(t *testing.T) {
	boom := templ.ComponentFunc(func(context.Context, io.Writer) error { return io.ErrShortWrite })
	err := kerneltest.Currency.Provide("X").Render(templ.WithChildren(context.Background(), boom), io.Discard)
	if err != io.ErrShortWrite {
		t.Errorf("got %v, want the block's error (K6)", err)
	}
}
