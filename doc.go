// Package ui is the kernel of neozp's UI packages: typed context values
// (Provider), element IDs that wire elements together without colliding
// across separately rendered fragments (ID), and named slots for inline
// content (Slot, SlotOf). It imports templ and the standard library only.
//
// The test kit is github.com/epistemer/ui/uitest. Forms live in
// github.com/epistemer/ui/forms, which imports no templ.
//
// The design is docs/brief.md and the contract docs/interface.md.
package ui
