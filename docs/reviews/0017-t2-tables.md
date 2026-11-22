# Review 0017: T2, tables

| | |
|---|---|
| Date | 2026-11-22 |
| Reviewer | Gemini 4 Argon (Medium), through agy (`adversarial_review`), about 35k tokens |
| Under review | `ui/tables` and `tablestest` without tests, with server-state-interface.md §3 |
| Scope | The code against the contract: URLs, the listing query, paging, the selection, sealing |

There were 10 findings. Claude checked each; all ten changed the code.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | An empty text key becomes "0" | Valid | The key is read by its column's kind |
| 2 | A huge `page` overflows the offset and the pager's window | Valid | The page is capped; the pager multiplies in int64 |
| 3 | Chips show raw query values, also for a rejected state | Valid | Chips read the bound values (an option's label, a bool's word, a date as `Date` renders it, a number), and a rejected state shows none |
| 4 | Hidden inputs keep empty values that links drop | Valid | The query is cleaned once: empty values dropped |
| 5 | Junk in `page`, `sort` and `size` travels on in links | Valid | The query links are built from is normalised at `Bind`: sort, size and page as bound |
| 6 | `New` keeps the caller's slice | Valid | Cloned |
| 7 | A window between sealing and checking | Valid | Sealed first, unsealed if `Check` refuses (it panics anyway) |
| 8 | `Selected` returns nothing for a body that doesn't parse | Valid as stated; it is what the contract asks (the body for a POST), said so | Kept, documented |
| 9 | `View.Selection` isn't in the locked struct | Valid | Removed; the rendering takes the name from the table |
| 10 | A negative total gives a negative range | Valid | Treated as 0 |
