# Visual packages: design draft

| | |
|---|---|
| Status | Draft. epistemer decided Q2, Q3, Q4 and Q7 on 2026-11-22; Q1, Q5, Q6, Q8 and Q9 stand on their defaults, open to change until the D6 lock |
| Date | 2026-11-22 |
| Roadmap item | D3 |
| Inputs | [ui-inventory.md](ui-inventory.md), ADR [0004](adr/0004-own-css.md) (accepted), ADR [0005](adr/0005-client-behaviour.md) (accepted), the brainstorm of 2026-11-21, the design review [0002](reviews/0002-visual-design.md) |

This draft covers what the visual packages share: layers, tokens,
conventions, behaviour, accessibility and the catalogue. Each package gets
its own section in interface.md after the slice (D4) has tried these
defaults. Server-state components (tables, filters, paging, remote select,
uploads) get a design of their own, as forms did, and stay out of the first
lock (Q9).

## 1. Scope

**In scope:**
- tokens and the theme;
- icons and elements;
- inputs rendered from `forms.Control`;
- components (dialog and confirmation, menu, navigation tabs, toast region,
  detail list);
- layout (app shell, auth shell, page header);
- the ui script;
- the catalogue.

**Out of scope:**
- emails and PDFs (inline styles, no script);
- charts;
- a utility-class framework;
- a JavaScript framework;
- moving zp.

## 2. Principles

These come from the brainstorm, where epistemer proposed four axes and
Claude reviewed them:

1. **Coverage, not volume.** A component exists because a real page needs
   it: the inventory's coverage list, or a neozp app later. Every exported
   identifier is API kept until v1.0.0, so interface.md lists each package's
   surface, and growing it needs a need.
2. **Layers with one rule.** The layers are tokens → elements (with icons) →
   inputs → components → layout. Each is built only from the public API of
   the layers below it, plus the kernel and forms. A user can therefore
   rebuild any composite from public parts, and that is what "dig deeper"
   means here. Slots and attributes are how a caller reaches into a
   composite without rebuilding it.
3. **Variants are parameters, not components.** One `Button` has
   independent, typed parameters (variant, size, tone). Whether it renders
   `<button>` or `<a>` follows from whether it has an href. A floating
   action button is a button plus a shape and a position.
