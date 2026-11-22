// Package spike holds the D7.2 slice spike (server-state-design.md §12):
// links, actions and tables for the toy shop's orders, with the column,
// value and link shapes declared locally so no other repository changes
// first. Nothing here is API; its findings amend the design.
//
//   - col: the column and plain value models would generate (S3);
//   - actions, actions/actionstest: §5;
//   - tables, tables/tablestest: §6;
//   - tableui: what ui/components would gain (§7);
//   - shop, cmd/shop: the toy shop's pages and handlers, run with
//     go run ./internal/spike/cmd/shop.
package spike
