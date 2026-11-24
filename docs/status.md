# Status log

Newest entry first. Each working session adds one: what landed, decisions
taken for epistemer, what's next, and what's waiting on epistemer.







## 2026-11-24 — C3–C7: A9 built, v0.5.0

**Landed:** C3 (#52: `NavItem` and the gate, `PostCode`/`HasPostCode`,
`AssertLabelled`), C4 (#53: the switches, `Control.TwoState`,
`<ui-tristate>`, the 32 kB script test), C5 (#54: every Lucide 1.77.0
icon generated, `IconSpec`, ADR 0014), C6 (#55: `Reply` and the flash,
`components.Flasher`), and the release v0.5.0. Gemini 3.8 Flash (High),
Gemini 4 Neon (High) and Gemini 4 Hydrogen (High) built C3, C4 and C5;
Claude built C6, reviewed every diff, and checked the switches and icons in
the pane. Review [0026](reviews/0026-a9-build.md).

**Decided for epistemer (clarifications):** flashed toasts render right
after the toast region, not inside it, so they're announced; LI's post code
pattern had GB's bug and is fixed with it.

**Next.** Confab moves to v0.5.0. U2 (serving htmx) waits for the htmx v2
fork discussion. A post-release Gemini review of C4–C6 when its quota is
back (GPT-OSS 120B stood in).

## 2026-11-24 — C2: amendment A9 drafted

**Landed** (Claude): [confab-1-interface.md](confab-1-interface.md), the
exact signatures and markup of U1, U3 and U5–U8, reviewed by Gemini 4
Carbon (High) ([0025](reviews/0025-a9-draft.md), 18 findings, 15 valid
or partly, all taken in), then checked by Claude against Lucide 1.77.0's
files (all 2,410 pass the allowlist; the 22 current icons are unchanged)
and against the GB post code pattern (it accepts junk around a code).
U1 mirrors handlers' flash as epistemer locked it today (handlers A7).

**Locked** by epistemer on 2026-11-24 with the defaults of Q1–Q10 (Q7:
icons as plain Go; Q8: the script's budget is 32 kB gzipped, measured
by a test). Next: C3–C6 build it, then v0.5.0.

## 2026-11-24 — C1: Confab iteration 1, fixes within the terms

**Landed** (Gemini 4 Neon (High) wrote the code in a worktree; Claude
reviewed it, removed a late `htmx.process` call the contract forbids, and
wrote the docs): F23, F24, F27, F6's combobox part, recorded as
clarifications A8; docs for F10, F26 and U4 (F11). The `ui` skill says to
answer 422 whenever htmx may be involved (F17). Checked in the pane: a
combobox that connects before htmx upgrades on `htmx:load`; a boosted
switch from an RTL page to an LTR one sets `dir`, and Back puts it back.

**Review** [0024](reviews/0024-c1.md), Gemini 4 Carbon (High): 9
findings, 7 valid and one partly, all fixed by Claude; the history
handling moved to sessionStorage and htmx 2's real event order.

**Next.** Amendment A9 (U1, U3, U5–U8), drafted for epistemer's lock.

## 2026-11-23 — U5: v0.4.0

**Landed** (Claude): the API checked both ways against
rich-inputs-interface.md (every new identifier is there; `View[R].Err`
recorded as a clarification); CHANGELOG `[v0.4.0]`, the README, the `ui`
skill; the release, checked through the proxy. `theme/themeschema` stays
at v0.2.0.

**Next.** The roadmap after U: nothing planned in the D track; the
backlog's items wait on epistemer or a second user.

## 2026-11-23 — v0.4.0 release review

Review [0023](reviews/0023-v0-4-0.md), Gemini 4 Neon (High) (Argon was
out of quota): 17 findings, 14 valid and one valid the other way round,
all fixed by Gemini 4 Neon (High) in a worktree; the script's fixes
checked in the pane. `Upload` now keeps the last posted file, as `File`
does.

## 2026-11-23 — U4: tables (`Column.Remote`) and the toy shop

**Landed**, to rich-inputs-interface.md §5 and rich-inputs-design.md §3.5, §4.2 and §4.5:
- `tables`: `Column.Remote(search ui.Link, l listing.Labeler)` panics at
  once on a nil link or `Labeler`; `Table.Check` (and so the first
  `Bind`/`Row`) reports `Remote` without `Filter()` or on any kind other
  than `column.Text` or `column.Integer`. The filter form builds a `Text`
  or `Number[int64]` field with `forms`' `Remote(search, l)`, emitting
  `listing.Eq` on the typed value (`string` or `int64`) and owning the
  single parameter `<col>` (or `<table-name>.<col>` when prefixed) in
  `View.Own` / `data-ui-own`, `Clear` and chip `Remove` links.
- Chip resolution and error path: `State` resolves chips once per active
  link filter per request (cached on `State`, and only for `Page` on a
  valid state — `Rejected` never computes chips). Found shows the name,
  miss shows the id, and a `LabelOf` error is recorded in `View.Err`
  (exported on `tables.View[R]`); `components.TableOf.Page` returns
  `v.Err` from `Render` before writing markup, failing the request (500)
  without altering `State.Page`'s locked signature.
- `internal/toyshop`: added `OrderListViewColumns.CustomerID` (`customer_id`,
  `Hidden().Filter().Remote(...)`) to `Orders`, customer and product search
  handlers (`GET /shop/customers/search`, `GET /shop/products/search`,
  fetching `components.SuggestionLimit+1`), stored-file route
  (`GET /shop/files/{key}` serving `receipt-pdf` and `box-png`), and the
  new order page (`GET/POST /shop/orders/new`, wrapped in
  `http.MaxBytesHandler` at 1.5 MiB returning 413 on `*http.MaxBytesError`,
  422 on invalid input preserving keep states and showing the lost note,
  and a success alert on valid submission), linked from the shop nav and
  rendered with `inputs.WithLabelCache` and the shop's gate.
- Tests covering `Column.Remote` panics, `Check`, typed `Eq` filters,
  found/miss/error chips, single `LabelOf` call per request, `TableOf.Page`
  error propagation, and the toy shop's search handlers and order page
  (GET, 422, 200, 413). Updated orders table goldens for the new filter.

**Next.** U5: release v0.4.0.

## 2026-11-23 — U3: the ui script for A7

**Landed**, to rich-inputs-interface.md §6 and rich-inputs-design.md §3.3–§3.4, §4.3–§4.5 and §5.2–§5.3:
- `theme/js/combobox.js`: `<ui-combobox>` upgrades when `data-ui-search`
  and `globalThis.htmx` exist, reusing the server-rendered popup, listbox,
  status, live region and clear button; debounces typing (250 ms) and
  dispatches `ui-search` (at once on Down/Alt+Down), adds `q` in
  `htmx:configRequest` keeping the link's query, aborts in-flight requests
  with `htmx:abort` on commit/restore/blur, verifies status 200,
  `responseURL` origin+pathname and `data-ui-suggestions` in its own
  `htmx:beforeSwap` listener (stopping propagation), numbers option ids and
  announces after swap, and handles the keyboard table and form reset;
- `theme/js/repeat.js`: `<ui-repeat>` rewrites the detached clone's own
  path prefix in `name`, `data-ui-path` and `data-ui-when` at every depth
  (including nested `<template>.content`), suffixes this level's ids and
  rewrites references token by token at every depth, appends and runs
  `globalThis.htmx?.process(item)`, evaluates conditions before focusing,
  renumbers legends and Remove buttons, and marks bounds with
  `aria-disabled` and a visually hidden reason;
- `theme/js/upload.js`: keep cards over checkboxes, chosen list with Remove
  via `DataTransfer` and opt-in `data-ui-previews` object-URL thumbnails,
  size and case-insensitive `accept` checks (untyped files deferring to the
  server) with `setCustomValidity` and a capturing `submit` guard, drop
  highlight, upload progress from `xhr.upload` in `htmx:beforeSend`, and
  413 handling on upload fields with files;
- `theme/js/actions.js`: `isAction` ignores `ui-combobox`, and the
  `htmx:beforeRequest` listener that drops `pending` ignores
  `e.defaultPrevented`;
- `.ui-combobox-clear` in `theme/css/components.css`, with the clear button
  rendered server-side `hidden` in `inputs/field.templ` so `TestStories`'s
  reverse selector check sees it while script-off markup keeps it hidden;
- `GET /demo/customers/search` (and `/demo/products/search`) in
  `stories.Demo`; `internal/spike` removed; script tests in
  `internal/jstest` for single-pass `fmt`, path/condition rewriting, size
  formatting and `accept` matching.

**Found in the pane** (Claude): in htmx 2.0.11 the swap events'
`detail.elt` is the swap target (the listbox), not the element that asked,
so the first version never ran its after-swap step; the requester is
`detail.requestConfig.elt` (fixed, with a comment). Also: htmx reads
`hx-get` when it processes an element, so a test that changes it must
call `htmx.process` again.

