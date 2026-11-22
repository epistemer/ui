# Review 0013: A5, links

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 3.8 Flash (High), through agy (`adversarial_review`), about 207k tokens |
| Under review | The A5 diff (the kernel's `Link` and `URL`, `internal/links`, the eight signatures, their callers and tests), with interface.md §4A and visual-interface.md §2.7 and §13.4 |
| Scope | The signatures, what no address does, sanitising, the kernel's imports, tests |

There were 5 findings. Claude checked each; four changed the code. The
reviewer confirmed that exactly the eight signatures changed, that the
kernel still imports only templ and the standard library, and that the
owned `href` keys stop `Attrs` from bypassing the sanitising.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | A typed nil pointer passed as a link panics in `URL()` | Valid: it is a nil link (§2.7) | `links.Href` treats a nil pointer as no address |
| 2 | A crumb's address is resolved twice, and the last one only while writing | Valid | Every crumb is resolved once, before the header writes anything |
| 3 | Some components resolve a link when they're built, `Link` and `Image` when they render | Valid | `Link` and `Image` resolve when called too |
| 4 | `data:` and `blob:` images no longer render, which review 0008 allowed | Rejected: A5 makes them no address on purpose. §2.7 counts an address `templ.URL` refuses as none, and ADR 0008 records the consequence: no scheme outside templ's list can be passed as trusted, and no page needs one. A served image gets its address from a route. Review 0008's allowance is superseded | Recorded; tested in `internal/links` |
| 5 | Gaps in the no-address tests | Valid | Nil, empty and refused cases for Tab, MenuItem, NavItem and the crumbs |
