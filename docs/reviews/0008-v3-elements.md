# Review 0008: V3, icons and elements

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 223k tokens (it read widely) |
| Under review | The V3 diff (icons, elements, their CSS, the tooltip script), with visual-interface.md §2, §4, §5 and §11 |
| Scope | The API against §4 and §5, immutability, tones, markup and accessibility, the CSS against ADR 0004, the tooltip's Esc |

There were 14 findings. Claude checked each; thirteen changed the code.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | `Image` casts its `SafeURL` to a string, so templ sanitises `data:` and `blob:` URLs away | Valid | The `SafeURL` goes to `src` as it is |
| 2 | The alert's level word should carry its own space | Rejected: §11 says the words end with their punctuation and ui renders a space after them, which the code does. §5.2's wording says the word carries its space; the locked text contradicts itself, and the code follows §11's specific rule. Flagged to epistemer | — |
| 3 | The tooltip's Esc can swallow the Esc meant for an open menu or popover; a dialog's first focus shows the tooltip | Valid | An open popover or modal that doesn't hold the tooltip gets the key; tooltips show on hover and on keyboard focus (`:has(:focus-visible)`), not on a click's or a dialog's first focus |
| 4 | `Merge` doesn't deduplicate or type-check a list present on one side only | Valid | Every joined list goes through the same join, from either side |
| 5 | A gap between trigger and text makes the tooltip impossible to hover (WCAG 1.4.13) | Valid | A bridge under the text; `visibility` transitions with `opacity` |
| 6 | A dismissed tooltip can stay hidden for good | Valid | It comes back when the trigger is entered or focused again after leaving |
| 7 | Neutral outline and ghost buttons show no hover on the page surface | Valid | Their hover tint is the border colour, visible on the page and on cards |
| 8 | A blank avatar name, and a nil or blank `Labelled`, render unnamed images; initials drop combining marks | Valid | Each panics with its package's message; initials keep the marks that follow a letter |
| 9 | Unknown variants, sizes and tones render attributes no CSS selects | Valid (§2.1: mistakes panic) | Validated at render |
| 10 | The tooltip script reads a nested tooltip's text | Valid | `:scope >` |
| 11 | Keys that differ only by case in one map merge in random order | Not taken: a map with both `class` and `CLASS` is a caller's mistake no component makes | — |
| 12 | Blank title, label and tooltip texts | Valid | A blank title is no title; a blank spinner label falls back; a blank button label or tooltip text panics |
| 13 | Whitespace-only `href` and `src`; `Label()` read several times | Valid | Trimmed before the `Empty` check; read once |
| 14 | The card body keeps its children's outer margins | Valid | Trimmed, as the alert's and dialog's bodies do |

Checked again in the browser pane: the tooltip is hidden when idle, shown on
keyboard focus, not on a click's focus, and Esc hides it until it is
focused again. The pane renders no frames while it is hidden, so CSS
transitions stand still there; the checks finished the animations first.
