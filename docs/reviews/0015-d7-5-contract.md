# Review 0015: D7.5, the contract of actions and tables

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (High), through agy (`adversarial_review`), about 71k tokens |
| Under review | The draft server-state-interface.md, against the design's decisions (§9), the spike's findings (§12.1), and the APIs of handlers v0.2.0 (`Link`, `Gate`) and models v0.4.0 (`column`) |
| Scope | Findings and decisions dropped or contradicted; API that can't be built; what an app is missing; conventions; security; accessibility; each question's default |

There were 20 findings. Claude checked each; nineteen changed the draft, in
part or whole.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | `actions.Gate` and `handlers.Gate` don't compile together; a nil gate crashes without the named panic | Rejected for the types: `ui.Link` is an alias (A5), so the function types are identical. Valid for a nil gate | A nil gate panics as a missing one does |
| 2 | `Reply` leaves `HX-Retarget` off `Remove` and `HX-Reswap: none` off `Refresh`, `Fail` and no change, so a dialog's form is wiped | Valid (the spike sent them; the table didn't say) | Every change's headers written out |
| 3 | `Answer` drops `HX-Reselect` | Valid | Both headers |
| 4 | Bulk buttons can't post the selection as written; no way to read it or carry it through a bulk `Open`'s form | Valid | The bulk bar's markup (selection form, `formmethod="get"` for `Open`), `Table.Selected(r)`, `SelectionName()` |
| 5 | Non-local URLs in `hx-*` attributes; `View.URL` from a path like `//host` | Valid | `Request` and `Open` items need a local address; addresses `Bind` builds are local |
| 6 | The size is lost on a search; a search sends unapplied filters | Valid for the size (a spike bug); the search is the state form's, said so | A hidden size input beside the sort's |
| 7 | `Reply` allows two changes, `Go` with a toast, `Fail` without an ending, `Fail(nil)` | Valid | Each panics; `Fail(err).Else(l)` |
| 8 | `Fail`'s fixed 422; `errors.T` without a context; imports; `ErrNotLocal` against "no new errors" | Valid in part: 422 stays, by design, said why | The typo, the import, `ErrNotLocal` called private |
| 9 | `Is` matches two actions built from one base; `Item` hides a field | Valid | Every builder call is a new identity; the field is shown as unexported |
| 10 | No `RowID` for `Remove`; `Row` resolves twice; text keys break ids and selectors | Valid | `Table.RowID`; `TableView.Row(row)` resolves at render; keys escaped in ids |
| 11 | Dropping `Targeted` makes a page of several tables query all of them | Valid | `Targeted` kept |
| 12 | Focus after a removal or a cleared selection has nowhere to go; `ActionButtons` can't key bulk buttons | Valid | A fallback (the scroll region, else the search box); the table renders its bulk bar itself |
| 13 | No row label column leaves `aria-labelledby` dangling | Valid | A table with actions or bulk needs one: `Check` fails, first use panics |
| 14 | Missing declaration panics (a `Dialog` on a POST, `Confirm` on a link, `Primary` with `Danger`, several renderers on a column, `RowLabel` or `Sort` unknown); `Amount` refusing integers; builders after use | Valid | Each panics where it can be seen; a confirmation works on links; `Amount` takes integers; tables seal at first use |
| 15 | `Pagination` can't target a table or stand alone | Valid | `Pager.Target`; a pager without a table isn't offered yet |
| 16 | `InActionDialog`'s "render the form alone" loses the dialog's name | Valid | `ActionTitle` |
| 17 | `historyCacheSize: 0` left to the app | Valid as a question | T11, default: the app's, since ui loads no htmx |
| 18 | `Image` against S8; infinite dates; `<mark>` and byte lengths | Valid | `Image(src func(R) ui.Link, alt)`; infinite values show their text; the rule is per rune |
| 19 | `Attrs` missing on `Pagination` and `ActionButtons` | Valid | Both are specs with `Attrs` (the singletons `ActionForm` and `ActionDialog` stay components, as `ConfirmDialog` is) |
| 20 | `tablestest.Bind` can't refuse links | Valid | It takes `actionstest` options |

On the questions, the reviewer agreed with every default but T6, and
recommended `selected` over `id` for the selection's parameter; the draft's
default is now `selected`.