**Checked in the pane** (catalogue frames, with htmx): search, ids, count
and announcement; Down and Enter choosing with `change` on the hidden
input; typing then Esc within the debounce leaves the popup closed; more
than 20 results; a 200 page that isn't an answer and a 404 both say the
search failed, with the link's query kept; a nested list's clone with its
remote select searching (`htmx.process`), unique ids, no broken
references; upload keep cards, the size check refusing the submit, its
error cleared; real keys in RTL dark (type, Down, Enter, Tab).

Gemini 4 Argon (Medium) wrote it in a worktree, about 338k tokens; Claude
fixed the swap-event check, ran the pane checks and every Go check.

**Next.** U4: tables (`Column.Remote`) and the toy shop.

## 2026-11-23 — U2: inputs and components for A7

**Landed**, to rich-inputs-interface.md §3, §4 and §7:
- `inputs.Field` renders the remote select (with its htmx attributes,
  popup and live region), the upload shape for all four file kinds, and
  lists with `<ui-repeat>` and their template row;
- `inputs.WithLabelCache`, `FieldSpec.Previews`; the gate asked for the
  search link and stored files' links;
- `components.Suggestion`, `SuggestionLimit`, `Suggestions`;
- the keys of §7; the CSS moved from the spike into components.css;
  stories and goldens for a remote select, an upload, a list and a
  nested list.

Choices where the contract is silent: the server-rendered listbox is
named by the field's label; the clear button's CSS waits for U3, whose
script creates it; `tooLarge` is filled with `MaxSize`'s value formatted
as rule formats sizes.

Gemini 4 Argon (Medium) wrote it in a worktree, about 313k tokens;
Claude reviewed the goldens and ran every check.

**Next.** U3: the script.

## 2026-11-23 — U1: forms for the remote select, uploads and lists

**Landed**, to rich-inputs-interface.md §2:
- `ui/forms/files` (`Stored`, `Upload`); `Form.Upload`/`Uploads` with the
  stored files, their binding, presence and panics;
- `Field[T].Remote` and `KindRemote`, `KindUpload`, `KindUploads`;
  `Control.Remote`, `Stored`, `Lost`, `Bounds`; `inputmode` for an
  integer remote field;
- a new list's control yields `MinItems` blank items; `List`'s doc
  comment says which rows bind drops (L7);
- `rule.Files`, `MinFiles`, `MaxFiles`, `ErrMinFiles`, `ErrMaxFiles`;
  `MaxSize` gives `data-ui-max-size`;
- an oversized body's `*http.MaxBytesError` is found through `f.Err()`.

One choice the contract left open: `Form.List` builds its template row at
declaration, so `Upload` in a list panics there (I10); a build function
runs once at declaration and mustn't read the list's own handle
(CHANGELOG).

Gemini 4 Argon (Medium) wrote it in a worktree from a brief naming
contract §2, about 323k tokens; Claude reviewed the diff and ran build,
vet, race tests, tidy and staticcheck.

**Next.** U2: inputs and components.

## 2026-11-22 — A7 locked

**Locked** by epistemer: [rich-inputs-interface.md](rich-inputs-interface.md),
amendment A7, with the answers recorded before and every other default
(R3, R5, R6, R8, R10–R14, R17–R22, R25, R26, R28; L2–L6). handlers' A5
accepted with H5 and H7–H9 on their defaults.

Recorded in interface.md §10, visual-interface.md §13.6 and
server-state-interface.md; ADRs 0012 and 0013 accepted; the roadmap's U
milestones planned (U1 forms, U2 inputs and components, U3 the script,
U4 tables and the toy shop, U5 v0.4.0).

**Next.** handlers A5 built and released as v0.3.0, then U1.

## 2026-11-22 — D8: epistemer's first answers

**Decided** by epistemer:
- R1: a remote select is declared on the forms field (`Remote`);
- R2 and R9: only `inputs` calls the `Labeler`, at render; whether an id
  may be picked is the app's check;
- R4: the search goes through htmx, not the script's own `fetch` (the
  draft's default, which the spike tried). The design (§3.3), ADR 0011 and
  the contract are rewritten for it: `hx-get` on `<ui-combobox>`,
  triggered by the script's `ui-search` event, stale requests vetoed by
  htmx's events, ids given after the swap, no upgrade without htmx,
  `htmx.process` on cloned rows. Not spiked in that form;
- R7: a refused search link leaves the text input of the id;
- R15 and R16: `Upload`/`Uploads` take the stored files at declaration;
- R23: "Item %d" for every list; R24: adding and removing rows needs
  script; R27: a new list's control yields `MinItems` blank items;
- L1: the upload types live in `ui/forms/files`;
- L7: no item marker; forms §5.8 stays as locked, and a row that could
  post nothing uses `Bool` or a field that always posts;
- H6 (handlers): a route's limit replaces the registration's.

**Next.** The lock of A7 and acceptance of handlers' A5, with the rest
on their defaults.

**Waiting on epistemer**: the remaining R, L and H questions, or the
lock with their defaults.

## 2026-11-22 — D8.3 and D8.4 drafted: the handlers amendment and the contract

**Landed** (Claude), docs only, plus the spike:
- Three facts the spike hadn't settled, now in the design:
  - with a route capped below two files, a 413 that htmx doesn't swap
    showed nothing, so the spike's script now puts a message on the
    form's upload fields and keeps the files (§4.5);
  - the `LabelOf` memo needs an owner: an opt-in `inputs.WithLabelCache`
    (R10);
  - a remote field with no gate panics, as actions do.
- The record: D8.2's row and §8.1 say the spike has no goldens and
  checked the shape without script in markup only.
- D8.3: handlers' [ADR 0006](https://github.com/epistemer/handlers/pull/18),
  `//h:body` and `x-max-body`, questions H5–H9.
- D8.4: [rich-inputs-interface.md](rich-inputs-interface.md), amendment
  A7, questions L1–L7; `StoredFile` and `Upload` move to a leaf package
  `ui/forms/files` (L1), since `rule` can't import forms.
