package ui_test

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/a-h/templ"
	"github.com/epistemer/ui"
)

func ExampleProvider() {
	currency := ui.NewProvider("USD")

	ctx := context.Background()
	fmt.Println(currency.Value(ctx))

	ctx = currency.With(ctx, "EUR")
	fmt.Println(currency.Value(ctx))

	// Output:
	// USD
	// EUR
}

func ExampleID() {
	// Key returns a stable id built from data, needing no context.
	orderRow := ui.NewID("order-row")
	fmt.Println(orderRow.Key("1042"))

	// Get and Part return scoped ids inside a Scope.
	// ui.WithIDs with ui.Sequence provides deterministic tokens for stable output.
	emailID := ui.NewID("email")
	scope := ui.Scope(emailID)

	child := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Println(emailID.Get(ctx))
		fmt.Println(emailID.Part(ctx, "hint"))
		return nil
	})

	ctx := ui.WithIDs(context.Background(), ui.Sequence("ex"))
	_ = scope.Render(templ.WithChildren(ctx, child), io.Discard)

	// Output:
	// order-row-1042
	// email--ex-1
	// email--ex-1-hint
}

func ExampleSlot() {
	headerSlot := ui.NewSlot()
	footerSlot := ui.NewSlot()

	layout := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintln(w, "<article>")
		if headerSlot.Filled(ctx) {
			fmt.Fprintln(w, "  <!-- custom header -->")
		}
		headerFallback := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintln(w, "  <header>Default Header</header>")
			return nil
		})
		if err := headerSlot.Place().Render(templ.WithChildren(ctx, headerFallback), w); err != nil {
			return err
		}

		footerFallback := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintln(w, "  <footer>Default Footer</footer>")
			return nil
		})
		if err := footerSlot.Place().Render(templ.WithChildren(ctx, footerFallback), w); err != nil {
			return err
		}
		fmt.Fprintln(w, "</article>")
		return nil
	})

	card := ui.Slotted(layout, headerSlot, footerSlot)

	// Caller block fills headerSlot, replacing its fallback, and leaves footerSlot unfilled.
	headerFill := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintln(w, "  <header>Custom Header</header>")
		return nil
	})
	callerBlock := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return headerSlot.Render(templ.WithChildren(ctx, headerFill), w)
	})

	_ = card.Render(templ.WithChildren(context.Background(), callerBlock), os.Stdout)

	// Output:
	// <article>
	//   <!-- custom header -->
	//   <header>Custom Header</header>
	//   <footer>Default Footer</footer>
	// </article>
}

func ExampleSlotted() {
	headerSlot := ui.NewSlot()
	footerSlot := ui.NewSlot()

	layout := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintln(w, "<article>")
		if headerSlot.Filled(ctx) {
			fmt.Fprintln(w, "  <!-- custom header -->")
		}
		headerFallback := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintln(w, "  <header>Default Header</header>")
			return nil
		})
		if err := headerSlot.Place().Render(templ.WithChildren(ctx, headerFallback), w); err != nil {
			return err
		}

		footerFallback := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			fmt.Fprintln(w, "  <footer>Default Footer</footer>")
			return nil
		})
		if err := footerSlot.Place().Render(templ.WithChildren(ctx, footerFallback), w); err != nil {
			return err
		}
		fmt.Fprintln(w, "</article>")
		return nil
	})

	card := ui.Slotted(layout, headerSlot, footerSlot)

	headerFill := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fmt.Fprintln(w, "  <header>Custom Header</header>")
		return nil
	})
	callerBlock := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return headerSlot.Render(templ.WithChildren(ctx, headerFill), w)
	})

	_ = card.Render(templ.WithChildren(context.Background(), callerBlock), os.Stdout)

	// Output:
	// <article>
	//   <!-- custom header -->
	//   <header>Custom Header</header>
	//   <footer>Default Footer</footer>
	// </article>
}
