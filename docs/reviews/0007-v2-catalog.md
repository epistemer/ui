# Review 0007: V2, the catalogue, catalogtest and uitest's checks

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 42k tokens |
| Under review | The V2 diff, with visual-interface.md §9, §9.1, §13.1 and §1's import rows |
| Scope | The catalogue's API, mounting and CSP; catalogtest against §9.1; uitest's checks against §13.1; the a11y parser; tests |

There were 13 findings. Claude checked each; twelve changed the code.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | The catalogue imports `internal/themehook` | Rejected: internal, not API; recorded as a clarification of §1 in the V2 status entry | — |
| 2 | `title` isn't a name source, though §13.1 lists it | Valid | `title` is the last resort, after content |
| 3 | uitest's `Prefix` adds to `ui-` instead of replacing the default | Valid (§13.1 says "default") | `Prefix` replaces the default; catalogtest still always covers `ui-` |
| 4 | Variant names can traverse paths or collide in golden file names | Valid | catalogtest fails a variant name outside `[a-z0-9-]` and two variants sharing a golden file. `New` keeps §9's rules |
| 5 | catalogtest has no tests of its own | Valid | A passing `Check` with `CSS`, `Prefix` and `External`; the reverse check's helper tested |
| 6 | An empty or exact prefix confuses the class matching | Valid | Empty prefixes are ignored; a class must be longer than its prefix |
| 7 | A `/*` in a string swallows rules; `:is()` and `:where()` arguments are dropped; `~=` and case flags aren't read | Valid | Comments are stripped outside strings only; `:is` and `:where` keep their arguments; attribute operators and flags are read (`^=`, `$=`, `*=` count as presence) |
| 8 | Empty `value` on submit and reset passes; a `<label for>` doesn't name a button; presentational images fail | Valid | Each fixed and tested |
| 9 | `StripPrefix("/catalog/")` leaves paths without a slash, and the mux redirects to the root | Valid | `ServeHTTP` puts the slash back; tested with both mounts |
| 10 | The shared map of used selectors isn't guarded | Valid in principle (subtests are sequential today) | A mutex; the switch's limits documented |
| 11 | The token panel: an error response, a lazy frame's null document, restored values on reload | Valid | Each handled |
| 12 | A failing story writes half a page, then a 500 | Valid | Pages render to a buffer first |
| 13 | The mounted test skips frames and tokens | Valid | Both tested under the mount |
