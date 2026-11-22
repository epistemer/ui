package ui

// Link is anything with a URL: a route link from handlers' generated
// builders, or URL's adapter for a literal or external address
// (interface.md §4A, ADR 0008). It is an alias of the bare interface, so a
// function another package declares over interface{ URL() string } is
// assignable where ui wants one over Link, with no import either way.
//
// The visual packages take a Link wherever they take an address and render
// it through templ.URL, which lets http, https, mailto, tel and relative
// addresses through. A nil link, one whose URL is empty, and one templ.URL
// refuses are no address (visual-interface.md §2.7).
type Link = interface{ URL() string }

// URL makes a Link of a literal or external address: URL(s).URL() is s,
// and URL("") is a link with no address.
func URL(s string) Link { return literal(s) }

type literal string

func (l literal) URL() string { return string(l) }
