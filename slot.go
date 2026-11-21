package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/a-h/templ"
)

// Errors a slotted component's render returns. Test for them with
// errors.Is; each message names the slot by the file and line of its
// NewSlot, or the wrapper written in the wrong place.
var (
	// ErrStrayContent: the block wrote something other than white space
	// outside a fill.
	ErrStrayContent = errors.New("ui: content outside a slot fill in a slotted component's block")
	// ErrFilledTwice: a block filled the same slot twice.
	ErrFilledTwice = errors.New("ui: slot filled twice")
	// ErrNotDeclared: a block filled, or a layout placed, a slot the
	// component did not list in Slotted.
	ErrNotDeclared = errors.New("ui: slot not listed in Slotted")
	// ErrNoComponent: a slot was filled outside any slotted component's
	// block, or placed outside a layout.
	ErrNoComponent = errors.New("ui: slot used outside a slotted component")
	// ErrInBlock: Provide, Scope or Place was written directly in a
	// slotted component's block, where its values would never reach the
	// fills. Open it around the component call instead.
	ErrInBlock = errors.New("ui: wrapper written directly in a slotted component's block")
)

// Slot is a named place in a component that a caller fills with inline
// content. Declare it once, in a package-level variable, list it in
// Slotted, and place it in the layout with Place:
//
//	var CartBody = ui.NewSlot()
//
// A caller fills it by rendering it inside the component's block:
//
//	@shop.CartModal("Your cart") {
//		@shop.CartBody {
//			...
//		}
//	}
type Slot struct{ h *slotHandle }

// SlotOf is a slot that passes a T from the component to whoever fills
// it, read with Value.
type SlotOf[T any] struct{ h *slotHandle }

// SlotRef is a slot of either kind, as Slotted lists them.
type SlotRef interface{ slot() *slotHandle }

type slotHandle struct {
	pos string // file:line of NewSlot, for messages
}

func (s *Slot) slot() *slotHandle      { return s.h }
func (s *SlotOf[T]) slot() *slotHandle { return s.h }

// NewSlot declares a slot that passes no value.
//
// It is never inlined: inlined into a package's initializer, its caller's
// line would be lost from error messages.
//
//go:noinline
func NewSlot() *Slot { return &Slot{newSlotHandle()} }

// NewSlotOf declares a slot that passes a T to its fill. Like NewSlot, it
// is never inlined.
//
//go:noinline
func NewSlotOf[T any]() *SlotOf[T] { return &SlotOf[T]{newSlotHandle()} }

// newSlotHandle records where NewSlot or NewSlotOf was called. Callers
// and CallersFrames, unlike Caller, see through inlining, which package
// initializers get.
func newSlotHandle() *slotHandle {
	pcs := make([]uintptr, 8)
	n := runtime.Callers(2, pcs) // skip Callers and newSlotHandle
	frames := runtime.CallersFrames(pcs[:n])
	for {
		f, more := frames.Next()
		if !strings.HasSuffix(f.Function, "ui.NewSlot") && !strings.Contains(f.Function, "ui.NewSlotOf[") {
			return &slotHandle{pos: fmt.Sprintf("%s:%d", f.File, f.Line)}
		}
		if !more {
			return &slotHandle{pos: "unknown"}
		}
	}
}

func (h *slotHandle) errorf(sentinel error, format string, a ...any) error {
	return &slotError{sentinel, fmt.Sprintf("ui: slot declared at %s: ", h.pos) + fmt.Sprintf(format, a...)}
}

type slotError struct {
	sentinel error
	msg      string
}

func (e *slotError) Error() string { return e.msg }
func (e *slotError) Unwrap() error { return e.sentinel }

// Slotted makes a slotted component: layout renders the component, and
// slots lists every slot its block may fill. The caller's block runs once,
// holds only fills and white space, and is never output itself; layout
// then renders straight to the writer (brief §7). It panics on a nil
// layout or slot, or a slot listed twice.
func Slotted(layout templ.Component, slots ...SlotRef) templ.Component {
	if layout == nil {
		panic("ui: Slotted(nil, ...)")
	}
	declared := make(map[*slotHandle]bool, len(slots))
	for _, s := range slots {
		if s == nil || s.slot() == nil {
			panic("ui: Slotted lists a nil slot")
		}
		if declared[s.slot()] {
			panic("ui: Slotted lists the slot declared at " + s.slot().pos + " twice")
		}
		declared[s.slot()] = true
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		children, ctx := block(ctx)
		if c := collectorOf(ctx); c != nil {
			// A slotted component written directly in another's block is
			// stray content, never a fill.
			return &slotError{ErrStrayContent, "ui: a slotted component written directly in another slotted component's block; wrap it in a slot fill"}
		}
		fr := &frame{declared: declared, fills: map[*slotHandle]*fill{}}
		c := &collector{frame: fr, lexical: frameOf(ctx)}
		sink := &sink{}
		if err := children.Render(context.WithValue(ctx, collectorKey{}, c), sink); err != nil {
			if sink.stray != "" {
				return &slotError{ErrStrayContent, fmt.Sprintf("ui: a slotted component's block wrote %q outside any fill", sink.stray)}
			}
			return err
		}
		return layout.Render(context.WithValue(ctx, frameKey{}, fr), w)
	})
}

