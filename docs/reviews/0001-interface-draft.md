# Review 0001: interface.md draft (M0.2)

| | |
|---|---|
| Date | 2026-11-21 |
| Reviewer | Gemini 4 Neon (High), through agy (`adversarial_review`), about 71k tokens. Argon was in quota cooldown; per the owner's rules the strongest available model ran instead of waiting |
| Under review | docs/interface.md (draft) with excerpts of brief §2, §7.1 and forms-design §3 |
| Scope | Contradictions with the invariants, signatures Go can't express, undefined behaviour, key collisions, `any` leaks, silent misuse |

18 findings (8 ranked critical, 9 major, 1 group of minor). All were checked
by Claude against the draft and, where they made a claim about Go, by a
compiler spike on Go 1.27.4.

| # | Finding | Verdict | Resolution |
|---|---|---|---|
| 1 | rule's import list omits i18n, which §6.3 needs | Valid | §1 adds i18n to rule |
| 2 | Rules before `As` vs conversion in `Bind` | Valid | §9.2 Conversion stages: each stage's rules run on its value up to the stage that failed; first rule failure wins |
| 3 | Checkbox/Bool wire values; `Filled` on a checkbox | Valid | §9.2 Booleans |
| 4 | `Eq` on slices and files; server formatted value vs client text | Valid | `Eq`/`In` on slices and files panic at declaration; §5.7 compares the bound text, as a client does |
| 5 | `AssertLabelled` not implementable from the public API | Valid | §9.5: reads an internal package of forms; scope defined |
| 6 | Sub-form lifecycle, group checks, "no field failed" | Valid | §5.1 "any node"; §9.2 sub-forms and group checks |
| 7 | `FromRequest` query merging, multipart cleanup | Valid | §9.3 |
| 8 | `rule.MinItems(1)` cannot infer `T` | Valid (confirmed: "cannot infer T") | §6.2 documents `rule.MinItems[string](1)` |
| 9 | `Parsed` rejects pointer-receiver marshallers (`big.Int`) | Valid (confirmed) | §5.3 constraint becomes `T any, PT interface{ *T; TextMarshaler; TextUnmarshaler }`. A signature change before the lock, strictly more permissive |
| 10 | Id validity: prefix, tokens, zero `ID`, nil source | Valid | §9.1 |
| 11 | Slot messages without a slot, white space, fallback `Value`, bad `Slotted` input | Valid | §9.1 |
| 12 | List index syntax, ghost items, `Add` vs `Bind`, limits | Valid | §9.2 Lists |
| 13 | `Options` timing and `Enum` edge cases | Valid | §9.2 Options |
| 14 | `ContentType` trust, `URL` schemes, anchoring, NaN, dates, unspecified predicates, size units | Valid | §9.4, §9.2 Numbers |
| 15 | `Active` before Bind, `Texts` kind, `Attrs` duplicates, traversal | Valid | §9.2, §9.3 |
| 16 | Multiple values, trimming, empty file parts, `Set`/`Validate` order | Valid | §9.2 |
| 17 | `-update` flag collision, fragment parsing drops rows, ID lists | Valid | §8, §9.5 |
| 18 | Silent misuse: bare `Def1`, `Raw` on non-text, `As` carry-over, nils, empty `All`/`Any`, form-level error text, formstest imports, separators, zero-size `Provider` | Valid | §9.1–§9.5, §1 |

**Left for epistemer at the lock.** The `Parsed` constraint (finding 9) is a
signature change; the explicit type argument on `MinItems`/`MaxItems`
(finding 8) is a cost of the approved shape, not a change.
