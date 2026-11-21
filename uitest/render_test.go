package uitest_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
	"github.com/epistemer/ui/internal/kerneltest"
	"github.com/epistemer/ui/uitest"
)

func TestRender(t *testing.T) {
	if got := uitest.Render(t, kerneltest.NestedScopes()); got != "<a>email--t-1</a><b>email--t-2</b><c>email--t-1</c>" {
		t.Errorf("got %s", got)
	}
	provide := uitest.Context(func(ctx context.Context) context.Context { return kerneltest.Currency.With(ctx, "EGP") })
	if got := uitest.Render(t, kerneltest.Price(), provide); got != "<b>EGP</b>" {
		t.Errorf("with a provider: %s", got)
	}
	own := uitest.Context(func(ctx context.Context) context.Context { return ui.WithIDs(ctx, ui.Sequence("own")) })
	if got := uitest.Render(t, kerneltest.Siblings(), own); got != "<a>email--own-1</a><b>email--own-2</b><c>email--own-3 hint--own-4</c>" {
		t.Errorf("a step's WithIDs must override: %s", got)
	}
}

func TestRenderErr(t *testing.T) {
	boom := errors.New("boom")
	_, err := uitest.RenderErr(templ.ComponentFunc(func(context.Context, io.Writer) error { return boom }))
	if !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestContextOrderAndNil(t *testing.T) {
	first := uitest.Context(func(ctx context.Context) context.Context { return kerneltest.Currency.With(ctx, "AAA") })
	second := uitest.Context(func(ctx context.Context) context.Context { return kerneltest.Currency.With(ctx, "BBB") })
	if got := uitest.Render(t, kerneltest.Price(), first, nil, second); got != "<b>BBB</b>" {
		t.Errorf("got %s: steps run in order, and a nil Option does nothing", got)
	}
	if got, err := uitest.RenderErr(kerneltest.Price(), second); err != nil || got != "<b>BBB</b>" {
		t.Errorf("RenderErr with options: %s %v", got, err)
	}
	defer func() {
		if recover() == nil {
			t.Error("Context(nil) did not panic")
		}
	}()
	uitest.Context(nil)
}

func TestRenderUsesTheTestContext(t *testing.T) {
	var done <-chan struct{}
	probe := templ.ComponentFunc(func(ctx context.Context, _ io.Writer) error { done = ctx.Done(); return nil })
	uitest.Render(t, probe)
	if done == nil {
		t.Error("Render's context is not the test's")
	}
}
