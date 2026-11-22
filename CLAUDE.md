# ui

neozp's UI module. Its first packages are the kernel (root package `ui`:
providers, element IDs, slots; test kit `ui/uitest`) and forms (`ui/forms`,
`ui/forms/rule`, `ui/forms/formstest`: declare, bind, validate and describe
HTML forms). Later packages (icons, elements, inputs, components, layout) join
this module.

- Read [docs/roadmap.md](docs/roadmap.md), then the newest entry of
  [docs/status.md](docs/status.md). The designs are
  [docs/brief.md](docs/brief.md) (kernel),
  [docs/forms-design.md](docs/forms-design.md) (forms) and
  [docs/visual-design.md](docs/visual-design.md) (the visual packages, a
  draft; its ADRs are 0004 and 0005); the contract is
  [docs/interface.md](docs/interface.md), locked by epistemer. The
  visual packages' contract is drafted in
  [docs/visual-interface.md](docs/visual-interface.md) until epistemer
  locks it (D6).
- Links, actions and tables (D7) are designed in
  [docs/server-state-design.md](docs/server-state-design.md), a draft with
  ADRs 0008–0010. Its amendment A5 (links: `ui.Link`, `ui.URL`, eight
  signatures) is locked, in both contracts, and built (2026-11-22).
- The contract of actions and tables is
  [docs/server-state-interface.md](docs/server-state-interface.md),
  locked by epistemer on 2026-11-22 as amendment A6; milestones T1–T4
  build it.
- The remote select, uploads and list fields (D8) are designed in
  [docs/rich-inputs-design.md](docs/rich-inputs-design.md), a draft with
  ADRs 0011–0013. Its contract,
  [docs/rich-inputs-interface.md](docs/rich-inputs-interface.md), was
  locked by epistemer on 2026-11-22 as amendment A7; milestones U1–U5
  build it.
- `go run ./cmd/catalog` serves the stories and the toy shop's tables
  and actions (`internal/toyshop`) at /shop/orders.
- Claude owns this repository until v1.0.0, on the terms of
  [docs/ownership.md](docs/ownership.md) (delegated 2026-11-21). Only API
  design is epistemer's, and the lock of interface.md.
- Load the `i18n`, `errors` and `models` skills before touching labels,
  errors or options.
- The kernel imports templ and the standard library only. `ui/forms` imports
  no templ, no web framework and no decimal library; its public API has no
  `any`. Invariants K1–K8 (brief) and I1–I10 (forms design) are fixed.
- zp (`/home/epistemer/zp`) is private and read-only. Keep its names out of
  this repository, which is public: examples and tests use a toy shop. zp's
  own issues are out of scope, and no release waits on moving zp.
- Every working session adds an entry to docs/status.md.
- Generate templ from the repository root, with the version in go.mod:
  `go run github.com/a-h/templ/cmd/templ@v0.3.1118 generate`. Never use
  `-path`, because generated files record file names relative to where
  templ runs, and CI's drift check runs from the root. templ also skips a
  generated file that is newer than its source, so after a `-path` run,
  touch the `.templ` files before generating again.