- Review [0021](reviews/0021-d8-4-contract.md) (Gemini 4 Argon High,
  with handlers' 0007): 18 findings, 17 valid and one in part, all folded in.

**Next.** On epistemer's answers: handlers A5 built and released, then
the build milestones of A7.

**Waiting on epistemer** (API design): R1–R28 (design §9), L1–L7
(contract §8), H5–H9 (handlers' ADR 0006).

## 2026-11-22 — D8.2: the slice spike of the remote select, uploads and list fields

**Landed** (Claude):
- `internal/spike` again, mounted by the catalogue at `/spike/order` and
  `/spike/table`: the toy order form with a customer remote select, a
  receipt upload with two stored files, and lines (a list holding a
  remote select, a file and a nested list); a filter form inside a
  `<ui-table>`; a remote select in the action dialog. The design's forms
  shapes are declared locally, keyed by path; nothing in forms or inputs
  changed. Go tests check ids, names, selectors, the `Labeler` rules and
  the keep keys.
- Checked in the pane by script, and with real keys for Esc, Enter and
  Tab, in LTR and RTL, light and dark. Seven findings amended the design
  (§8.1): R5's suggestion becomes a ui struct (models' option label is a
  key), `Control.Bounds()`, an Add button named for its list, count-free
  bound texts, announcements through a timer, htmx's progress events
  telling upload from download only by order, and a remote filter owned
  by its table.
- Review [0020](reviews/0020-d8-2-spike.md).

**Next.** D8.3, the handlers amendment (a per-route body limit), then
D8.4, the contract.

**Open for epistemer** (API design): R1–R28, unchanged in number; R5's
default changed with the spike.

## 2026-11-22 — D8.1: the design of the remote select, uploads and list fields

**Landed** (Claude), docs only:
- [rich-inputs-design.md](rich-inputs-design.md), with ADRs 0011 (remote
  select), 0012 (uploads) and 0013 (list fields). zp was read for needs
  only (§1, counts and shapes).
- Questions R1–R28, each with a default and an alternative (§9). The
  defaults that reverse accepted text, each named in its question:
  - R1 and R16 put the search link, the `Labeler` and the stored files on
    the forms field, against forms-design §10.3, because `inputs.Field`
    renders groups, list rows and table filters generically;
  - R4 has the remote select fetch with the ui script instead of htmx,
    which takes it off ADR 0005's `hx-*` list.
- Review [0019](reviews/0019-rich-inputs-design.md): Gemini 4 Argon (High),
  about 69k tokens, 18 findings, all valid, all folded in.
- The roadmap splits D8 into D8.1–D8.4. The neozp project board caught up:
  D7.2, the actions and tables milestones and T4 marked Done, and D8 In
  progress.

**Next.** D8.2, the slice spike (design §8). The questions stand on
their defaults until epistemer answers or the lock (D8.4) takes them.

**Open for epistemer** (API design, not blocking the spike): R1–R28,
above all R1, R4, R15 and R16.

## 2026-11-22 — T4: v0.3.0

**Landed** (Claude):
- The API review. Every exported identifier of `actions`, `tables`, their
  test kits, `components`, `elements` and `layout` is checked both ways
  against server-state-interface.md and visual-interface.md. One gap was
  found: `tables.ErrRange`, now named in the contract's §8 as a
  clarification. `Unavailable` and `Menu.ID` are in the contract's §5 as
  prose.
- CHANGELOG `[v0.3.0]`, the README, the `ui` skill.
- The release: ui `v0.3.0`, checked through the proxy. `theme/themeschema`
  is unchanged at v0.2.0.

**Next.** D8, the second round: the remote select, uploads and list
fields (design §13).

## 2026-11-22 — T3: the table's rendering

**Landed** (Claude), to server-state-interface.md §4.1, §6 and §7:
- `TableOf` and `Pagination` in `ui/components`:
  - the state form, results, grid, cells by kind, `<mark>`ed hits, chips,
    empty states;
  - the bulk bar of keyed submit buttons of the selection form;
  - row menus keyed `<table>-row-<key>-actions`, named with the row's
    label.
- The table's CSS in components.css, the spike's draft under ADR 0004's
  checks. The filter row is aligned (finding 17).
- `theme/js/table.js`: `<ui-table>`, the live-address merge, the refresh.
- `internal/toyshop`, the spike's shop on the real packages, mounted by
  `cmd/catalog` at /shop/orders, with goldens, checks and every answer
  tested.
- Stories `components/table` and `components/pagination`.
- `internal/spike` removed.

**Review** ([0018](reviews/0018-t3-table-view.md)): Gemini 4 Argon
(Medium), about 36k tokens. Nine of 13 findings fixed:
- status and focus after a whole-table swap;
- only state requests mark the results busy;
- keyed ids on chips, clear, size, empty and Apply links;
- a status region that is never hidden;
- `marked` checking the query and invalid UTF-8.

**Checked in the pane**, at /shop/orders:
- two tables keeping each other's state through a filter, a search and a
  sort;
- the size kept by a search;
- a rejected search taking the whole table;
- a row's refund replacing its row, the trigger named with the row;
- Reassign's focus coming back to the keyed trigger after the refresh;
- a bulk Mark shipped, with focus falling back to the scroll region once
  the selection cleared.

## 2026-11-22 — T2: tables

**Landed** (Claude), to server-state-interface.md §3:
- `ui/tables` on models v0.4.0's `column`:
  - declaration panics, and a table sealed at its first `Bind` or `Row`,
    which also panics on what `Check` reports;
  - ids escaped from text keys;
  - the selection under `selected` (T6), read back by `Selected`;
  - `Answer`'s retarget and reselect, and `Targeted`;
  - a hidden size beside the sort;
  - chips that format their values: an option's label, a bool's word, a
    date as `Date` renders it;
  - every address local, and the pager named after its table.
- `tablestest`, with `actionstest`'s options.
- Two clarifications of the locked text, in the contract:
  - `selected` is a reserved filter name;
  - ids escape `_` too, so an escape is never ambiguous.

**Review** ([0017](reviews/0017-t2-tables.md)): Gemini 4 Argon (Medium),
about 35k tokens. Ten findings, all fixed:
- an empty text key;
- a capped page;
- chips from bound values;
- a query normalised once, so junk never travels on in links;
- a cloned slice of columns, and sealing before checking;
- `View.Selection` removed, since it isn't in the locked text.

## 2026-11-22 — T1: actions

**Landed** (Claude), to server-state-interface.md §2, §4.2–§4.4 and §5:
- `ui/actions` and `actionstest`. Every builder call is a new identity
  for `Is`. Panics:
  - a blank label or question, `Dialog` with `Confirm`, `Primary` with
    `Danger`;
  - no gate, or a nil one;
  - a `Dialog` action bound to a link that isn't a GET.

  A `Request` or `Open` item with an address that isn't local is absent.
- `ButtonSpec.Unavailable` (in a Tooltip, inert: its href, form, request
  and confirmation attributes dropped). `MenuItemSpec.Unavailable` (named
  by its label, described by its reason). `MenuSpec.ID`.
- `ActionButtons`, `ActionMenu`, `ActionForm`, `ActionDialog`,
  `ActionTitle`, `InActionDialog` and `Reply`, with every change's
  headers. Panics: two changes, `Fail(nil)`, `Go` after a change or a
  toast.
- The shells render the action form and dialog.
- `theme/js/actions.js`: the 422, the dialog, focus back to its opener.
  `menu.js`: arrow keys stop on unavailable items.
- Stories `components/actions` (buttons, menu, in a card, htmx), with demo
  endpoints, `stories.Demo`, which `cmd/catalog` mounts.

**Review** ([0016](reviews/0016-t1-actions.md)): Gemini 4 Argon (Medium),
about 64k tokens. Four valid findings fixed:
- unavailable controls drop htmx's `data-hx-*` aliases and commands;
- focus is remembered for every action's request, not only the dialog's;
- a stale opener is forgotten;
- the dialog is emptied before focus moves.

Three findings were rejected against the goldens and the pane.

**Checked in the pane** (LTR light and RTL dark):
- the Open action: the dialog named by its title, a 422 kept open, then
  success with the dialog emptied, a toast, and focus back;
- a confirmed request's toast, and `Fail`'s danger toast;
- arrow keys stopping on an unavailable menu item, which is named by its
  label, described by its reason, and inert.

## 2026-11-22 — D7.3, D7.4 released; D7.5 drafted

**Released** (Claude), as epistemer accepted them with their defaults:
- models v0.4.0, column sets for views (its amendment A4, ADR 0007,
  review 0013);
- handlers v0.2.0 with `authguard`, `openapi` and `cmd` v0.2.0: builders
  return `handlers.Link`, and `handlers.Gate` (its A4, ADR 0005,
  review 0006).

Both were checked through the proxy, and both skills are updated.

**Drafted** for epistemer's lock:
[server-state-interface.md](server-state-interface.md), the contract of
`ui/actions`, `ui/tables`, their test kits, and what components,
elements, layout, the CSS and the script gain. It is built on the spike's
findings and the two releases.

**Review** ([0015](reviews/0015-d7-5-contract.md)): Gemini 4 Argon
(High), about 71k tokens. 20 findings; 19 changed the draft:
- `Reply`'s headers for every change, and the combinations it refuses;
- `Answer`'s reselect;
- the bulk bar's markup, `Selected` and `RowID`;
- local addresses for `Request` and `Open` items;
- the size kept on a search;
- ids escaped from text keys;
- a focus fallback;
- the declaration panics.

The reviewer recommended `selected` for the selection's parameter (T6),
now the default.

**Locked** by epistemer the same day, with the defaults of T1–T11:
amendment A6 (interface.md §10, visual-interface.md §13.5). D7 is done; the
T milestones are planned in the roadmap.

## 2026-11-22 — D7.2: the slice spike of links, actions and tables

**Landed** (Claude), in `internal/spike`:
- the column shape models would generate (`col`);
- `actions` and `actionstest`, `tables` and `tablestest`;
- `tableui`: what `ui/components` would gain (`TableOf`, `Pagination`,
  `ActionButtons`, `ActionMenu`, `ActionForm`, `ActionDialog`,
  `InActionDialog`, `Reply`);
- a stylesheet and script of its own;
- the toy shop: orders and toys in memory, their handlers, `cmd/shop`.

Run it with `go run ./internal/spike/cmd/shop` (localhost:7332).

**Checked in the browser pane**, with htmx and without (`/plain`), every
item of design §12:
- search, filters, sort, paging and sizes;
- the 422, Back, two tables;
- row actions in a menu and a card: a link, a confirmed request, an
  `Open` action failing once then succeeding, one the gate refuses, one
  disabled by a state rule and still reachable;
- two bulk actions, one of them `Open`;
- `Reply`'s replace, remove, refresh, toast and close.

**Tests:**
- goldens, `CheckIDs`, `CheckNames` and `CheckSelectors` on seven table
  states and an order's card;
- both test kits;
- every answer over HTTP, with and without htmx.

The spike ran in its own server, not the catalogue as D7.2's row said,
because a table needs handlers and an address (finding 14).

**Findings** ([server-state-design.md](server-state-design.md) §12.1):
20 of them. Those that reach D7.5's contract:
- the live address merged into a table's state requests;
- `HX-Trigger-After-Settle` for `Reply`;
- keyed ids for action triggers and bulk buttons, which asks `Menu` for
  an id;
- `Reply.Fail`;
- the action dialog's own markup and name;
- S21's shapes, and a menu item named by its label alone;
- pagers named after their tables;
- the selection's parameter name;
- enum filters as strings;
- a number renderer.

**Review** ([0014](reviews/0014-d7-2-spike.md)): Gemini 4 Argon (High),
about 246k tokens. 17 findings:
- twelve changed the spike or a fix, among them the redirect check, a
  rejected search taking the whole table, focus after a refresh, and the
  toast fix's second modal;
- four were rejected, after checking the pane and htmx's behaviour;
- three are new findings, 18–20.

**Fixed on the way**, outside the spike:
- forms' `FromRequest` read a spent body as a blank form;
- `toast.js` lost a toast that arrived as its dialog closed.

**Next.** D7.3 and D7.4 are drafted and wait for epistemer:
- handlers' [ADR 0005](https://github.com/epistemer/handlers/pull/15),
  with questions H1–H4;
- models' [ADR 0007](https://github.com/epistemer/models/pull/14), with
  questions M1–M5.

D7.5, the contract of actions and tables, follows their answers.

## 2026-11-22 — V7: v0.2.0

**Landed** (Claude):
- An API review of every exported identifier, checked both ways against
  the contracts:
  - Every function, type and method in theme, icons, elements, inputs,
    components, layout, catalog and catalogtest is in visual-interface.md.
  - Every function the contract names is in the code. The one exception,
    `Group`, is themeschema's, in its own module.
  - Run again after A5: the kernel's `Link` and `URL` are in interface.md
    §4A.
- CHANGELOG `[v0.2.0]`, with A5; a README section on building a page; the
  `ui` skill covers the visual packages and links.
- Released, in the roadmap's order:
  - ui `v0.2.0` was tagged on #28's merge;
  - #31 pointed `theme/themeschema` at it and dropped its `replace`;
  - `theme/themeschema/v0.2.0` was tagged on #31's merge.
- Checked through proxy.golang.org from a new module: `go get` of both
  works, a Button renders with a `ui.URL` link, and themeschema pulls ui
  v0.2.0.
- Merged on the way: #29 (D7.1's design, from another session) and #30
  (A5), both before the tag.

**Defaults taken** (epistemer's rule: an unanswered question takes its
default):
- §5.2 and §6.2 said the level words carry their own space, while §11 and
  the code render the space. The wording now follows §11. No behaviour
  changes.
- V15, forms' `Placeholder`: no change.
- D7.1's S7–S20 and S22 stand on their defaults.

**Next.** D7.2: the slice spike of the orders table (server-state-design
§12).

## 2026-11-22 — A5: links

**Landed** (Claude), to interface.md §4A and visual-interface.md §2.7 and
§13.4:
- `ui.Link` and `ui.URL` in the kernel, which still imports only templ and
  the standard library.
- `internal/links.Href` resolves a link: through `templ.URL`, and `""` for
  no address (nil, empty, or refused).
- The eight signatures take a `ui.Link`. Every golden is unchanged.
- Callers pass `ui.URL("…")` for literal addresses: stories, tests,
  examples, doc comments.
- No address keeps what an empty `templ.SafeURL` did before, with tests
  for each case:
  - Button and MenuItem stay buttons;
  - Link and Image render `Empty`;
  - Avatar shows the initials;
  - the last Crumb is the current page as text;
  - NavItem panics, as it did on an empty href.
- One addition the contract doesn't name: `Tab` with no address now panics,
  as `NavItem` does, under §2's rule that programmer mistakes panic at
  render. Before, it rendered `href=""`.

**Review** ([0013](reviews/0013-a5-links.md)): Gemini 3.8 Flash (High),
about 207k tokens. Five findings; four changed the code: typed nil
pointers are nil links, crumbs and value links resolve once, and more
no-address tests. Rejected: keeping `data:` and `blob:` images, which A5
gives up on purpose (ADR 0008); review 0008's allowance is superseded.

**Next.** V7: the release, on top of this.

## 2026-11-22 — D7.1: the design of links, actions and tables

A design session with epistemer: a reading of zp's server table and
actions, a brainstorm, then the draft. No code.

**Landed** (Claude):
- [server-state-design.md](server-state-design.md): what the record of zp
  asks for, links, `ui/actions`, `ui/tables`, filters as a form, the
  pager, selection and bulk actions, transport, accessibility, the two
  test kits, and the spike's list.
- ADRs [0008](adr/0008-links.md) (an address is a link),
  [0009](adr/0009-actions.md) (actions are a logic layer, gated by their
  route) and [0010](adr/0010-tables.md) (generated columns, state in the
  URL, one response).
- The roadmap: D7 split into D7.1–D7.5, D8 for the second round, A5
  before V7, and the T milestones in outline.
- The inventory's "keep: rows streamed over SSE" is withdrawn (ADR 0010).

**Decided by epistemer** (design §9): the permission comes from an
action's route (S1); models generates each view's columns (S2), as a kind
and plain values (S3); one handler and one response, no SSE for rows
(S4); handlers' builders return a link and every href in ui takes
`interface{ URL() string }` (S5); this round is tables and actions, and
v0.2.0 is tagged after it is reviewed (S6).

**Review** ([0012](reviews/0012-server-state-design.md)): Gemini 4 Argon
(High), about 93k tokens, and about 94k more on Argon (Medium) to repeat
three findings its answer had cut. Sixteen findings; fifteen changed the
draft. The largest: request attributes inherit, so they sit on the
table's controls and never on its region; a request action without
script needs a form, so the shells get one; a search box must not sit in
its own swap target; an answer retargets itself at a row instead of
sending rows out of band.

**Decisions taken for epistemer, on defaults** (S7–S22, design §9). The
ones most worth a look:
- S7: ui doesn't import handlers. It owns `Link` and the gate's type.
- S8: `Src` and `Image` take a link too.
- S22: a column's page options sit on `tables.Col`, not on models'
  column. This differs from the brainstorm's sketch
  (`c.Email.Searchable()`); the review argued it and Claude agrees.
  Either way the table a page declares decides which columns show and
  which are searched, sorted or filtered, which is what epistemer asked
  for; S22 is only which package's type carries the options.

**S21, decided by epistemer** later the same day: a disabled action must
be reachable to say why. At D7.5, after the spike, the lock gains a
second Button state that is `aria-disabled` and focusable (`Disabled`
stays native), and menus stop on disabled items.

**Amendment A5 is locked** (epistemer, the same day): the kernel's
`Link` and `URL`, and eight signatures of the visual packages, with
`Src` and `Image` in on S8's default. It is written into interface.md
(§4A, §10) and visual-interface.md (§2.7, §13.4). The code still takes
`templ.SafeURL` until the roadmap's item A5 is built, so **V7 waits for
it**; release PR #28 is open, and its API review will not match the
contract before then.

**Waiting on epistemer:** S7, S9–S20 and S22, or their defaults.

**Next**, in a coding session: build A5; then the slice spike
(D7.2), whose findings amend the design; then the handlers and models
amendments (D7.3, D7.4) and the contract (D7.5).

## 2026-11-22 — V6: ui/layout

**Landed** (Claude):
- `ui/layout` per §8:
  - Document, with lang and dir from `i18n.Code`, and `ColorSchemeOf`
    reading the cookie;
  - App, with the drawer toggle, NavItem, NavSection and ColorSchemeMenu
    (menuitemradio items under a Sun/Moon trigger);
  - Auth;
  - PageHeader: breadcrumbs, and actions only when they write something.
- `layout.css`: breakpoint 60rem; sizes in `em`, because the checker
  rejects `rem` outside media queries; a 3.5em bar that the drawer, the
  sticky column and the scroll padding start below.
- `layout.js`: the drawer, and the colour scheme with its cookie.
- Stories: `layout/app` (shell, dark-picked, no-nav, toast), `layout/auth`
  (sign-in, with errors), and `layout/page-header`. The app shell is the
  roadmap's full page of the toy shop.
- Kept from §8.2's list without fixing an order: the toggle sits before
  the brand, at the drawer's edge. The app renders the toggle and the nav
  only when `AppNav` is filled, and the brand and bar-end only when
  theirs are.

**Review** ([0011](reviews/0011-v6-layout.md)): Gemini 4 Argon (Medium),
about 63k tokens. Nine findings; eight changed the code.

**Checked in the browser pane**, at narrow widths through a 500px iframe,
because viewport emulation doesn't reach a hidden pane:
- the drawer: from the start edge in RTL, below the bar, focus on the
  first item; Esc returns focus to the toggle; an outside click, or a
  plain click on an item, closes it; a ctrl-click doesn't;
- Esc goes to an open menu before the drawer;
- the scheme menu: `<html>`, the cookie, `aria-checked`, the trigger's
  icon, and the sync on `pageshow`.

Not checked: closing the drawer when the screen widens. A hidden pane
fires neither `resize` nor `matchMedia` change events. The layout itself
is right either way, since the wide grid is CSS alone.

**Merged:** #25 (V4) and #26 (V5), after green CI and review 0010.

**Next.** V7: the v0.2.0 release.

## 2026-11-22 — V5: components

**Landed** (Claude):
- `ui/components`, moved from the spike and cut to §7:
  - Dialog: an outline default trigger, a required body, scroll
    containment, and an `Attrs` that keeps `id` and `aria-labelledby`;
  - the confirmation, with `autofocus` on Cancel;
  - Menu: `Attrs` on `<ui-menu>`, and `MenuItem.Submit(form)`;
  - navigation tabs, toasts and the detail list.
- The anchored menu CSS sits behind `@supports` and can be switched off
  with `<html data-ui-no-anchor>`. The catalogue's `anchor=off` uses that
  to test the script's placement.
- htmx 2.0.11 (Zero-Clause BSD) is vendored into the catalogue's assets
  for its stories only. `cmd/catalog` answers two demo requests with
  `ToastSwap`.
- The exception for `ToastSwap`'s `hx-swap-oob` (V18) is written into
  ADR 0005.

**Checked in the browser pane:**
- The htmx confirmation: a click asks before any request, then sends one;
  a select's change asks through `htmx:confirm`. Both answer with a toast
  out of band.
- Toasts: the timeout, closing, and focus moving to the next toast.
- The menu's own placement, running in a browser for the first time: RTL
  start edge, flipping above near the bottom, clamping in the viewport.
- The detail list's container query.

**What the check §7.5 asked for found:** a toast shown above an open modal
dialog is visible but inert, because the modal makes the rest of the page
inert, a later popover included. As §7.5 allowed, toasts now go into a
region inside the open dialog and move back when it closes. The script
notices the close through the dialog's `open` attribute (a
MutationObserver), not its `close` event, which waits for rendering
(finding 13).

