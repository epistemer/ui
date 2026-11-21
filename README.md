# ui

neozp's UI module for Go and [templ](https://templ.guide).

- **`ui`** — the kernel: typed context values (`Provider`), element ids
  that never collide across fragments (`ID`), and named slots for inline
  content (`Slot`, `SlotOf`). Imports templ and the standard library only.
- **`ui/uitest`** — golden HTML and id-reference checks for components.
- **`ui/forms`** — declare, bind, validate and describe HTML forms: one
  typed `Field[T]` per field, typed rules (`ui/forms/rule`), cross-field
  conditions shared with the client, groups and lists. Imports no templ.

Status: pre-release. The design is in [docs/brief.md](docs/brief.md) and
[docs/forms-design.md](docs/forms-design.md); the contract is
[docs/interface.md](docs/interface.md); progress is in
[docs/roadmap.md](docs/roadmap.md).

Requires Go 1.27.

## License

MIT
