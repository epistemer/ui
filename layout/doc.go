// Package layout is ui's page frames (visual-interface.md §8): the HTML
// document, the app shell with its navigation drawer and colour scheme
// menu, the auth shell for sign-in pages, and the page header. They build
// on components, elements, icons and the kernel's slots.
//
// A page is a Document around a shell:
//
//	@layout.Document(title).Head(th.Head()).ColorScheme(layout.ColorSchemeOf(r)) {
//		@layout.App() {
//			@layout.AppBrand { Bricks & Co }
//			@layout.AppNav {
//				@layout.NavItem("/orders").Current(here == "/orders") { Orders }
//			}
//			@layout.AppBar {
//				@layout.ColorSchemeMenu(layout.ColorSchemeOf(r))
//			}
//			@layout.AppMain {
//				@layout.PageHeader(title) { …actions… }
//				…
//			}
//		}
//	}
//
// A shell renders the confirmation dialog and the toast region outside
// #ui-main, so a swap of the main content keeps them.
package layout