The confirmation's click listener now skips fields that take input, so a
select that confirms its change isn't asked when clicked.

**Review** ([0010](reviews/0010-v5-components.md)): Gemini 4 Argon
(Medium), about 40k tokens. Twelve findings; eight changed the code, mostly
the toasts around modals (toasts already shown move into a modal that
opens, live regions are shown before they're filled, no pause flag), the
menu's own placement (clean measurement, inline flip, one per frame, the
gap from `--ui-space-1`), blank names, and checkboxes in the confirmation.
Checked again in the pane: toasts move into the confirmation and back and
still time out; the menu flips near either edge in RTL.

**Next.** V6: `ui/layout`, and `internal/spike` removed (already empty).
## 2026-11-22 — V4: inputs

**Landed** (Claude):
- `ui/inputs`, moved from the spike and cut to §6:
  - keys renamed to `ui.inputs.field.*` and unexported;
  - help above the control;
  - errors led by the `CircleX` icon and a hidden "Error:";
  - `ui-field-row` for checkboxes and passwords;
  - the password reveal, with `data-ui-reveal` in the script, hidden until
    `data-ui-ready`;
  - groups carry `aria-describedby`, help and error on their fieldset, and
    take `Attrs`.
- Conditions refresh on `pageshow`.
- Sign-in stories: a password, its errors, and a checkbox's help.