4. **Logic splits from the view only where there is logic.** There are two
   kinds:
   - **Server state** (a table's sort, filters and page; an upload) becomes
     a Go type read from the request, the way forms works. Its renderer is
     replaceable.
   - **Behaviour** (dialog, menu, combobox) becomes attributes handed to the
     caller's markup through `SlotOf[templ.Attributes]`, as the kernel's
     `CartToggle` example does.

   Anything else is a view.
5. **Only tokens are configuration.** Structure (a divider on a card, a
   label beside its field rather than above it) is set through variants and
   slots, never through flags in a config object.
6. **Every text a person sees is an `i18n.Ter`,** screen-reader text
   included. ui's own strings use a new key namespace,
   `ui.<package>.<name>` (`ui.dialog.close`, `ui.pagination.next`).
7. **Accessibility is a tested contract.** Each interactive component names
   the ARIA Authoring Practices pattern it follows, and tests check what can
   be checked without a browser (§7).
8. **RTL and dark mode hold by construction where CSS decides them.**
   Logical sides and token colours are enforced by checks on ui's CSS
   (ADR 0004). What HTML decides (`dir` on the page and on
   left-to-right inputs) is in the markup and covered by goldens.

## 3. Packages

| Package | Layer | Holds |
|---|---|---|
| `ui/theme` | tokens | the token table; the reset and base layers; every stylesheet of ui (`theme/css/`, ADR 0004) and the handler that serves them with the script; the run-time theme |
| `ui/icons` | elements | inline SVG icons (`currentColor`, hidden from assistive technology unless labelled, mirrored in RTL when directional) |
| `ui/elements` | elements | button, badge, alert, card, avatar, spinner, tooltip, separator, heading, value renderers |
| `ui/inputs` | inputs | widgets rendered from `forms.Control` |
| `ui/components` | components | dialog and confirmation, menu, navigation tabs, toast region, detail list |
| `ui/layout` | layout | app shell, auth shell, page header |
| `ui/catalog` | dev | the story registry and its HTTP handler (§8) |

The kernel stays at the root.

In Go, the visual packages don't import `ui/theme`: they emit class names
and attributes, and the CSS for them lives in `theme/css/`. So the package
that serves the CSS sits above them in the import graph, while its layers
sit below them in the cascade.

## 4. Tokens and the theme

**The token set.** Colours are named by role, not by palette:
- `--ui-color-surface`, `-surface-raised`, `-text`, `-text-muted`,
  `-border`, `-focus`.
- Each **accent role** has six tokens: brand, danger, warning, success and
  info. For brand they are:
  - `--ui-color-brand`, the fill;
  - `-on-brand`, text on the fill;
  - `-brand-hover` and `-brand-active`, the fill's shades;
  - `-brand-text`, the role used as text on a surface;
  - `-brand-subtle`, a tinted background for alerts and badges.

  The other four roles follow the same pattern. The fill, on-colour and
  shades are the same in both schemes; text and subtle are derived per
  scheme.

The other tokens are scales:
- space `--ui-space-1`…`8`;
- radius `--ui-radius-s`, `m`, `l`, `full`;
- border width;
- type `--ui-text-xs`…`2xl`;
- fonts `--ui-font-sans`, `--ui-font-mono`;
- shadow `--ui-shadow-1`…`3`;
- motion `--ui-duration-s`, `m` and `--ui-ease`.

Colour values use `light-dark()`. No accent colour is mixed in the
browser: all of them are derived in Go, below.

**Schema-first.** One Go table in `ui/theme` declares every token: its name,
kind (colour, length, font, shadow, duration), its light and dark defaults,
and whether it can be overridden. From that table:
- `ui/theme` renders the token layer when the program starts. There is no
  generated CSS file (ADR 0004).
- `ui/theme/themeschema` builds the cfgx group for the tokens that can be
  overridden, as `authschema` does for auth. Its keys are `Mutable`, so an
  admin changes them in the running app, as they edit translations. Each
  key validates its value by kind: a parsed colour, a font-family list of
  quoted names and generic families, a parsed length. That keeps a stored
  value from injecting CSS into a stylesheet the CSP trusts.
- `theme.Config` is the getter interface the theme reads, which cfgx's
  generated settings satisfy. A test keeps the interface and the table in
  step.

**At run time.**
- The theme stylesheet holds `:root { --ui-color-brand: …; }`, built from
  `theme.Config` and validated again as it is built. Invalid values fall
  back to the default.
- Its URL carries a hash of the current values, recomputed when the values
  differ from the last ones seen (three or four string comparisons per
  page). It doesn't depend on cfgx's change callbacks.
- A request for an older hash gets the current theme with
  `Cache-Control: no-cache`, not a 404 and not an immutable copy. A page
  rendered just before a change therefore still gets a theme, and caches
  never keep a stale one.
- The page links `ui.css` and then the theme stylesheet through one
  component. It takes the theme as an argument, so no per-request setup is
  needed. Its name is decided in D5.

**Contrast is built, not just checked.** When a colour is set, Go (stdlib
maths, no dependency) derives its partners:
- **on-colour:** black or white, whichever contrasts more with the fill.
  One of them always reaches 4.58:1.
- **hover and active:** the fill mixed in OKLab away from the on-colour
  (towards black under white text, towards white under black text), at 88%
  and 76%. Moving away from the on-colour only raises its contrast, so it
  holds in every state.
- **text colour:** the role shifted in oklch lightness, with chroma reduced
  to stay inside sRGB, until it reaches 4.5:1 against the stricter of
  surface and raised surface, in light and in dark.
- **subtle colour:** a tint that keeps text-colour contrast.
- **focus ring:** checked for 3:1 against both surfaces.

zp's lime brand shows why this matters: it works as a fill under black text
but is unreadable as text on white. With derivation, an admin can't pick an
unreadable theme, and nobody has to reject the colour they chose.

## 5. Conventions every component follows

- **Content goes in children; text parameters are `i18n.Ter`.** A component
  takes what it can't do without as parameters: a dialog's title, a link's
  href. Free content goes in children or slots.
- **Attributes pass through, merged in Go.** A component accepts extra
  attributes (`hx-*`, `data-*`, `aria-*`), and its doc names the element
  that receives them: for an input, the `<input>`, not its wrapper. They
  are merged with the component's own before one spread is rendered:
  - keys are compared without regard to case;
  - `class`, `aria-describedby` and `aria-labelledby` are joined;
  - keys the component's behaviour needs (`id`, `name`, `type`, `role`,
    `commandfor`, `command`, `aria-invalid`) replace the caller's.

  Rendering both would leave the first one in force, because HTML keeps the
  first copy of a duplicated attribute. This is the backlog's `Merge`,
  which lives in `ui/elements` until a second package needs it.
