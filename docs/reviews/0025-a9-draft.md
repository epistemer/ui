# Review 0025: the draft of amendment A9

| | |
|---|---|
| Under review | [confab-1-interface.md](../confab-1-interface.md), first draft |
| Reviewer | Gemini 4 Carbon (High), through agy (`adversarial_review`), about 131k tokens, on the draft and an excerpt of the decisions and house rules |
| Date | 2026-11-24 |

18 findings; what changed in the draft:

| # | Finding | Verdict | Outcome |
|---|---|---|---|
| 1 | The icon generator checked element names only: an attribute value could break out into generated code; `-fetch` had no integrity source or tarball limits | Valid | Per-attribute value grammars, no DTD or entities, shapes written from parsed values with `strconv.Quote`; `-fetch` checks npm's `sha512` integrity, reads two paths, refuses `..`, links and oversize entries |
| 2 | A flash set without handlers' middleware, or a typed nil, panics after the work | Partly | Wiring rule stated (inside `Flash.Middleware`, beside `actions.Gate`); the import-cycle claim is wrong (`components` already imports `ui`) |
| 3 | The upgraded switch lost the select's help, error, `aria-invalid` and `required`; changes weren't announced; arrows don't reach it in browse mode | Valid | The button takes them; a live region says the state; Space and clicks are the main path |
| 4 | `required` on a hidden select: the browser's message can't show | Valid | `required` moves to `aria-required`; the server's `ErrRequired` says it |
| 5 | Flashing `Fail`'s error leaks internals, spoofs, overflows the cookie, doubles an app's own flash | Partly | The message is `errors.T` (private errors read generic); size is handlers'; the double message is listed in §7 |
| 6 | `PostCode("gb")` silently passes everything; value case unspecified | Valid | Country compared trimmed and upper-cased; values matched in upper case |
| 7 | Boosted requests' column unspecified; a targeted boost drops the flash | Valid | Stated: `HX-Request` takes the htmx column; a boost not of the body doesn't show flashes |
| 8 | Flashes outliving a logout; `HX-Redirect` safety | Partly | `Go` keeps refusing non-local addresses; a flash's lifetime is handlers' |
| 9 | Start/End naming ambiguous; glyphs that mustn't mirror; digit names; geometry attributes undefined | Valid | Word-based rule with examples, an exclusion list, an exact attribute table |
| 10 | Marks drawn by CSS `data:` URLs break a strict CSP; no forced-colors rules | Valid | Marks are ui's icons cloned from a `<template>`; forced-colors stated |
| 11 | Double upgrade after htmx's history restore; stale state after reset | Valid | Idempotent upgrade; re-reads on `input`, `change` and after `reset` |
| 12 | Gate panics on error pages; empty nav; crumbs' premise | Partly | Error pages need a gate (stated); empty nav stays as an empty fill; the crumb sentence rewritten |
| 13 | `Stroke(NaN)`, zero values, owned attributes, wire values | Valid | All specified |
| 14 | ~2,400 templ components are ~100k generated lines | Valid | Q7: plain Go by default |
| 15 | Breaks list missed inferred variables and generics | Valid | Added |
| 16 | A click on the start side of an unset switch went to Yes | Valid | The pointer's third chooses the state |
| 17 | Neutral toasts turned info after a redirect; Toast with Fail | Invalid / valid | An info toast has no tone, so the mapping is lossless; Toast with Fail stated |
| 18 | JSON texts duplicate the options' words | Valid | Words from the options, one `data-ui-unset` |

After the review, Claude checked the draft against the 1.77.0 files: the
allowlist lacked `fill="currentColor"` on 34 circles, four name
collisions are aliases with the same shapes, the 22 current icons are
unchanged, and the GB (and CR) patterns do accept junk around a code.
The script's size (24.2 kB gzipped against §10's 12 kB) became Q8.
