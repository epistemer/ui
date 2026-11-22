# ADR 0008: an address is a link, an interface with a URL

| | |
|---|---|
| Status | Accepted; its signatures are amendment A5, locked by epistemer on 2026-11-22 |
| Date | 2026-11-22 |
| Roadmap item | D7.1, A5 |
| Deciders | epistemer decided it in the brainstorm of 2026-11-22 (S5); Claude proposed the alias, the adapter and the sanitising |

## Context

An action is shown only when the request's guard admits the route it goes
to (ADR 0009). So whatever binds an action to a route must carry the route
together with the built URL. handlers generates a URL builder per route
that returns a `string`, and finds a `Route` by its key, so an action would
name its route twice, and nothing would check that the two names agree.

ui's locked components take an address as `templ.SafeURL`. Every call that
passes a built URL converts it: `Href(templ.URL(adminroutes.Order(id)))`.

## Options

1. **A second builder per route** (`OrderLink(id)`) returning the route
   and the URL, beside the string builder. Nothing breaks. Every route
   then has two names, and a page picks between them by what the receiving
   function happens to take.
2. **Builders return a link, and ui keeps taking `templ.SafeURL`.** One
   name per route, and `.URL()` plus a conversion at every href.
3. **Builders return a link, and everything in ui that takes an address
   takes `interface{ URL() string }`.** One name per route and no
   conversion at any call. A literal address needs an adapter. Six locked
   signatures change.
4. **No change to handlers; ui matches an action's URL against the route
   table at render.** A lookup per action per row, and a URL that matches
   no route is found only when a page renders.

## Decision

Option 3.

- The kernel declares `type Link = interface{ URL() string }` and
  `func URL(s string) Link`. It is an alias so that a function another
  package declares over the bare interface is assignable where ui wants
  one over `Link`, with no import either way.
- Every ui element or helper that takes an address takes a `Link`. They
  render it through `templ.URL`, so the sanitising a `templ.SafeURL`
  argument asked the caller for now happens inside the component.
- ui does not import handlers (S7, on its default). The route never
  reaches ui; the gate reads it.

## Consequences

- handlers changes its generated builders: a breaking amendment of its
  own, for its v0.2.0 (design §10).
- Eight signatures of the locked visual interface change, the two that
  take an image's `src` among them (S8): amendment A5, locked on
  2026-11-22 (interface.md §4A, visual-interface.md §2.7 and §13.4). It
  is built before v0.2.0 is tagged, so the release doesn't carry
  signatures already replaced.
- A page writes `Href(adminroutes.Order(o.ID))` for a route and
  `Href(ui.URL("/docs"))` for a literal. An untyped string constant no
  longer converts by itself.
- A scheme outside templ's list can't be passed as trusted any more. No
  page in the inventory needs one.
- Anything with a link can be asked of the gate, not only actions: a nav
  item, a tab, a link in a cell.
