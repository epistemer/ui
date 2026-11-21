# ADR 0003: golang.org/x/net/html for uitest

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-21 |
| Roadmap item | M3.2 |
| Deciders | Claude, as owner (a dependency an ADR justifies) |

## Context

uitest normalises HTML before comparing it with a golden file (one
element per line, attributes sorted, text trimmed) and walks it to check
id references (interface §8, §9.5). Both need a parser that follows the
HTML5 tree-construction rules, including fragments in context, so that a
table row or an option rendered alone is not dropped.

## Options

1. **golang.org/x/net/html.** The Go team's HTML5 parser; `ParseFragment`
   takes a context element; already in most Go web apps' module graphs.
2. **A hand-written tokenizer.** No dependency, but HTML5's implied tags,
   raw-text elements and fragment contexts are exactly what a small
   tokenizer gets wrong.
3. **A third-party DOM library** (goquery and similar). Brings more than
   needed, and is built on option 1 anyway.

## Decision

Option 1, imported by `ui/uitest` only. The kernel and forms never import
it, so apps that don't import uitest from tests don't link it.

## Consequences

- The module's go.mod requires golang.org/x/net. `go list -deps .` of the
  kernel still shows only templ and the standard library; a test checks it.
- Fragment parsing picks its context from the first element (§9.5).