Checked in the browser pane: the reveal switches the input and
`aria-pressed`, swaps its icons, and a form reset puts both back.

The spike now holds only the components (V5).

**Review 0009** (Gemini 4 Argon (Medium), about 23k tokens): 7 findings.
Five changed the code:
- a hidden field takes `Attrs`;
- a group's error is described by its fieldset;
- passwords are hidden on `pagehide`;
- the reveal ignores a disabled input;
- the reveal button's styles.

Two were rejected: the space after "Error:" is the §6.2 versus §11 question
already flagged, and the `context` import is used.

**Next.** V5: `ui/components`.
## 2026-11-22 — V3: icons and elements

**Landed** (Claude):
- `ui/icons`: 22 icons, Lucide's paths with its licence (ADR 0007), and
  `Labelled`.
- `ui/elements`, built to §5:
  - Button moved from the spike, with `Label` and tones resolved against
    each component's default, so `Neutral` renders `data-tone="neutral"`
    and `Brand` on a button renders nothing;
  - Badge, Alert (four message tones, with an icon and a hidden level
    word; any other tone panics), Card with three slots, Avatar (initials
    or a picture, decorative or named), Spinner, Tooltip (a required
    trigger, Esc through the script), separators, Heading;
  - the value renderers, which fall back to `Empty`;
  - `Merge`, which now also joins `aria-controls`.
- Stories for every element: 26 goldens pass the ids, names and selector
  checks, and the reverse check on ui's CSS. The reverse check caught one
  dead rule (`.ui-button[data-tone="brand"]`).

Checked in the browser pane:
- the tooltip shows on focus, is named by `aria-describedby`, hides on Esc
  and comes back after focus leaves;
- `data-ui-ready` is set;
- the alerts are right in RTL and dark mode.

The spike's own elements and icons are gone. The spike's inputs and
components use ui's now.

**Review 0008** (Gemini 4 Argon (Medium), about 223k tokens): 14 findings.
Twelve changed the code:
- `data:` URLs in `Image`;
- tooltip Esc against open menus, and tooltips shown on keyboard focus
  only;
- a hoverable tooltip;
- `Merge` deduplicating one-sided lists;
- visible neutral hovers;
- validation of variants, sizes and tones;
- blank texts;
- initials with combining marks.

One was declined (keys differing by case in one map). One was rejected
against §11, which points to a contradiction in the locked text, below.

**For epistemer (locked text):** §5.2 says the alert's hidden level word
"carries its own punctuation and space". §11 says the words end with their
punctuation and that ui renders the space after them. The code follows §11,
so translators write "Erreur :" without a trailing space. The default is to
correct §5.2's wording to match.

**Next.** V4: `ui/inputs`.
## 2026-11-22 — V2: the catalogue and the checks

**Landed** (Claude):
- `ui/catalog`, moved from the spike and cut to §9:
  - `New` panics on bad names, duplicates and empty stories.
  - The theme is the app's to mount.
  - URLs are relative, and the index writes its relative Location itself,
    because `http.Redirect` would make it absolute from the path
    `StripPrefix` left.
- `ui/catalog/catalogtest`.
- uitest's `CheckNames` and `CheckSelectors` (A4), on `ui/internal/a11y`,
  which now takes class prefixes.
- ui's stories moved to `internal/stories` and run through
  `catalogtest.Check`, with the reverse check on ui's CSS.
- `cmd/catalog`.

Checked in the browser pane: relative links, 20 frames, and the token panel
with themeschema's key names.

**Clarification of §1 (not API):** the catalogue's token panel reads the
derived tokens through `ui/internal/themehook`, as catalogtest reads the
stylesheet. Review 0006 noted that §1's import table names the hook only
for catalogtest.

**Review 0007** (Gemini 4 Argon (Medium), about 42k tokens): 13 findings.
Twelve changed the code:
- `title` counts as a name;
- uitest's `Prefix` replaces the default;
- golden-safe variant names;
- the selector parser handles comments in strings, `:is`, `:where` and
  attribute operators;
- a trailing-slash mount works;
- pages are buffered;
- catalogtest has tests of its own.

**Next.** V3: `ui/icons` (Lucide, with an ADR) and `ui/elements`.
## 2026-11-22 — V1: `ui/theme`

**Landed** (Claude):
- `ui/theme`, moved from the spike and cut to visual-interface.md §3.
  - Everything but §3's identifiers is unexported: the token table, the
    colour maths and the derivation.
  - `Config`'s getters carry the `Theme` prefix.
  - The checks return declared errors, and a font check refuses CSS-wide
    keywords.
- `theme/themeschema`, a nested module, so cfgx stays out of ui's go.mod.
  - A toy shop's generated settings satisfy `theme.Config`, load the token
    table's defaults, and refuse a bad colour, font list or radius.
  - It builds against the repository's ui through a `replace` until
    v0.2.0 is tagged.
- `ui/internal/themehook` hands the joined stylesheet and the derived
  tokens to ui's own packages. The catalogue's token panel now asks it,
  with themeschema's key names. It is internal: the module table in
  visual-interface.md §1 names it for catalogtest, and the catalogue's use
  is the same kind.
- The spike now uses `ui/theme`, so there's one copy. jstest reads
  `theme/js/cond.js`, and CI runs themeschema in every per-module step.

**Review 0006** (Gemini 4 Argon (Medium), about 70k tokens): 10 findings.
Nine changed the code:
- a nil `Config` is refused;
- `//` and control characters are refused in the prefix;
- the cache key and stylesheet come from one snapshot of the config;
- the bundle is built once;
- `CheckFont` matches §3.2's five keywords exactly;
- boundary tests for fonts and radius;
- themeschema tests all eight keys and mutability.

The tenth is themeschema's `replace`: V7 now spells out the order for
tagging it after ui.

**Next.** V2: `ui/catalog`, `catalogtest`, and uitest's `CheckNames` and
`CheckSelectors` (A4).
## 2026-11-22 — D6: the visual packages' interface is locked

