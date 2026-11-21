# What the visual packages must do: an inventory of zp's UI

Read from zp's templates on 2026-11-22, as a record of needs, not a design
to copy. The counts come from the templates and their static assets; app
code that only calls a component through another is not counted. Nothing here
names zp's business: components are described by what they do.

## The stack, and what it costs

| Layer | zp today | Cost |
|---|---|---|
| CSS | Tailwind v3 with the Flowbite plugin and stylesheet; brand colours as hex scales in the Tailwind config | Colours are fixed when the CSS is built: no theming or white-labelling while the app runs. A Node build sits in front of every style change. |
| Markup | about 590 `class="…"` attributes in templates; long class strings repeated or kept in Go constants | Variants are copied strings, so a fix to one copy misses the others. |
| Dark mode | 615 `dark:` classes: every colour is written twice | A new colour is easy to forget in one mode; the OS preference is ignored. |
| Colours in markup | 1,030 raw palette classes (`gray-500`, `white`, `blue-600`) against 189 semantic ones (`primary-600`) | Components disagree: tabs use blue, buttons the brand colour. |
| Client | htmx (with its SSE extension), Alpine (16 components with `x-data`), 21 of zp's own scripts (about 2,250 lines), jQuery 3.2.1 slim with a flash-message library, and a small DOM helper; about 20 script tags on every page | Three ways to add behaviour, and every page pays for all of them. |

## What pages are made of

Ranked by calls from pages and row partials:

| Shape | Uses | Made of |
|---|---|---|
| App shell | 67 pages | collapsible side navigation (Alpine), top bar with global search, profile menu, notifications, locale and theme toggles |
| Detail page | 16 | a card of description-list rows, one row type per value kind (text 58 calls, link 21, bool 17, image 6, enum 6, date 4), plus an actions area |
| Client table | 17 | heads, rows, cells by kind (enum, bool, amount, link, image, date), row actions |
| Server table | 10 | search box, filter form, sortable heads, rows streamed over SSE, row and bulk actions, pagination with page size; the query, offset, limit and filters post as one form |
| Form | 40 | one component that renders a form from metadata and dispatches each field to an input: text, textarea, select, checkbox, three-state toggle, date (JS picker), file (with preview), remote select with suggestions, list of values, JSON editor, hidden, collapsible fieldset |
| Card and alert | 38 and 37 | a container; an inline message by level |
| Actions | 13 components | see finding 2 |
| Auth pages | 5 | a centred card, outlined inputs, password visibility toggle, submit button |
| Dialogs and menus | 12 and 4 | a modal (Alpine calls `showModal`), a dropdown with three hand-positioned placements |
| Icons | 63 files, 70 icons | inline SVG components |

Emails (6) and PDFs (5) are templ too. They need inline styles and no
script, so they are a separate concern from the web packages.

## Findings

1. **Theming is compiled in.** Hex scales live in the Tailwind config, raw
   palette classes outnumber semantic ones five to one, and dark mode is a
   second copy of every colour. ui needs tokens that can change while the
   app runs, and components that only read tokens.
2. **Variants turn into components.** Two button components share one
   300-character class constant because one renders `<button>` and the
   other `<a>`. Actions come in 13 components: where they sit (a row menu or
   a toolbar button) times what they do (link, htmx request, htmx with
   confirmation, form, POST, POST with confirmation, modal). Four of the 13
   have no caller at all, and four more are reached only through wrappers
   such as an edit or delete button. ui needs independent parameters, and
   confirmation as something that wraps any action.
3. **Value rendering is written twice.** Table cells and detail rows each
   have their own renderer per kind (bool, enum, date, link, image, amount).
   ui needs one set of value renderers that both use.
4. **Native HTML would remove most scripts:**
   - a dialog opened by a script (`commandfor` and `command="show-modal"`
     do it without one);
   - a dropdown positioned by hand in three placements (`popover` with
     anchor positioning);
   - a date picker written in JavaScript (`<input type="date">`);
   - a collapsible fieldset in JavaScript (`<details>`);
   - flash messages through jQuery (a live region).
5. **Accessibility has gaps:**
   - 11 screen-reader strings ("Close modal", "Loading...", "Next") are
     English literals, not i18n keys;
   - focus styles use `:focus` 256 times and `:focus-visible` 3 times, so
     focus rings show on mouse clicks;
   - menus carry almost no roles (one `menuitem` in the whole app);
   - the tooltip isn't referenced by `aria-describedby`.
6. **Visible text is sometimes a plain `string`.** The tooltip's text, a
   modal's title and some confirmation messages are `string`; their
   neighbours take `i18n.Ter`. ui takes `i18n.Ter` for every text a user
   sees.
7. **RTL is mostly right.** Logical classes outnumber physical ones 56 to
   12. The rest (a centred tooltip, `text-left` in tables, a few margins)
   are what a check would catch.

**What zp got right, to keep:**
- slots that pass values, and scoped ids (the kernel grew from both);
- a CSS-only tooltip that works on pages without Alpine;
- the theme read from a cookie on the server, so a page never flashes the
  wrong theme;
- dropdown placements named by logical sides;
- table rows streamed over SSE;
- typed placement enums instead of strings.

## Coverage for ui

What the packages must provide to cover these pages, by layer:

- **tokens:** colour roles with light and dark values, spacing, radius,
  type scale, shadows, focus ring, motion.
- **elements:**
  - icon;
  - button (as `<button>` or `<a>`);
  - badge;
  - alert;
  - card;
  - avatar;
  - spinner and loading placeholder;
  - tooltip;
  - separator;
  - heading;
  - value renderers by kind.
- **inputs** (on `forms.Control`):
  - text and textarea;
  - select;
  - checkbox and three-state toggle;
  - date and time;
  - file with preview;
  - remote select (a combobox over models' `listing.Labeler`);
  - a repeatable group (forms' list template row);
  - hidden;
  - fieldset;
  - password with a visibility toggle.
- **components:**
  - dialog and confirmation;
  - menu;
  - navigation tabs;
  - pagination;
  - table;
  - server table (over models' `listing.Source`: search, filters, sort,
    paging, row and bulk actions, streaming);
  - detail list;
  - toast region;
  - command palette for global search.
- **layout:** app shell (side navigation, top bar), auth shell, page header
  with breadcrumbs.

The JSON editor is the one input with no general need behind it; it waits
for a real form.