- **One class per component, `ui-<name>`, plus variant attributes**
  (Q3). State comes from HTML and ARIA (ADR 0004).
- **Ids come from the kernel:** generated ones for wiring inside a render
  (`ID.Get`, unique across the process, so fragments swapped in by htmx
  never collide), keyed ones for what htmx targets (`ID.Key`). Cloned
  template rows get fresh ids from `<ui-repeat>` (ADR 0005).
- **A disabled link is not a link.** `Button` with an href and disabled
  renders `<a>` without `href`, with `aria-disabled="true"` and
  `role="link"`. CSS styles `:is(:disabled, [aria-disabled="true"])`.
- **Value renderers are shared.** `ui/elements` renders a value by kind:
  - bool;
  - date;
  - enum (`option.SelectOption`);
  - link;
  - image;
  - text already formatted;
  - amount: formatted text, with tabular numerals and isolated direction,
    so a currency stays in place in RTL.

  Table cells and detail rows both use these, so the inventory's duplicated
  renderers per kind (finding 3) don't come back.
- **Actions are described once and placed anywhere.** An action is a label,
  an optional icon, and what it does: follow a link, make an htmx request,
  submit a form, or open a dialog. A placement renders it as a toolbar
  button or a menu item. `Confirm(msg)` adds a question to any action,
  answered by the one shared confirmation dialog (ADR 0005). This replaces
  zp's 13 action components with two ideas (Q6).

## 6. Behaviour

Everything not listed below is native HTML or CSS.

| Need | How |
|---|---|
| Dialog | `<dialog>` opened by `commandfor` and `command="show-modal"`; the ui script stands in where invoker commands are missing |
| Confirmation | one shared dialog in the layout shell, driven by `htmx:confirm` and `data-ui-confirm` |
| Menu | `popover` opened by `popovertarget` (Baseline 2024), which also makes the trigger the popover's implicit anchor, so `position-area` places it with no anchor name and no inline style; `<ui-menu>`: focus to the first item, arrow keys, `aria-expanded` kept in step, and placement where anchoring is missing |
| Tooltip | CSS on hover and focus, the text referenced by `aria-describedby`, Esc to dismiss from the delegated listener |
| Disclosure | `<details>` |
| Fieldset | `<fieldset>` with `<legend>` (so the group has a name, and `disabled` reaches every control); a collapsible one wraps its content in `<details>` |
| Navigation tabs | links with `aria-current="page"`: the navigation pattern, not the APG Tabs pattern. In-page tabs wait for a page that needs them |
| Remote select | `<ui-combobox>`: the APG combobox pattern, with suggestions fetched by htmx and the chosen label from models' `listing.Labeler` |
| Repeatable group | `<ui-repeat>` clones forms' template row, renumbers `__i__` and gives the clone fresh ids |
| Conditional fields | the forms condition evaluator, a module of the ui script, reading each control's serialised `When` |
| Toasts | `<ui-toasts>`, a live region kept in the shell; items arrive through an out-of-band `beforeend` swap and pause on hover and focus |
| Theme toggle, password toggle | delegated listeners (ADR 0005) |

## 7. Accessibility contract

- Each interactive component's doc comment names the APG pattern it follows
  and lists its keyboard behaviour.
- uitest gains a check for accessible names:
  - **It requires a name on:** every button, link, form control, image and
    dialog. The name can come from text, `aria-label`, `aria-labelledby`,
    a `<label>`, or `alt`.
  - **It ignores:** text inside `aria-hidden` or `hidden` subtrees, and
    hidden inputs.
  - **It accepts** `alt=""` on a decorative image, but rejects a missing
    `alt`.
  - **It also checks** that every `<fieldset>` has a `<legend>`, and that
    several `<nav>` landmarks on one page have distinct names.

  `CheckIDs` already checks references.
- The catalogue's tests run both checks on every story.
- Keyboard behaviour and focus need a browser. They wait for the browser
  test job (ADR 0005) and are listed in each component's doc until then.

## 8. Catalogue

- A story is a name and one or more variants, each
  `func(ctx context.Context) templ.Component`.
- `ui/catalog` holds the registry and an `http.Handler`, so an app can list
  its own components in the same page.
- ui's own stories live in `internal/stories`, one file per package, served
  by `go run ./cmd/catalog`.

**The page** shows each variant in a two-by-two grid of iframes: LTR and
RTL, light and dark. A switcher sets the width (phone, tablet, desktop)
rather than tripling the grid. A token panel edits the variables of every
iframe live through CSSOM, which a strict CSP allows, so the panel doubles
as a demonstration of theming at run time.

