# ADR 0005: client behaviour from native HTML, htmx and one small ui script

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Roadmap item | D2 |
| Deciders | Claude proposed; epistemer accepted it on 2026-11-22 (Q7) |

## Context

zp's pages run on three client runtimes and loads all of them on every page
([ui-inventory.md](../ui-inventory.md)):
- htmx, with its SSE extension;
- Alpine, in 16 components;
- 21 of zp's own scripts, about 2,250 lines;
- jQuery 3.2.1 slim, with a flash-message library.

With ADR 0004, Flowbite is gone. What the visual packages still need on the
client:
- the forms condition evaluator, which must pass `forms/testdata/cond`;
- keyboard handling that native HTML lacks (arrow keys in a menu, a
  combobox);
- cloning a list's template row;
- toasts;
- a password visibility toggle;
- a file preview;
- writing the theme cookie;
- fallbacks for invoker commands.

ui must not need Node, and its markup should work under a strict Content
Security Policy.

## Options

1. **Keep Alpine for behaviour, htmx for requests.** This is familiar from
   zp. But behaviour lives in strings inside attributes that no test
   compiles. Alpine needs `unsafe-eval` unless its CSP build is used, which
   drops most of its expressions. And every page loads a framework.
2. **Native HTML first, htmx for requests, and one small ui script for the
   rest.**
   - The browser does what it now can on its own: `dialog`; `popover` with
     light dismiss and Esc; `commandfor` and `command`; `details`; `inert`;
     form validation states.
   - htmx does requests, fragment swaps and streamed rows, which ui's
     components already address by id.
   - The ui script holds the rest: plain ES2022 modules with no framework,
     embedded and served like the CSS.
3. **Web components for everything,** with shadow DOM. Encapsulation would
   block ADR 0004's cascade (an app's styles could no longer reach inside),
   and server rendering of shadow DOM is still awkward.

## Decision

Option 2. The design review of 2026-11-22
(docs/reviews/0002-visual-design.md) corrected the first draft of this
section.

- **The ui script** has a size budget (proposed: 12 kB gzipped for all of
  it), measured in CI. There is no bundler and no minifier: the files are
  served as written, and the browser loads them as modules.
  - They are served from the same content-hashed directory as the CSS
    (ADR 0004), so a relative `import "./cond.js"` resolves to the same
    release.
  - There is no import map, which would need an inline script.
- **Stateless behaviours** are one delegated listener on `document`, keyed
  by `data-ui-*` attributes. Content that htmx swaps in works with no setup
  code. They cover:
  - the invoker-command fallback, which returns early where the browser
    supports `commandfor` (so nothing fires twice) and calls
    `preventDefault`;
  - the password toggle;
  - the theme cookie;
  - Esc to dismiss a tooltip (WCAG 1.4.13).

  Every `<button>` ui renders states its `type`, so a trigger never submits
  a form.
- **Stateful behaviours** are custom elements in light DOM, with no shadow
  DOM, so ADR 0004's styles apply. `connectedCallback` sets each one up
  when htmx swaps it in, and everything still renders on the server.
  - **`<ui-menu>`** moves focus to the first item on open and handles arrow
    keys. It keeps the trigger's `aria-expanded` in step with the popover's
    `toggle` event, and positions the popover where anchor positioning is
    missing.
  - **`<ui-combobox>`** is the remote select.
  - **`<ui-repeat>`** clones a list's template row. It renumbers `__i__` in
    names and gives every id in the clone a fresh suffix, rewriting the
    references to it (`for`, `aria-*`, `commandfor`) so clones never share
    an id.
  - **`<ui-toasts>`** is a live region that stays in the layout shell. Toasts
    arrive inside it through an out-of-band `beforeend` swap, and pause
    their timeout on hover and focus.
- **Confirmation** uses one shared `<dialog>` in the layout shell, not one
  dialog per action.
  - For htmx actions, the script answers htmx's `htmx:confirm` event.
  - For links and forms, it answers `data-ui-confirm`.
  - The question text is rendered on the server, translated, into the
    action's attribute.

  A dialog never sits inside a menu's popover or inside the form it
  confirms.
- **Alpine and jQuery are dropped.** No behaviour lives in attribute
  strings. ui emits no inline script, no `hx-on` and no `javascript:` URLs,
  so its markup works under `script-src 'self'`.
- **htmx** is the app's dependency, not ui's. ui emits `hx-*` attributes
  only where a component exists to make requests: the server table, the
  remote select, the command palette. Each of those documents the htmx
  version it targets.
  - One exception, locked with the visual packages (V18, 2026-11-22):
    `components.ToastSwap` wraps toasts in `hx-swap-oob="beforeend:#ui-toasts"`.
    A toast isn't a request; it's how a response answers, and htmx 2
    appends it to the region out of band.
  - htmx injects an inline style for its loading indicators, which
    `style-src 'self'` blocks. Apps set `includeIndicatorStyles: false`,
    and `ui.base` styles `.htmx-indicator` instead.
- **Tests:**
  - The condition evaluator runs against the conformance vectors inside
    `go test`, through a pure-Go JavaScript engine, so no Node is needed.
    The test lives in a nested module of its own, so the engine never
    enters ui's go.mod. It gets its own ADR when it lands (D4).
  - Behaviours that need a real DOM are tested in a browser, in a separate
    CI job whose tooling is decided with the first component that needs it.

## Consequences

**Becomes easier:**
- A page loads one ui script instead of three runtimes and a jQuery.
- Behaviour is ordinary code, so it can be reviewed and tested.
- Strict CSP is possible.
- Most dialogs, popovers and disclosures need no script at all.

**Costs:**
- We write keyboard handling for menus and the combobox ourselves. The ARIA
  Authoring Practices patterns are the specification, and each component
  documents the one it follows.
- Browser tests need tooling that isn't chosen yet.

**Changes elsewhere:**
- forms' client evaluator, which forms-design §7.3 expected to move to
  ui/inputs, becomes a module of the ui script.
- The roadmap's D4 slice includes it.