epistemer locked [visual-interface.md](visual-interface.md) with the
defaults of V1–V18. Those include:
- `…Spec` builders;
- `popovertarget` menus;
- keys `ui.<package>.<component>.<name>`;
- no `Action` type until D7;
- `Neutral` plus five accent tones;
- Lucide icons;
- the drawer for narrow screens;
- the `ToastSwap` exception to ADR 0005, still to be written into the ADR
  (V5).

uitest's `CheckNames` and `CheckSelectors` are amendment A4. The document
stays separate from interface.md, whose §10 points to it.

The roadmap gains milestones V1–V7: theme, catalogue and checks, icons and
elements, inputs, components, layout, release v0.2.0.

**Next.** V1, `ui/theme`.
## 2026-11-22 — Interface draft of the visual packages (D5)

**Landed** (this PR):
- docs/visual-interface.md: the interface of theme (with the nested
  `themeschema` module), icons, elements, inputs, components, layout and
  the catalogue, the ui script's contract and ui's i18n keys, for
  epistemer to lock at D6. On lock it folds into interface.md;
  interface.md itself is unchanged;
- review 0005;
- the roadmap's D5 ticked.

**What the draft settles** (each open choice is a question, V1–V18, with a
default):
- builders return `…Spec` values (finding 7);
- `theme.Config`'s getters carry the group prefix (`ThemeBrandColor()`),
  pinned with `GoName("Theme")`, and theme exports `CheckColor`,
  `CheckFont` and `CheckRadius` for cfgx's `.Validate`;
- ui's keys are `ui.<package>.<component>.<name>`; the spike's keys are
  renamed;
- no `Action` type in this lock (Q6): confirmation is attributes, and the
  table (D7) brings one action description with placements;
- uitest's `CheckNames` and `CheckSelectors` are amendment A4, proposed;
- forms' `Placeholder` fallback: no change proposed (V15).

Every piece the spike didn't prove is marked, and §14 lists them: tabs,
toasts, the detail list, layout, most elements, `themeschema` and
`catalogtest` among them.

