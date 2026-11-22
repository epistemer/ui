# Review 0009: V4, inputs

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 23k tokens |
| Under review | The V4 diff against the spike's field (inputs, their CSS, the script's reveal and `pageshow`), with visual-interface.md §6, §2.2 and the script rows |
| Scope | §6.2's markup and wiring, owned keys and where `Attrs` go, the password reveal, CSS, accessibility |

There were 7 findings. Claude checked each; five changed the code.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | A hidden field drops the control's and the caller's attributes | Valid | Merged as every control's are, with the owned keys kept |
| 2 | A group's error isn't named by its fieldset's `aria-describedby` | Valid in that part. `aria-invalid` on a fieldset (role group) isn't meaningful; groups have no rule attributes of their own | The fieldset names its help and its error |
| 3 | `ui.inputs.field.error` should carry its trailing space | Rejected: §11 says ui renders the space after the level words (the same contradiction with §6.2's wording as review 0008's, flagged to epistemer) | — |
| 4 | A revealed password survives into the back-forward cache | Valid | Passwords are hidden again on `pagehide`, before the page is cached |
| 5 | The reveal toggles a disabled input | Valid | Ignored when the input or the button is disabled |
| 6 | The reveal button's height and disabled state | Valid | It stretches to the row's height, and has a disabled style |
| 7 | The `context` import in the template | Rejected: the group's attribute function uses it | — |
