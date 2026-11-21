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
  [docs/interface.md](docs/interface.md), locked by epistemer.
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
