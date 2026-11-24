# Review 0026: the build of A9 (C3–C6)

| | |
|---|---|
| Changes | PRs [#52](https://github.com/epistemer/ui/pull/52) (C3), [#53](https://github.com/epistemer/ui/pull/53) (C4), [#54](https://github.com/epistemer/ui/pull/54) (C5), [#55](https://github.com/epistemer/ui/pull/55) (C6) |
| Date | 2026-11-24 |
| Reviewers | C3: Gemini 3.8 Flash (High), about 314k tokens. C4, C5, C6: GPT-OSS 120B (Medium), about 105k tokens in all, because every Gemini tier had run out of quota for two hours after the build (Argon out until about 2026-11-28); never waiting is the rule. Each review saw the diff (generated files left out) and the contract's sections only |

## Who built what

C3 by Gemini 3.8 Flash (High), C4 by Gemini 4 Neon (High), C5 by Gemini 4
Hydrogen (High), each in a worktree outside the repository; C6 by Claude.
Claude reviewed every Gemini diff and checked C4's switches (keyboard,
pointer thirds, label clicks, reset, LTR/RTL, light/dark) and C5's icons
page and RTL arrows in the pane.

## Findings

| PR | # | Finding | Verdict | Outcome |
|---|---|---|---|---|
| C3 | 1 | The pattern test took only an alternation's first branch, so it couldn't see the anchoring bug | Valid | Samples reach every branch and optional part, with junk on either side; without the fix the test now fails for GB, CR and LI (LI had the bug too) |
| C3 | 2 | `HasPostCode` sat in the table of rules | Valid | A sentence below the table |
| C3 | 3 | A closure for an unknown country | Valid | The zero `Rule` |
| C3 | 4 | A string copy to test a buffer for white space | Valid | `bytes.TrimSpace` |
| C3 | 5 | `AssertLabelled`'s test didn't count failures | Valid | Exactly one, naming the field |
| C4 | 1–9 | Listener leaks, `aria-checked` on a native switch, `required` lost, physical CSS, no budget test, unsanitised text | Invalid | Listeners go with an `AbortController`; a native checkbox exposes its state; `required` stays on the two-state input; the CSS is logical; the budget test exists; text goes in with `textContent` |
| C5 | 1–5 | Owned attributes overridable, token joining lost, a count test, CI order, tar checks | Invalid | Owned keys are kept at render (tested); `attrs.Merge` joins tokens (no golden changed); the count test is `>=`; the tar checks are in the generator and its tests |
| C6 | 1–5 | A dropped change, info as fallback, free-form levels, test messages, many messages | Invalid | Each is the contract's own rule (§6) |

Claude's own review added: C4 kept; C5 re-checks the vendored shapes at
generate time, so a hand edit of `shapes.txt` can't reach the rendered SVG
unchecked; CI found a deprecated `tar.TypeRegA` (C5) and an unused helper
and gofmt (C6), all fixed.
