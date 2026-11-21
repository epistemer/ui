// Package ui is the kernel of neozp's UI packages: typed context values
// (Provider), element IDs that wire elements together without colliding
// across separately rendered fragments (ID), and named slots for inline
// content (Slot, SlotOf). It imports templ and the standard library only.
//
// The test kit is github.com/epistemer/ui/uitest. Forms live in
// github.com/epistemer/ui/forms, which imports no templ.
//
// # Context values
//
// A [Provider] passes a typed value down a component subtree without
// threading it as a parameter:
//
//	@Currency.Provide("USD") {
//		@Price(p)
//	}
//
// In Go code, [Provider.With] sets the value on a context, and
// [Provider.Value] reads the nearest provided value (or the default).
//
// # Element IDs
//
// An [ID] gives unique generated IDs for wiring elements within a render,
// and stable keyed IDs for elements targeted across requests:
//
//	@ui.Scope(EmailID) {
//		<label for={ EmailID.Get(ctx) }>Email</label>
//		<input id={ EmailID.Get(ctx) } aria-describedby={ EmailID.Part(ctx, "hint") }/>
//	}
//
// Generated IDs ([ID.Get], [ID.Part]) are unique per [Scope]. Keyed IDs
// ([ID.Key]) are stable across renders and need no context.
//
// # Slots
//
// Slots let a caller place inline content into named places of a component
// wrapped with [Slotted]:
//
//	@shop.CartModal("Your cart") {
//		@shop.CartBody {
//			<p>Your cart has 2 items.</p>
//		}
//	}
//
// The component places the slot with [Slot.Place] (supplying fallback
// content in its block) or checks [Slot.Filled] to conditionally render
// enclosing tags. Valued slots ([SlotOf]) pass data to their fills via
// [SlotOf.Value].
//
// The design is docs/brief.md and the contract docs/interface.md.
package ui
