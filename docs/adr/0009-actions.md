# ADR 0009: actions are a logic layer, gated by their route

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Roadmap item | D7.1 |
| Deciders | epistemer proposed the logic layer and decided where the permission comes from (S1) in the brainstorm of 2026-11-22; Claude proposed the rest |

## Context

The first lock of the visual packages has no `Action` type
([visual-interface.md](../visual-interface.md) §7.2, V6). The spike had
found that a confirmation needs none: it is an attribute on whatever
starts the action. "Described once, placed anywhere" was left for D7.

What changed is permissions. The record of zp
([server-state-design.md](../server-state-design.md) §1) shows about 100
action call sites over 13 components, none of which knows who is asking:
every user saw every button. It also shows that a state rule written once
and called from both the row and the service was the only kind that never
disagreed with itself.

Once whether an action shows depends on the request, there is logic, and
visual-design §2.4 says logic splits from the view where there is logic.
epistemer proposed giving actions the modularity forms has: a layer that
knows nothing of the UI and may yield no action at all.

## Options

For the layer:

1. **No type; a page writes `if` around each button.** What zp did. Every
   page repeats the check, and nothing can test a page's actions without
   rendering it.
2. **A logic package, `ui/actions`:** declare an action once, bind it to a
   subject's link, resolve the set against the request, and describe each
   as an item a placement renders.

For where the permission comes from:

a. **Declared on the action** (`Needs(permission)`). The permission is
   then written on the route and again on the action, and the two can
   drift apart, which is zp's duplicated state rule in a new place.
b. **From the action's route.** The link an action is bound to belongs to
   a route, and the route's guard is asked. Shown and allowed are one
   fact.
c. **A flag the page computes.** Option 1 again.

## Decision

Option 2 with b.

- `ui/actions` declares, binds, resolves and describes
  (design §5.1). It renders nothing; it imports templ only for the types
  of an icon and of extra attributes.
- `Resolve` asks a gate, a function over a link that the app sets once
  per request from its routes' guard. No gate set panics.
- An action the gate refuses is absent. An action the record's state
  forbids is disabled with a reason, through `Unless(err)`, where the rule
  is a function the handler also calls and its errors declaration is the
  reason shown.
- How an action runs follows from its link's method and from whether it
  opens the action dialog. The caller never picks a mechanism.
- A handler answers through one builder with a small vocabulary: replace,
  remove, refresh, go, toast (design §5.6).
- Confirmation stays the locked attribute. V6's finding stands; this ADR
  adds the type V6 postponed.

## Consequences

- A page's actions can be tested without HTML (`actionstest`).
- The 13 components of the record become two placements and a bulk bar,
  all reading the same items.
- An app must set the gate, one middleware. Forgetting it fails loudly.
- The gate can't know a route's freshness requirement, so an action that
  needs a recent proof shows, and the route answers with auth's error.
- The layer is thinner than forms: no codec and no validation. It must
  stay out of request details, which remain attributes, or it becomes the
  record's request-settings struct again.
- ui emits `hx-*` on action controls and in `Reply`. ADR 0005 allows it
  for components that make requests; actions join that list.
- The contract is written after the slice spike (D7.2) and locked as an
  amendment (D7.5).
