# Review 0006: V1, `ui/theme` and themeschema

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 70k tokens |
| Under review | The V1 diff (renames detected), with visual-interface.md §1 and §3 |
| Scope | The exported API against §3; the checks; `New`, `ServeHTTP`, `Head`; themeschema; the internal hook; tests and CI |

There were 10 findings, all major or minor. Claude checked each against
the code. Nine changed it; one is recorded as a clarification.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | themeschema's `replace` and pseudo-version can't be published | Valid for the release | V7's done-when sets the order: tag ui, point themeschema at it and drop the `replace`, then tag themeschema |
| 2 | `CheckFont` refuses `default`, which §3.2 doesn't list | Valid | Removed: the check matches §3.2's five keywords exactly |
| 3 | Radius boundaries untested | Valid | Tests for `0px`, `32.000px`, `32.001px`, `2.001rem`, `100px`, `0.1234rem` |
| 4 | A nil `Config` passes `New` and panics at render | Valid | `New` returns an error |
| 5 | themeschema's test covers three keys, without the errors or mutability | Valid | All eight keys refuse bad values with theme's errors; an update stores all eight and refuses a bad radius |
| 6 | The prefix allows `//` after the start, and control characters | Valid | §3.3 says no `//` anywhere: checked with `Contains`, and every byte below `!` is refused |
| 7 | The catalogue's use of `themehook.Tokens` isn't in §1's import table; `Stylesheet` rebuilds the bundle on every call | Half valid. The hook is internal, not API; §1 names it for catalogtest, and V2 records the catalogue's use as a clarification | The default bundle is built once (`sync.OnceValues`) and shared by `New` and the hook |
| 8 | The stylesheet's cache key and content come from two reads of the config | Valid | One snapshot of the eight values feeds both |
| 9 | Unexported types keep exported fields; `token.Mutable` is unused | The unused field is valid; exported fields on unexported types are ordinary Go | `Mutable` removed |
| 10 | Font boundaries untested | Valid | Tests for punctuation in quoted names, 64 and 65 characters, case-folded keywords, a quoted keyword, a trailing comma |
