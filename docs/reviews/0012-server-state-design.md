# Review 0012: the server-state design (D7.1)

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 93k tokens; findings 10–12 were cut from its answer and repeated by Gemini 4 Argon (Medium) in the same session, about 94k more |
| Under review | The first draft of [server-state-design.md](../server-state-design.md) and ADRs 0008–0010, with excerpts of the locked text they must fit (visual-interface.md §2.2, §5.1, §5.4, §7.2, §7.3, §7.5, §10.3; ADR 0005's htmx rule; forms' `Source`; models' `listing`; handlers' `Route` and `Guard`) |
| Scope | Contradictions with the lock; what won't work as written in Go, HTML or htmx 2; security; accessibility; missing cases; API shapes. Decisions S1–S6 were not open |

There were 16 findings. Claude checked each against the draft. Fifteen
changed the text, one of them in part; none could be run, since nothing is
built, so the htmx behaviours they describe are also on the spike's list
(design §12).

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | Request attributes on the table's region are inherited by every link and action inside it: a link in a cell would fetch a page and swap nothing into the table | Valid | The attributes sit only on the controls that change the table's state (§6.8) |
| 2 | The gate admits any link that is no route link, and a wrapper loses `Method` | Valid in part. Admitting a literal is intended: nothing can be asked about it, and the route's guard still decides what is allowed | §4 and §5.3 say so plainly; a wrapper must pass `Method` on; the middleware example handles `Admit`'s error |
| 3 | A request action has no form to submit without script; a form per button would nest inside a page's own form | Valid | One hidden action form in the shells, named by every request control through the `form` attribute (§5.4) |
| 4 | A disabled action: the locked menu skips disabled items with the arrow keys, a locked Button is natively `disabled` and so can't be focused, and an `aria-disabled` submit button still submits without script | Valid | A disabled item renders with nothing that acts (§5.5). Reaching it needs two changes to the lock, put to epistemer as S21 |
| 5 | A table row can't travel out of band in htmx 2 without a `<template>`; `Reply.Replace` can't mark a component it didn't render; and no row renderer or row id existed outside a page | Valid | `Reply` retargets the answer at the id instead, one change per answer (§5.6); `Orders.Row`, `Orders.RowID` and a view declared once with `components.TableOf` (§6.3, §6.6) |
| 6 | htmx 2 doesn't swap a 400, so a filter's error would never show; and the view of an invalid state showed "no rows match" | Valid | 422, which the script lets swap, and `Rejected()`, a view with neither rows nor an empty state (§6.3) |
| 7 | A bulk `Open` action: a POST form's button reaching a GET route, and htmx not including a form on a GET | Valid | The button submits the selection form with `formmethod="get"`; with htmx the selection is included (§6.7) |
| 8 | The search box sat inside its own swap target: typing would be overwritten, and focus lost on every sort and page | Valid | Two parts: the state form, and the results, which alone a search swaps; focus is put back by id (§6.6, §6.8) |
| 9 | A GET form replaces the whole query, wiping a second table's state; ids and `name="id"` collided under a prefix | Valid | The state form carries the query's other parameters; `Prefixed()` uses the table's name; the selection is scoped to its form (§6.4, §6.7) |
| 10 | The action dialog couldn't tell its first load from a successful submission; the 422 branch without script wasn't covered | Valid | It opens when content arrives and closes only on an answer's event; one helper serves both branches (§5.4, §5.6) |
| 11 | `Toast` with `Go` can't combine; `Refresh` has no table URL in a POST handler; `Go` and `Else` were open redirects; which call sends the answer was unclear | Valid | One change, an optional toast and one of two endings; the table reloads from its own URL on an event; `Else` and `Go` take local addresses only (§5.6) |
| 12 | `Amount` needs formatted text the table can't make; `Text` escapes `<mark>`; an image had no alternative text; an address templ refuses isn't empty | Valid | `Amount(func)` and `Image(func)` on the column (§6.2); the table writes marked cells itself (§6.6); a refused address is no address (§4, A5) |
| 13 | Row checkboxes and row menus had no distinct names; the header checkbox had none and was dead without script; the bulk bar came before the checkboxes in the tab order; the status element had no source after a swap | Valid | `RowLabel`, names combined from the control and the row, the bar after the table, the status copied from the results (§6.6, §6.7, §8) |
| 14 | Unlisted cases: a page past the last, time zones and inverted ranges, a bulk failure without htmx, enum tones and numeric alignment, the tiebreak's direction | Valid | A third empty state; a range rule; UTC until neozp carries a zone; the app's flash; numbers aligned and `Cell` for toned badges; the key in the sort's own direction (§6.2–§6.7) |
| 15 | API shapes: `To(nil)` panicking per row; the alias not helping function results; panics promised at a declaration a builder chain can't see the end of; page options on models' column; `tablestest.Bind` with no gate | Valid for four. The alias stays: it is for the gate's function type, and a function literal states `ui.Link` as its result | No address is absent; the key is `New`'s argument, and whole-declaration mistakes panic on first use and in `tablestest.Check`; options move to `tables.Col`, put to epistemer as S22; the test kit sets a gate (§5.1, §6.2, §6.9) |
| 16 | Key names against the design's old example; `q` cut by bytes; the cross-site assumption unstated; ADR 0005's list not amended | Valid but the first: the lock's §11 names keys `ui.<package>.…`, which the draft follows | Runes; §5.4 states what refuses a cross-site request; §10 lists ADR 0005's change |