**The same registry drives `go test`:**
- a golden file for every variant, in LTR and RTL where markup differs;
- `CheckIDs`;
- the names check;
- a selector check: every `ui-*` class and variant attribute in the goldens
  has a rule in ui's CSS, and every component rule in the CSS is used by
  some story.

HTML goldens can't see styling. The CSS checks in ADR 0004 cover the
mistakes they can, and screenshots come later with a browser job, possibly
with axe loaded in the page.

This beats Storybook for ui because its tests run in `go test` with no Node
(see the 2026-11-22 status entry). `templ/storybook` remains possible later,
in a module of its own, over the same registry.

## 9. API decisions

epistemer decided Q2, Q3, Q4 and Q7 on 2026-11-22, each on its default.
The others stand on their defaults until the D6 lock.

| # | Question | Default | Alternatives | State |
|---|---|---|---|---|
| Q1 | Package names and split (§3) | as in §3 | merge icons into elements; a `ui/components` per component | default |
| Q2 | How a component takes options | **builders, as in forms**: `@elements.Button().Variant(elements.Outline).Disabled(!ok) { … }`. A builder is an immutable value that renders. Each method is listed under its own type in `go doc`; the same name on two components never collides; a condition is just an argument; the zero call is the common look | typed variadic options (`@elements.Button(elements.Outline)`): shorter, but options shared by several components hide which ones apply, and a condition needs a helper; a props struct (`elements.ButtonProps{…}`): explicit, but verbose for the common case | decided 2026-11-22 |
| Q3 | Variants in markup | `class="ui-button" data-variant="outline" data-size="s"` | modifier classes (`ui-button ui-button--outline`) | decided 2026-11-22 |
| Q4 | Which tokens can be overridden at run time, and where from | the five accent colours, the font families and the radius scale, as cfgx Mutable settings through `themeschema`; per tenant later | every token; a theme file per deployment, with no run-time changes | decided 2026-11-22 |
| Q5 | ui's own text | ui declares i18n keys `ui.<package>.<name>` with English defaults, as errors does; a caller can pass its own text instead | callers pass every text, and ui declares nothing | default |
| Q6 | Actions (§5) | one action description, placements, and `Confirm` answered by one shared dialog | separate components per placement (zp's shape) | default |
| Q7 | Behaviour stack (ADR 0005) | native HTML, htmx and one ui script; Alpine and jQuery dropped | keep Alpine | decided 2026-11-22 |
| Q8 | Catalogue (§8) | our own `ui/catalog`, stories in `internal/stories`, `cmd/catalog` | `templ/storybook` with Node | default |
| Q9 | What the first lock (D6) covers | theme, icons, elements, the inputs the slice proves, dialog, confirmation, menu, navigation tabs, toasts, detail list, layout and catalog. Table, server table, pagination, filters, remote select and uploads wait for their own design | everything in §3 at once | default |

## 10. The slice (D4)

Four pieces that between them try every default, and the riskiest bets,
before any volume:

- **Button** (elements):
  - tokens, the builder API and variants (Q2, Q3);
  - an icon;
  - `<button>` against `<a>`, including a disabled link;
  - attribute merging;
  - the focus and RTL rules.
- **Text input on `forms.Control`** (inputs):
  - label, help and error wired by generated ids;
  - `aria-invalid` and `aria-describedby`;
  - `dir="ltr"` for email;
  - the condition evaluator on `When`, tested against the conformance
    vectors.
- **Dialog with `Confirm`** (components):
  - `commandfor` with its fallback;
  - `SlotOf[templ.Attributes]` for the trigger;
  - the shared confirmation dialog for an htmx action;
  - the close button's text from `ui.dialog.close`;
  - focus returning to the trigger.
- **Menu** (components):
  - `popover`, with anchor positioning and the script's fallback;
  - how each trigger names its anchor without an inline style;
  - `<ui-menu>` as a light-DOM custom element;
  - an action placed as a menu item;
  - a menu inside a table row swapped in by htmx.

With them come:
- the run-time theme, with contrast derivation;
- the catalogue;
- the CSS checks;
- the names and selector checks;
- the invoker-command fallback.

They land in `internal/spike/…`, and nothing is exported until epistemer
locks the interface (D6). The spike's findings amend this draft (§11).

## 11. Spike findings

From D4: the theme pipeline, Button and the catalogue (part 1); the
forms-backed field, Dialog, the shared confirmation and Menu (part 2).

1. **Shades move away from the on-colour.** The first spike mixed hover and
   active towards black in light mode and white in dark mode, in the
   browser. That dropped the on-colour's contrast to 3.56:1 on the default
   blue in dark mode, and the default green with white text started at
   only 4.38:1. Deriving the shades in Go, away from whichever of black and
   white the fill carries, holds 4.5:1 by construction. Accents therefore
   have six tokens, not four (§4).
2. **Mix in OKLab, not OKLCh.** A subtle tint of a lime brand mixed into the
   dark raised surface (`#1a1d23`, a near-grey with a little blue) moved
   from hue 127° to 245° in OKLCh, turning blue, and only to 143° in
   OKLab. OKLCh interpolates hue, and a near-grey has a real one.
3. **The token panel asks the server.** Partner colours are derived in Go,
   so the catalogue's panel sends the values to `/tokens.json` and applies
   the answer to each frame through CSSOM. It works under the catalogue's
   strict CSP (`script-src 'self'; style-src 'self'`), with no violations.
4. **Components alias tokens privately.** A tone is one block of
   `--_fill: var(--ui-color-danger)` aliases rather than a copy of every
   variant. The CSS check allows a private custom property only when its
   value is token references.
5. **The checks need tests of their own.** The first scanner skipped every
   `--_` property, because its name pattern didn't allow `_`. A test feeding
   it known-bad CSS caught that.
6. **Reduced motion lives in the token layer.** A rule in `ui.reset` loses
   to every component's transition, because a later layer wins. Every
   transition reads a duration token, so reduced motion zeroes the tokens
   instead.
7. **Builder types need a name of their own.** `elements.Button()` returns
   `elements.ButtonSpec`, because a Go function and a type can't share a
   name. D5 settles the pattern (a `Spec` suffix, or another).
8. **Markup and CSS agree.** For every Button story in LTR, RTL, light and
   dark, checked in the browser:
   - the disabled link (no `href`, `aria-disabled`, `role="link"`) is
     styled and inert;
   - the chevron mirrors only in RTL;
   - derived text colours read on both surfaces.

   In `go test`, the selector check confirmed that every `ui-*` class and
   variant attribute in the goldens has a rule in ui's CSS.
9. **Implicit anchors answer the anchoring question.** A popover opened by
   `popovertarget` takes its trigger as its implicit anchor. So
   `position-area: block-end span-inline-end` placed every menu under its
   own trigger, at the start edge in LTR and in RTL:
   - with no anchor name;
   - with no inline style;
   - in a table of menus;
   - in a row inserted after load.

   That held in the pane's Chromium. Where anchoring is missing,
   `<ui-menu>` places the popover.
10. **Custom elements keep state in private fields.** On a custom element,
    `this.popover` is the reflection of the `popover` attribute. Assigning
    the panel to it wrote `popover="[object HTMLDivElement]"`, and the
    menu never set itself up. Found in the browser, not by any check.
11. **A select needs an empty first option.** Without one, the browser
    chooses the first option when nothing is chosen, so the page disagreed
    with the server: the evaluator showed a field the server had hidden.
    `ChooseText` (`ui.inputs.choose`) fills it, and `required` makes the
    browser ask.
12. **forms' `Placeholder` falls back to the label.** ui's fields always
    show their label, so a placeholder repeating it is noise for a screen
    reader. The field renders one only on text inputs, and only when its
    text differs from the label's. It compares texts, because an
    `i18n.Ter` can be a func, which `==` can't compare.
13. **The confirmation answers on its own buttons.** Chromium dispatches a
    dialog's `close` event with rendering, which a page in the background
    holds back. Answering on the click keeps the action in the person's
    own task, and Esc still cancels through `close`. A confirmed submit
    runs once, with its original submitter.
14. **Fills see the placement's context, and it pays.**
    `components.DialogClose(ctx, value)`, written inside the caller's fill,
    names the dialog around it. The kernel carries the generated id to the
    caller's buttons with no plumbing.
15. **A control's edge needs 3:1** (WCAG 1.4.11), which a decorative
    border doesn't reach. `--ui-color-border-strong` does, tested in both
    schemes.
16. **The evaluator agrees with the server.** It passes forms' 21 vectors in
    `go test` (goja, ADR 0006), and in the browser the type select shows,
    hides and disables the dependent fields.
17. **Not yet exercised in a browser:**
    - the invoker-command fallback and the menu's own placement (the
      pane's Chromium supports both natively);
    - htmx paths (`htmx:confirm`, `htmx:afterSwap`), because the catalogue
      loads no htmx.

    They wait for the browser test job.