// frame is one render of a slotted component: its slots and their fills.
type frame struct {
	declared map[*slotHandle]bool
	fills    map[*slotHandle]*fill
}

type fill struct {
	content templ.Component
	lexical *frame // the frame where the fill was written, for rule 5
}

type collector struct {
	frame   *frame
	lexical *frame
}

type (
	collectorKey struct{}
	frameKey     struct{}
	valueKey     struct{ h *slotHandle }
)

func collectorOf(ctx context.Context) *collector {
	c, _ := ctx.Value(collectorKey{}).(*collector)
	return c
}

func frameOf(ctx context.Context) *frame {
	f, _ := ctx.Value(frameKey{}).(*frame)
	return f
}

// inBlock returns ErrInBlock, naming what, when ctx is a block's collect
// pass.
func inBlock(ctx context.Context, what string) error {
	if collectorOf(ctx) != nil {
		return fmt.Errorf("%w: %s; open it around the component call", ErrInBlock, what)
	}
	return nil
}

// sink keeps nothing and refuses anything but HTML white space.
type sink struct{ stray string }

func (s *sink) Write(p []byte) (int, error) {
	for i, c := range p {
		switch c {
		case ' ', '\t', '\n', '\f', '\r':
		default:
			rest := p[i:]
			if len(rest) > 40 {
				rest = rest[:40]
			}
			s.stray = string(rest)
			return i, ErrStrayContent
		}
	}
	return len(p), nil
}

// record fills h in the block being collected.
func record(ctx context.Context, h *slotHandle) error {
	c := collectorOf(ctx)
	if c == nil {
		return h.errorf(ErrNoComponent, "filled outside any slotted component's block")
	}
	if !c.frame.declared[h] {
		return h.errorf(ErrNotDeclared, "filled in a component that doesn't list it in Slotted")
	}
	if c.frame.fills[h] != nil {
		return h.errorf(ErrFilledTwice, "filled twice in one block")
	}
	c.frame.fills[h] = &fill{content: templ.GetChildren(ctx), lexical: c.lexical}
	return nil
}

// Render fills the slot: rendered in a slotted component's block, it
// records its own block as the fill and writes nothing.
func (s *Slot) Render(ctx context.Context, _ io.Writer) error { return record(ctx, s.h) }

// Render fills the slot, like Slot.Render.
func (s *SlotOf[T]) Render(ctx context.Context, _ io.Writer) error { return record(ctx, s.h) }

// Place renders the caller's fill if there is one, even one that renders
// nothing; otherwise this call's own block, the fallback.
func (s *Slot) Place() templ.Component { return place(s.h, nil) }

// Place renders the caller's fill, or this call's own block, with v as the
// slot's Value.
func (s *SlotOf[T]) Place(v T) templ.Component {
	return place(s.h, func(ctx context.Context) context.Context {
		return context.WithValue(ctx, valueKey{s.h}, provided[T]{v})
	})
}

func place(h *slotHandle, withValue func(context.Context) context.Context) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		fallback, ctx := block(ctx)
		if err := inBlock(ctx, "Place of the slot declared at "+h.pos); err != nil {
			return err
		}
		fr := frameOf(ctx)
		if fr == nil {
			return h.errorf(ErrNoComponent, "placed outside a layout")
		}
		if !fr.declared[h] {
			return h.errorf(ErrNotDeclared, "placed in a layout whose component doesn't list it in Slotted")
		}
		if withValue != nil {
			ctx = withValue(ctx)
		}
		f := fr.fills[h]
		if f == nil {
			return fallback.Render(ctx, w)
		}
		// Rule 4: the fill sees the placement's context. Rule 5: its slot
		// lookups resolve where it was written.
		return f.content.Render(context.WithValue(ctx, frameKey{}, f.lexical), w)
	})
}

// Filled reports whether the current component's caller filled the slot.
// It panics outside a layout, or for a slot the component doesn't list.
func (s *Slot) Filled(ctx context.Context) bool { return filled(ctx, s.h) }

// Filled reports whether the current component's caller filled the slot,
// like Slot.Filled.
func (s *SlotOf[T]) Filled(ctx context.Context) bool { return filled(ctx, s.h) }

func filled(ctx context.Context, h *slotHandle) bool {
	fr := frameOf(ctx)
	if fr == nil {
		panic(h.errorf(ErrNoComponent, "Filled outside a layout").Error())
	}
	if !fr.declared[h] {
		panic(h.errorf(ErrNotDeclared, "Filled in a layout whose component doesn't list it").Error())
	}
	return fr.fills[h] != nil
}

// Value returns the value of the nearest placement of the slot, in its
// fill or in its fallback. It panics outside one.
func (s *SlotOf[T]) Value(ctx context.Context) T {
	b, ok := ctx.Value(valueKey{s.h}).(provided[T])
	if !ok {
		panic(s.h.errorf(ErrNoComponent, "Value outside a placement of the slot").Error())
	}
	return b.v
}
