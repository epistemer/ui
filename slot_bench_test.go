package ui_test

import (
	"context"
	"io"
	"testing"

	"github.com/epistemer/ui/internal/kerneltest"
)

func BenchmarkThreeSlots(b *testing.B) {
	c := kerneltest.ThreeSlotsFilled()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := c.Render(ctx, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNestedFive(b *testing.B) {
	c := kerneltest.NestedFive()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := c.Render(ctx, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFillSize(b *testing.B) {
	b.Run("small", func(b *testing.B) {
		c := kerneltest.SmallFill()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := c.Render(ctx, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("100x", func(b *testing.B) {
		c := kerneltest.LargeFill()
		ctx := context.Background()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if err := c.Render(ctx, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
}
