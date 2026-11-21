# ADR 0004: ui ships its own CSS, written in modern CSS

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Roadmap item | D2 |
| Deciders | epistemer decided to ship ui's own CSS with no Tailwind or Flowbite (2026-11-21); Claude, as owner, proposed how it is organised |

## Context

The visual packages (icons, elements, inputs, components, layout) need
styles. zp's show what to avoid ([ui-inventory.md](../ui-inventory.md)):
- colours are compiled into a Tailwind config, so nothing can be themed while
  the app runs;
- raw palette classes outnumber semantic ones five to one;
- dark mode is a second copy of every colour;
- variants are copied class strings;
- every style change goes through a Node build.

The constraints:
- ui is a Go module, and apps must not need Node to use it;
- an app must be able to override any style without a fight over
  specificity;
- tokens must change while the app runs (white-labelling, as i18n does for
  text);
- pages run in RTL as well as LTR;
- dark mode is required.

## Options

1. **Tailwind classes in templates; the app's Tailwind build scans ui's
   source.** This is the familiar route. The cost is that the app's build has
   to find ui's templates in the module cache. A class passed by the user
   doesn't override the component's own (`px-4` against `px-5`): which one
   wins depends on the stylesheet's order, not on the attribute. Theming
   still needs custom properties underneath.
2. **Our own stylesheet in cascade layers, built on custom properties.** We
   write and maintain every rule. In return, apps need no build, overrides
   are predictable, and tokens change while the app runs.
3. **A third-party CSS framework.** This trades Tailwind's problems for
   another framework's naming and release cycle, and it still needs adapting
   for tokens and RTL.

## Decision

Option 2, organised as follows. The design review of 2026-11-22
(docs/reviews/0002-visual-design.md) corrected the first draft of this
section.

- **Cascade layers mirror the package layers:**
  `@layer ui.reset, ui.tokens, ui.base, ui.elements, ui.inputs, ui.components, ui.layout;`.
  - A component's rules beat an element's.
  - Any style an app writes outside a layer beats all of ui's.
  - An app that layers its own CSS puts ui after its resets and before its
    components and utilities. For Tailwind v4 that is
    `@layer theme, base, ui, components, utilities;`. If ui came first,
    Tailwind's Preflight (in `base`) would beat every ui rule, since a later
    layer wins regardless of specificity.
- **`!important` appears once:** `[hidden] { display: none !important; }` in
  `ui.reset`. Without it, any component's `display` would show an element
  that HTML (or an inactive forms control) hides. An important declaration
  in the lowest layer beats everything, which is what `hidden` needs.
  Nothing else in ui uses `!important`.
- **Tokens are custom properties named `--ui-*`**, defined in `ui.tokens`
  on `:root`. Component rules read only tokens. A check rejects a literal
  colour, length or shadow in a declaration value outside the token layer.
  - The check allows a short list: `0`, percentages, `fr`, `1px` hairlines,
    `transparent`, `currentColor`, `inherit`.
  - It skips `@media` and `@container` conditions, where `var()` isn't
    allowed.

  Derived shades use `color-mix(in oklch, …)` on the token, so one brand
  colour changed at run time re-themes all its shades.
- **Dark mode:**
  - `:root` declares `color-scheme: light dark`, and colour tokens use
    `light-dark()`, so each colour is written once.
  - The page follows the operating system unless the server renders
    `<html data-color-scheme="light">` or `"dark"` from the user's choice.
    That is a cookie, as in zp, so no page flashes the wrong theme.
  - `ui.tokens` maps the attribute to `color-scheme`. It is an attribute,
    not an inline style, because `style-src 'self'` would block an inline
    style.
- **RTL:**
  - Logical properties only (`margin-inline-start`, `inset-inline-end`,
    `text-align: start`).
  - The check rejects physical properties. It also rejects the values that
    hide physical sides: four-value `margin`, `padding` and `inset` with
    different left and right parts; multi-value `border-radius`;
    `translateX` outside a `:dir()` rule; and `to left` or `to right` in
    gradients.
  - Icons that point a direction mirror under `:dir(rtl)`.
  - On the HTML side, the layout sets `dir` on `<html>` from the locale, and
    inputs that hold left-to-right text (email, URL, phone) render
    `dir="ltr"`.
- **Focus:** rings use `:focus-visible`, never `:focus`, with a token for
  the ring.
- **State is styled from HTML and ARIA, not from classes:**
  - The selectors: `:disabled`, `[aria-disabled="true"]`, `[aria-current]`,
    `:user-invalid`, `[aria-invalid="true"]`, `:popover-open`, `[open]`,
    and `[aria-expanded="true"]`.
  - The markup that people with assistive technology rely on is then the
    same markup that shows the state.
  - The server never renders a state the browser changes later. For
    example, a menu trigger's `aria-expanded` is kept in step by the ui
    script on the popover's `toggle` event (ADR 0005), not rendered as a
    fixed value.
- **Modern CSS is used freely where it is Baseline widely available** as of
  2026-11:
  - nesting;
  - `:has()`;
  - container queries;
  - `color-mix()` and oklch;
  - `light-dark()`;
  - logical properties;
  - `:focus-visible`;
  - `<dialog>`;
  - `popover`.

  Features that are only newly available get a fallback that keeps the page
  fully usable:
  - **Anchor positioning** (Baseline January 2026). An open popover sits in
    the top layer, positioned against the viewport, so CSS alone can't
    place it next to its trigger. Where anchor positioning is missing, the
    ui script positions it through CSSOM, which a strict CSP allows. How
    each trigger names its anchor without an inline style is tried in the
    D4 spike.
  - **Invoker commands** (`commandfor`, Baseline December 2025): a few lines
    in the ui script (ADR 0005).

  Features outside Baseline (`contrast-color()`, customizable `<select>`,
  `popover="hint"`) only add polish, and nothing depends on them.
- **Delivery:**
  - All of ui's stylesheets are plain `.css` files in `theme/css/`, one per
    layer, embedded by `ui/theme`. Go's `embed` can't reach into sibling
    packages, and the visual packages emit class names without importing
    the package that serves the CSS.
  - When the program starts, `ui/theme` renders the token layer from its
    token table, joins the files in layer order, and serves the result with
    the ui script under one content-hashed directory (`…/<hash>/ui.css`,
    `…/<hash>/ui.js`), cached as immutable.
  - ui has no CSS build step and no generated CSS file.
- **Class names:** one class per component, prefixed `ui-`. Variants are
  given as attributes (interface to be designed: visual-design.md, Q3).
  Class names and variant attributes are public API, because apps style
  them.

## Consequences

**Becomes easier:**
- Apps need no Node, and ui's CSS stays small: it holds only the components,
  not a utility catalogue.
- Overrides are predictable: an app's styles beat ui's, outside a layer or in
  any layer the app orders after `ui`.
- Theming while the app runs, white-labelling and dark mode come from one
  mechanism.
- RTL and focus style are enforced by checks, not by review.

**Costs:**
- We maintain every rule ourselves.
- The two checks (literal values, physical sides) need a small CSS scanner
  of our own in a test. Neither proves a page looks right: they rule out
  the two mistakes zp's templates show most. Screenshots come with the
  browser test job.
- The fallbacks for anchor positioning and invoker commands stay until both
  are Baseline widely available, about mid-2028.

**Changes elsewhere:**
- visual-design.md's tokens and catalogue sections follow from this ADR.
- The roadmap gains the design track D1–D7.
- Emails and PDFs are out of scope: they need inline styles and are a
  separate concern.
