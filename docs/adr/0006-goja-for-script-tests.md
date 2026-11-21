# ADR 0006: goja runs the ui script's tests, in a module of its own

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Roadmap item | D4 |
| Deciders | Claude, as owner (a dependency an ADR justifies) |

## Context

ADR 0005 puts the forms condition evaluator in the ui script, and it must
pass the conformance vectors in `forms/testdata/cond` that forms' Go
evaluator passes. ui wants no Node toolchain, so the vectors should run
in `go test`. A test-only import still enters ui's go.mod, where every app
that requires ui sees it in its module graph.

## Options

1. **goja** (`github.com/dop251/goja`): an ECMAScript engine in pure Go,
   maintained, used widely. It runs ES2017 and most later syntax. It doesn't
   load ES modules, so the test strips the `export` keyword from the module
   it reads.
2. **A headless browser** driven from Go (chromedp). It is the real
   runtime, but needs Chrome on every CI runner, and it is far heavier than
   a pure function needs.
3. **Node in CI only.** It brings back the toolchain ui avoids.

## Decision

Option 1, in a nested module `internal/jstest` with its own go.mod. ui's
go.mod never sees goja. CI runs the module in each of its per-module steps
(build, vet, staticcheck, tidy, race tests). Modules that are pure functions
of their input (the evaluator) are tested this way. Behaviour that needs a
DOM waits for the browser job ADR 0005 leaves open.

## Consequences

- The evaluator's tests run in `go test`, beside the Go evaluator's, on the
  same vector files. A change to the vectors runs both.
- The evaluator is written in plain ES2017, with no optional chaining and
  no `Array.prototype.at`, so goja runs it unchanged apart from the export.
- goja has no tagged releases; go.mod pins a pseudo-version, which tidy
  updates only when asked.