**Review 0005** (Gemini 4 Argon (High), about 97k tokens, the milestone's
one Argon (High) run; Gemini 3.8 Flash (Low) restated three cut findings):
18 findings, 5 blocking. All handled, two answered rather than taken
(file kinds are forms' file input, not D7's uploads; V15's default stays).
The ones that changed the design:
- the toast region is a manual popover, so it shows above a modal
  dialog, and each toast is a `<ui-toast>` element;
- the app shell's narrow-screen drawer is driven by the ui script through
  `aria-expanded`, not a popover;
- an unknown asset release is a 404 again, as spiked: serving the current
  files would mix modules of two releases;
- a `Neutral` tone, so a neutral Cancel button can be expressed;
- a required slot left unfilled panics.

**Waiting on epistemer:** the D6 lock of visual-interface.md, with
answers to V1–V18 or their defaults.

## 2026-11-22 — Guidance cross-check of the spike

epistemer upgraded Node to 24, so the `modern-web-guidance` tool now runs.
Its guides for menus, dropdown placement, dialogs, dark mode, validation
and toasts were checked against the spike (visual-design.md §11,
finding 20). Most of the spike's choices hold. Four corrections go to D5:
- Browser support: anchor positioning has limited availability (Firefox
  lacks it), and popover is only newly available. ADR 0004's figures are
  amended.
- Menus: `commandfor` triggers and native `focusgroup`, and a fallback
  placement that flips and clamps.
- Fields: hints above the control, and an "Error" text alternative on
  error messages.
- Dialogs: scroll containment, and `autofocus` on the confirmation's
  Cancel.

## 2026-11-22 — Design of the visual packages (D1–D3)

**Landed** (this PR):
- docs/ui-inventory.md (D1);
- ADR 0004 (ui's own CSS) and ADR 0005 (client behaviour, proposed);
- docs/visual-design.md (D3 draft);
- review 0002;
- the roadmap's D track (D1–D7).

**Decided by epistemer (2026-11-21).** After a brainstorm on four axes
(volume, atomic granularity, splitting logic from UI, a configurable
theme), ui ships its own CSS, written in modern CSS, with no Tailwind and no
Flowbite. Claude's reading of the other axes is in visual-design.md §2:
- coverage instead of volume;
- layers with one dependency rule instead of atomic design's vocabulary;
- splitting logic from the view only where there is logic;
- only tokens as configuration.

**Storybook or our own catalogue.** templ ships `templ/storybook`, which
runs Storybook's server mode. Claude recommends our own catalogue
(visual-design.md §8, Q8):
- `templ/storybook` brings Node back (`npx sb init`, `npm run
  build-storybook`).
- Its arguments are only text, booleans, ints, floats and objects. ui's
  components take `i18n.Ter`, enums, slots and context values, so every
  story would need a hand-written wrapper anyway.
- It adds `rs/cors` and `x/mod` to go.mod.
- Storybook's server mode is a less maintained corner of Storybook:
  `@storybook/server-webpack5` is at 9.1.x, with no word on version 10.
- Our own registry drives goldens, id and name checks inside `go test`.

**Review 0002** (Gemini 4 Argon (High), about 71k tokens; Argon was out of
cooldown): 19 findings. Two were rejected after checking:
- Generated ids are process-wide, so htmx swaps don't collide.
- Depending on models' interfaces is epistemer's chosen direction.

Six blocking findings changed the design:
- Tailwind v4's Preflight would beat ui's layers unless ui sits after
  `base`.
- A CSS-only popover fallback can't anchor in the top layer.
- HTML keeps the first of two duplicate attributes, so behaviour
  attributes are merged in Go.
- A rendered `aria-expanded` goes stale.
- Per-row confirm dialogs break inside menus and forms; one shared dialog
  replaces them.
- `embed` can't reach sibling packages, so all CSS lives in `theme/css/`.

Q2's default moved from variadic options to builders, as in forms.

**Tools.**
- The `modern-web-guidance` skill can't run here: its package needs Node 22
  or newer, and this machine has Node 20.20.2 (snap). The browser-support
  claims in ADR 0004 come from web searches: invoker commands reached
  Baseline in December 2025, anchor positioning in January 2026 (Firefox
  147).
- `templ/storybook` was read in the module cache (v0.3.1118).

**Decided by epistemer (2026-11-22),** each on its default:
- Q2: components take options through builders, as in forms;
- Q3: variants in markup are data attributes;
- Q4: the accent colours, font families and radius scale can change at run
  time;
- Q7: ADR 0005 is accepted.

Q1, Q5, Q6, Q8 and Q9 stand on their defaults, open to change until the
D6 lock.

**D4 part 1** (PR #17, Claude):
- **theme:** the token table; OKLab and WCAG maths; accent partners derived
  for contrast; the stylesheet joined in layers under one hash; the
  run-time theme stylesheet; ADR 0004's CSS checks.
- **elements:** the Button builder and attribute merging.
- **a11y:** the names and selector checks.
- **catalog**, and Button stories.

Findings are in visual-design.md §11. The two that changed the design:
- Shades are derived in Go, away from the on-colour.
- Mixing is in OKLab, not OKLCh.

Checked in the browser pane:
- LTR, RTL, light and dark;
- RTL mirroring;
- live re-theming under a strict CSP, with no violations.

**Review 0003** (Gemini 4 Argon (Medium), about 71k tokens): 20 findings,
all handled. The main ones:
- the catalogue's unescaped URLs;
- a possible panic in `Derive` on admin input, now replaced by a stepping
  loop and proven by a sweep of 5,832 colours;
- disabled links still reaching htmx;
- gaps in the CSS checker and the names check;
- the selector check comparing classes and attributes separately.

**D4 part 2** (PR #18, Claude):
- **inputs:** a field rendered from `forms.Control`: label, help and error
  wired by generated ids, `aria-invalid`, `dir="ltr"` for email, the
  condition handed to the script.
- **components:** Dialog with slots and `DialogClose`, the shared
  confirmation, Menu with `<ui-menu>`.
- **ui script:** the condition evaluator, the menu, the confirmation.
- **internal/jstest** (ADR 0006): runs the evaluator against forms' 21
  vectors through goja, in a module of its own.

Checked in the browser:
- conditions showing and hiding fields;
- the menu's keyboard, focus and anchoring in LTR, RTL and inserted rows;
- Dialog's open and close, with focus returning to the trigger;
- the confirmation: cancel; and confirm, which submits once.

Bugs found that way, all fixed (visual-design.md §11): `this.popover` on a
custom element, a select without an empty option, and a confirmation
waiting on a deferred `close` event.

**Review 0004** (Gemini 4 Argon (High), about 80k tokens): 14 findings,
plus a verdict that §11's conclusions needed correcting before D5. Two
were rejected after checking:
- htmx forms don't double-confirm;
- relative units are allowed by ADR 0004.

The rest were fixed and checked again in the browser:
- the confirmation rebuilt as one capture-and-replay path;
- `refresh` no longer enabling what the app disabled;
- an htmx swap inside a form now refreshed;
- groups as fieldsets that carry their conditions;
- a three-state toggle as a Yes/No select;
- `multiple` for files, and the server's ticked rule for checkboxes;
- a textarea's leading newline kept;
- disabled menu items, arrow keys on the trigger, and typeahead.

Claude's own reading of forms' bind code found one more gap, which the
reviewer missed: the client read `["fixed", " "]` as absent and the server
as "fixed". Two vectors were added, and both evaluators pass them.

D4 is done.

**For epistemer (API):** forms' `Control.Placeholder` falls back to the
label (finding 12), so a renderer can't tell an explicit placeholder from
none. ui works around it by comparing texts. Exposing whether one was set
would be a forms amendment.

**Next.** D5: interface.md sections for what Q9 locks, built from the
spike, with a cross-model review.

## 2026-11-21 — v0.1.0

**Landed.** PR #13 (F6.1), PR #14 (R.1), PR #15 (release review fixes),
the CHANGELOG, the tag, and the global `ui` skill. Every roadmap item to
v0.1.0 is done.

- **F6.1** (Gemini 3.8 Flash High, about 205k tokens; reviewed by Claude,
  one snippet fixed): forms godoc examples that run as tests, the README
  guide for declaring and handling a form, `forms/rule/CONTRIBUTING.md`.
- **R.1** (Claude): docs/api-review.md. All 228 exported identifiers are in
  interface.md, every func signature matches, every identifier is
  exercised by a test; a new test pins §5.10's default kinds.
- **Release review** (Gemini 4 Neon XHigh, about 261k tokens; Argon was in
  cooldown, so the strongest available ran): 19 findings, two rejected
  after checking (flushing does reach templ's buffer; the root package has
  its doc.go). The rest fixed with tests and recorded as **amendment A3**:
  junk list keys at any depth, a POST whose query net/http refuses,
  checkbox state before Bind, an unreadable Submit and sub-forms, Enum with
  foreign options, Options kinds, root When, NaN and negative bounds,
  `*/*`, `Matches`' pattern portability, `URL` host names, uitest on whole
  pages and ids with white space, the `-update` flag, collect-pass order.
- **Models this session** (agy): Gemini 3.8 Flash (High and Medium) for
  delegated leaf items and small reviews, Gemini 4 Neon (Medium, High,
  XHigh) standing in for Argon, which was in quota cooldown all session.

**Waiting on epistemer.** Amendments A1–A3 (§10) are clarifications Claude
made as owner; epistemer may overturn any of them. Nothing else.

**Next.** v0.1.x fixes as they come; the next ui packages (icons,
elements, inputs, components, layout) start from their own designs.

## 2026-11-21 — Forms F5 (nesting, Control, formstest) and kernel M5.1 (docs)

**Landed.** PR #11 (F5) and PR #12 (M5.1). The kernel is feature-complete
for v0.1.0 (M1–M5.1); forms has F1–F5.

- **F5** (Claude). A form is a tree of fields, groups and lists. Groups
  and lists post under paths; lists bind sparse indices, renumber, honour
  limits, and expose a template row at `__i__`. Activity and cycle checks
  follow container chains; conditions can't reach into a list's items
  from outside. `Control` is the typed-free view for renderers; formstest
  has `Submit`, `Errors`, `AssertLabelled`. examples/shop has the
  promotion form of forms-design §11.2 (conditions, two groups, a list
  with conditions inside items).
- **F5 review** (Gemini 4 Neon High, about 86k tokens): junk list indices
  could trigger `ErrMaxItems` and push out real items; the building flag
  that lets a list declare items after Bind covered the whole form; list
  items accepted `When` and ignored it; an inactive group kept its error;
  `AssertLabelled` skipped empty lists; a template row inherited its
  list's inactivity, so a renderer would emit it disabled. All fixed with
  tests, recorded as **amendment A2** (clarifications; epistemer may
  overturn them). staticcheck caught an unused test variable in CI.
- **M5.1** (Gemini 3.8 Flash High, about 277k tokens; reviewed by
  Claude): godoc examples that run as tests for Provider, ID, slots and
  uitest; the package overview; README guides for writing and filling a
  slotted component, with HTML from a real run. Claude corrected one name
  (`Provider.Provide`).

**Next.** F6 (forms docs: godoc examples, README guide, the rule
contributor note), then R.1 (API review against interface.md) and R.2
(CHANGELOG, v0.1.0, the `ui` skill).

## 2026-11-21 — Kernel M4 (slots) and forms F4 (conditions)

**Landed.** PR #9 (F4) and PR #10 (M4). The kernel's milestones M1–M4 are
complete; forms has F1–F4.

- **M4** (Claude: M4.1–M4.4). `NewSlot`, `NewSlotOf`, `Slotted`, `Place`,
  `Filled`, `Value`; lexical lookups in fill content (rule 5); the five
  errors, each naming the slot's declaring line; `ErrInBlock` from
  `Provide`, `Scope` and `Place` in a block. Go reports a package-level
  initializer's line as `<autogenerated>:1` when a cross-package call is
  inlined into it, so `NewSlot` and `NewSlotOf` are `//go:noinline`.
- **M4.5–M4.6** (Gemini 3.8 Flash High, about 278k tokens; reviewed by
  Claude): shop components with goldens and `CheckIDs` (cart modal,
  dropdown, an order table whose rows put their actions in a dropdown's
  fill and target their own keyed ids, a product card dropping an empty
  footer), and benchmarks below. Allocations per render do not grow with
  the fill: 26 for a small fill and for one 100× larger.
- **M4 review** (Gemini 4 Neon High, about 77k tokens): templ buffers a
  block's output, so stray content before another collect-pass error was
  never seen, against §9.1's first-error rule. The kernel now flushes the
  block's buffer before reporting its own errors. Also: the sink's first
  stray content and a block that drops a write error, nil and zero slots,
  the declaring frame matched by import path. All fixed with tests.
- **F4** (Claude). `Cond`, `When`, cycle detection, the wire format, 21
  conformance vectors in `forms/testdata/cond` for a client evaluator.
- **F4 review** (Gemini 4 Neon High, about 60k tokens): a rule could read a
  later inactive field's stale value (now every inactive field is cleared
  before any rule); a checkbox's `Eq` disagreed with `Filled` for an "on"
  submission (both now read its value); cycle checks were exponential on
  chains of `In` (now linear). All fixed with tests.

**M4.6 benchmarks** (2026-11-21, Intel(R) Core(TM) i5-5300U CPU @ 2.30GHz)

```
goos: linux
goarch: amd64
pkg: github.com/epistemer/ui
cpu: Intel(R) Core(TM) i5-5300U CPU @ 2.30GHz
BenchmarkThreeSlots-4   	  403431	      4749 ns/op	    1833 B/op	      40 allocs/op
BenchmarkThreeSlots-4   	  307603	      3820 ns/op	    1833 B/op	      40 allocs/op
BenchmarkThreeSlots-4   	  367142	      3088 ns/op	    1833 B/op	      40 allocs/op
BenchmarkNestedFive-4   	  104952	     14609 ns/op	    5468 B/op	     118 allocs/op
BenchmarkNestedFive-4   	   96424	     15421 ns/op	    5468 B/op	     118 allocs/op
BenchmarkNestedFive-4   	   77715	     16352 ns/op	    5469 B/op	     118 allocs/op
BenchmarkFillSize/small-4         	  457353	      2311 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/small-4         	  564014	      2783 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/small-4         	  527125	      2556 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  349038	      3696 ns/op	    1208 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  329400	      3462 ns/op	    1209 B/op	      26 allocs/op
BenchmarkFillSize/100x-4          	  341342	      4285 ns/op	    1209 B/op	      26 allocs/op
PASS
ok  	github.com/epistemer/ui	16.413s
```

**Next.** Kernel M5.1 (docs and README guides) and forms F5 (groups,
lists, Control, formstest), then F6 and the v0.1.0 release items.

## 2026-11-21 — Kernel M3 and forms F3

**Landed.** PR #7 (F3, conversion) and PR #8 (M3, the test kit).

- **F3.** `Conv`, `As`, `Number`, `Parsed`. Rules run stage by stage, and
  an earlier stage's rule wins over a later conversion error. `Number`
  picks its parser once per field (no reflection per call). examples/shop
  wraps `Parsed` as `money.Decimal`; 0.10000000000000000001 binds exactly.
- **M3.** `Render`, `RenderErr` and `Context` (Claude); ADR 0003 accepts
  golang.org/x/net/html for uitest only, and a test checks the kernel
  still links none of it. `Golden` with its normaliser and `CheckIDs` were
  delegated to Gemini 3.8 Flash (High), about 327k tokens; Claude reviewed
  them and made the normaliser escape text, so text that looks like markup
  never reads as an element.
- **Review** of F3 and M3.1: Gemini 4 Neon (Medium), about 33k tokens. A
  nil interface value between conversion stages could panic on user input;
  `As` after `Set` desynchronised value and text. Both fixed, with the
  tests it asked for; `Render` now derives from `t.Context()`.

**Next.** Kernel M4 (slots), the kernel's largest milestone, and forms F4
(conditions).

## 2026-11-21 — Kernel M2 and forms F2

**Landed.** PR #5 (M2, element ids) and PR #6 (F2, the forms core).

- **M2.** `NewID`, `Scope`, `Get`, `Part`, `Key`, injective escaping,
  `IDSource`, `Sequence`, `WithIDs` and the process source (6 random
  base-36 characters, rejection-sampled, and a base-36 counter). 640,000
  ids from 64 goroutines without a duplicate under -race. An exported
  `ErrBadToken` was drafted and removed before review: the locked
  interface has none, so a bad token is a plain error naming the source.
  M2.3's fuzz tests were written by Gemini 3.8 Flash (Medium, about 110k
  tokens) and strengthened by Claude after review (strict decoding, two
  independent parts, hyphenated names).
- **M2 review**: Gemini 4 Neon (Medium), about 36k tokens. No critical
  finding; a duplicate id drew tokens before it was refused, the random
  prefix had a modulo bias, and tests lacked K3/K8 cases. All fixed.
- **F2.** `Form`, `Field[T]`, `Source`, the non-generic constructors, and
  `Enum` and `Options`, pulled forward from F3.2 because the store example
  needs them. `examples/shop` has a hand-written stand-in for generated
  models and the store form of forms-design §11.1, tested through its
  create, invalid, service-error and edit paths and in two locales. The
  handlers in the example redirect by hand: wiring them to handlers'
  generated routes is for the F6 example.
- **F2 review**: Gemini 4 Neon (High), about 93k tokens. Ten findings, all
  valid. The serious one: `Options()` with an empty list switched
  membership checking off, so any value passed. Also: conversion errors
  appeared only at Validate, an `Enum` let rules read an unchecked value,
  `Submit` didn't refuse a second call when the request was unreadable.
  All fixed with tests.
- A force-push after a rebase was blocked by the session's safety checks;
  branches now take main by merge.

**Models.** Argon stayed in quota cooldown all session; Neon (High) and
Neon (Medium) stood in for Argon (High) and Argon (Medium).

**Next.** Kernel M3 (test kit) and forms F3 (conversion: `Conv`, `As`,
`Number`, `Parsed`).

## 2026-11-21 — Forms F1: rules

**Landed.** PR #3 (F1.1) and PR #4 (F1.2). `ui/forms/rule` is complete for
v0.1: `Rule[T]`, `New`, `Err`, `Check`, `Attrs`, the twenty declared
defaults, and the whole catalogue of interface §6.2.

- **Amendment A1** (clarification, Claude): error texts follow errors'
  convention, lower case with no final period. epistemer may overturn it.
- `NotBefore`/`NotAfter` were written by Claude with an internal hook
  (`forms/internal/hook.DateOnly`) through which forms makes them compare
  calendar dates on a Date field and emit date attributes (§9.4).
- F1.2 was delegated to Gemini 3.8 Flash (High) in a worktree outside the
  repository, about 229k tokens; zp's data was read from a scratch clone,
  so zp itself stayed untouched. Claude reviewed it: correct, no private
  names; Claude tightened the no-compile test so each case names the
  compiler error it expects.
- F1.1's review (Gemini 3.8 Flash High, about 187k tokens; Argon still in
  cooldown): the date-only hook kept datetime attributes, errors showed the
  bound in its own zone, and `run` used a type assertion. All fixed, with
  tests for every default error.

**Next.** Kernel M2 (element ids) and forms F2 (the core), in parallel.

## 2026-11-21 — M1: context values

**Landed** (PR #2). `Provider[T]` (NewProvider, Provide, With, Value) and
the internal `block` helper that every kernel component uses for K3.

- Values are boxed in the context, so a provided nil interface value wins
  over the default (found by review).
- A padding byte keeps providers of zero-sized types distinct (K8).
- Review: Gemini 3.8 Flash (High), about 193k tokens (Argon in cooldown).
  Five findings: the nil-interface bug and three missing tests were fixed;
  "missing package" was wrong (it is on main); `ErrInBlock` in `Provide` is
  M4.4's, once the collect pass exists.

**Next.** Forms F1 (rules), then kernel M2 (element ids).

## 2026-11-21 — M0: the repository, the designs, the interface draft

**Before starting.** The two designs existed only as Google Docs ("neozp ui
kernel roadmap", 2026-11-17; "neozp forms API design", 2026-11-16). Checked
against neozp as it stands:

- Still true: Go 1.27.4 is installed and its spec allows type parameters on
  methods; templ v0.3.1118 is the latest release; auth's CSRF needs no token
  on a form (auth D13); errors' `Key`/`Key1`, `Invalid` (400), `T`, `All`,
  `KindOf` are as forms uses them; nothing in neozp carries a time zone.
- Drift, now fixed in the docs (brief §9, forms-design §15): forms moves
  into this module; forms adopts models' `option.SelectOption` (models
  ADR 0006); error keys follow `errors.<area>.<name>`; the forms examples use
  handlers' `net/http` admin handlers instead of echo; the JSON source is
  dropped (handlers' API validates through oapi-codegen); the kernel's
  v1.0.0 is no longer gated on zp; the kernel's claim that `net/http` turns
  a panic into a 500 was wrong (it aborts the connection).
- zp's file names and identifiers were removed from the forms design, which
  is now public; the salvage guide to zp's files lives in the maintainers'
  private notes.

**Decided by epistemer (2026-11-21).**

1. One module, `github.com/epistemer/ui`, Go 1.27: kernel = root package
   `ui` (+ `ui/uitest`), forms = `ui/forms` (+ `rule`, `formstest`).
2. The repository is delegated to Claude until v1.0.0 on the terms of
   ownership.md; interface.md's lock and API design stay epistemer's.
3. Catalogue rules carry declared errors, overridable with `.Err(err)`.
4. forms consumes `option.SelectOption`; its own `Option[T]` goes.
5. The other defaults of both designs are accepted (brief §8,
   forms-design §14).

**Landed.**

- M0.1: go.mod (Go 1.27, templ v0.3.1118), LICENSE, doc.go, CLAUDE.md, CI
  (gofmt, `templ generate` drift, build, vet, staticcheck, tidy,
  `go test -race` on Linux, macOS and Windows), `examples/shop` module.
- M0.2 (docs): brief.md, forms-design.md, roadmap.md (kernel M1–M5 and
  forms F1–F6 as independent tracks after M0), ADR 0001 (kernel primitives),
  ADR 0002 (forms model), ownership.md, interface.md (draft).
- M0.3: `internal/templcontract` pins K3 (a Go component that renders its
  block without `ClearChildren` leaks it to a children-rendering component
  inside; one that clears does not) and the collect pass's premise (a block
  of only calls, `if` and `for` writes only white space). Both pass on
  v0.3.1118.
- M0.3 compiler spike (Go 1.27.4, throwaway module, not committed): all
  compile — `As[U]` on `Field[T]`; `f.Number[int32]("n")` promoted through
  an embedded `*Form`; `PT` inferred in `Parsed[Dec]`; `Int().Is(Min(0))`;
  `is := f.Type.Eq`; `Enum` over `~string` + SelectOption; `Group` inferring
  `G`. Refused as intended: `Bool().Is(MaxLen(5))`. Also refused:
  `Number[int32]().Is(Min(0))` — a bound's type comes from its argument, so
  int32 fields write `rule.Min[int32](1)`. No fallback needed; interface §5.3
  records it.

- Cross-model review of interface.md ([review 0001](reviews/0001-interface-draft.md)):
  Gemini 4 Neon (High), about 71k tokens — Argon was in quota cooldown for
  4.6 hours, so the strongest available Gemini ran. 18 findings, all valid;
  resolved as interface §9 (behaviour in detail) plus fixes in §1, §5.1,
  §5.3, §5.7, §6.2 and §8. Two claims confirmed by a second spike: `Parsed`
  needs the wider constraint to accept `big.Int`, and `rule.MinItems(1)`
  cannot infer `T`.

**Lock.** epistemer locked interface.md on 2026-11-21, including the items
marked *(new)*, §9 and `Parsed`'s wider constraint. From here, a change to
what an app writes is an amendment (interface §10).

**Next.** Kernel M1 and forms F1, in parallel.

**Was waiting on epistemer** (now done): the lock of interface.md. Its new items, not in
either design, are marked *(new)*: formstest's API (§7), the error keys and
texts of the catalogue and of forms (§5.9, §6.3), and the condition
operator names on the wire (§5.7). Also new since the designs: §9's
behaviour in detail, and `Parsed`'s wider constraint (§5.3).
