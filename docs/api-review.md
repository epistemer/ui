# API review for v0.1.0

R.1 of the roadmap, 2026-11-21, by Claude. Every exported identifier of the
five packages, listed by `go doc -all`, checked three ways:

1. **Present in the contract.** Its name appears in docs/interface.md; the
   section is the first one that names it. All 228 do.
2. **Same signature.** Every `func` line in interface.md's code blocks was
   compared with `go doc` after normalising receiver names: all match, apart
   from two parameter names (`Help(h)` is `Help(t)` in code, since `h` is the
   receiver; uitest's `html` is `rawHTML`, since `html` is an imported
   package) and line wrapping. The rule constructors, which §6.2 gives as a
   table, match its types and attributes.
3. **Exercised by tests.** The test files (including examples/shop) that
   name it. Error variables are checked by key, kind and text in
   TestDefaultErrors (rule) and the forms tests.

Findings: none open. Behaviour the interface left open was settled during
the milestones and recorded as amendments A1 and A2 (§10).

## `ui` (§2–§4)

| Identifier | Interface | Tests |
|---|---|---|
| `ErrFilledTwice` | §4.2 | slot_test.go |
| `ErrInBlock` | §2 | slot_test.go |
| `ErrNoComponent` | §4.2 | slot_test.go |
| `ErrNotDeclared` | §4.2 | slot_test.go |
| `ErrStrayContent` | §4.1 | slot_test.go |
| `ID` | §3 | components_test.go, id_test.go, ids_test.go, stores_test.go |
| `ID.Get` | §3 | cond_test.go, conv_test.go, example_test.go, forms_test.go … |
| `ID.Key` | §3 | conv_test.go, example_test.go, forms_test.go, helpers_test.go … |
| `ID.Part` | §3 | example_test.go, id_fuzz_test.go, id_test.go |
| `IDSource` | §3 | through its values and methods |
| `NewID` | §3 | example_test.go, id_fuzz_test.go, id_test.go |
| `NewProvider` | §2 | example_test.go, provider_test.go |
| `NewSlot` | §4 | example_test.go |
| `NewSlotOf` | §4 | slot_test.go |
| `Provider` | §2 | through its values and methods |
| `Provider.Provide` | §2 | provider_test.go, slot_test.go |
| `Provider.Value` | §2 | api_test.go, cond_test.go, conv_test.go, example_test.go … |
| `Provider.With` | §2 | example_test.go, golden_test.go, provider_test.go, render_test.go … |
| `Scope` | §3 | example_test.go, id_fuzz_test.go, id_test.go, slot_test.go |
| `Sequence` | §3 | example_test.go, id_fuzz_test.go, id_test.go, render_test.go |
| `Slot` | §4 | slot_test.go |
| `Slot.Filled` | §4 | cond_test.go, example_test.go, nesting_test.go, slot_test.go |
| `Slot.Place` | §4 | example_test.go, slot_test.go |
| `Slot.Render` | §4 | components_test.go, contract_test.go, example_test.go, golden_test.go … |
| `SlotOf` | §4 | slot_test.go |
| `SlotOf.Filled` | §4 | cond_test.go, example_test.go, nesting_test.go, slot_test.go |
| `SlotOf.Place` | §4 | example_test.go, slot_test.go |
| `SlotOf.Render` | §4 | components_test.go, contract_test.go, example_test.go, golden_test.go … |
| `SlotOf.Value` | §2 | api_test.go, cond_test.go, conv_test.go, example_test.go … |
| `SlotRef` | §4 | through its values and methods |
| `Slotted` | §4 | example_test.go, slot_test.go |
| `WithIDs` | §3 | example_test.go, id_fuzz_test.go, id_test.go, render_test.go |

## `ui/uitest` (§8)

| Identifier | Interface | Tests |
|---|---|---|
| `CheckIDs` | §8 | components_test.go, example_test.go, ids_test.go |
| `Context` | §2 | cond_test.go, contract_test.go, conv_test.go, example_test.go … |
| `External` | §8 | ids_test.go |
| `Golden` | §8 | components_test.go, golden_test.go |
| `Option` | §5.3 | render_test.go |
| `RefOption` | §8 | through its values and methods |
| `Render` | §4 | components_test.go, contract_test.go, example_test.go, golden_test.go … |
| `RenderErr` | §8 | example_test.go, render_test.go, slot_test.go |

## `ui/forms` (§5)

| Identifier | Interface | Tests |
|---|---|---|
| `All` | §5.4 | cond_test.go, forms_test.go, review_test.go, stores_test.go |
| `Any` | §5.7 | cond_test.go |
| `Choice` | §5.10 | api_test.go |
| `Cond` | §5.1 | cond_test.go |
| `Cond.MarshalJSON` | §5.7 | cond_test.go |
| `Control` | §5.1 | api_test.go, conv_test.go, form_test.go, nesting_test.go |
| `Control.Active` | §5.1 | form_test.go, nesting_test.go |
| `Control.Attrs` | §5.10 | file_test.go, nesting_test.go, order_test.go, rule_test.go … |
| `Control.Children` | §5.10 | form_test.go, nesting_test.go |
| `Control.Err` | §2 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Control.Help` | §5.4 | review_test.go |
| `Control.Kind` | §5.4 | api_test.go, nesting_test.go, review_test.go, stores_test.go |
| `Control.Label` | §5.1 | api_test.go, forms_test.go, nesting_test.go |
| `Control.Options` | §5.4 | api_test.go, nesting_test.go, review_test.go |
| `Control.Path` | §5.8 | api_test.go, form_test.go, forms_test.go, nesting_test.go |
| `Control.Placeholder` | §5.4 | review_test.go |
| `Control.Required` | §5.3 | cond_test.go, conv_test.go, forms_test.go, nesting_test.go … |
| `Control.Template` | §5.10 | form_test.go, nesting_test.go |
| `Control.Text` | §5.3 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Control.When` | §5.1 | cond_test.go, form_test.go, nesting_test.go |
| `Conv` | §5.4 | conv_test.go |
| `DefaultMaxItems` | §5.8 | api_test.go, nesting_test.go |
| `EnumValue` | §5.3 | api_test.go |
| `ErrInvalid` | §5.9 | conv_test.go, forms_test.go, nesting_test.go, review_test.go |
| `ErrMaxItems` | §5.8 | nesting_test.go, rule_test.go, slices_test.go |
| `ErrMinItems` | §5.9 | forms_test.go, nesting_test.go, rule_test.go, slices_test.go |
| `ErrNotAnOption` | §5.4 | forms_test.go, review_test.go, stores_test.go |
| `ErrRequired` | §5.9 | cond_test.go, conv_test.go, forms_test.go, review_test.go … |
| `ErrUnreadable` | §5.1 | forms_test.go |
| `Field` | §5.1 | cond_test.go, conv_test.go, forms_test.go, golden_test.go … |
| `Field.As` | §5.1 | api_test.go, cond_test.go, conv_test.go, nesting_test.go |
| `Field.Control` | §5.1 | api_test.go, conv_test.go, form_test.go, nesting_test.go |
| `Field.Eq` | §5.3 | cond_test.go, nesting_test.go |
| `Field.Err` | §2 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Field.Fail` | §5.1 | forms_test.go, golden_test.go, nesting_test.go, review_test.go |
| `Field.Filled` | §4 | cond_test.go, example_test.go, nesting_test.go, slot_test.go |
| `Field.Get` | §3 | cond_test.go, conv_test.go, example_test.go, forms_test.go … |
| `Field.Help` | §5.4 | review_test.go |
| `Field.Hint` | §5.4 | api_test.go, id_test.go, review_test.go |
| `Field.In` | §2 | cond_test.go, example_test.go |
| `Field.InvalidWith` | §5.4 | forms_test.go, review_test.go |
| `Field.Is` | §4.2 | api_test.go, cond_test.go, contract_test.go, conv_test.go … |
| `Field.Label` | §5.1 | api_test.go, forms_test.go, nesting_test.go |
| `Field.Options` | §5.4 | api_test.go, nesting_test.go, review_test.go |
| `Field.Placeholder` | §5.4 | review_test.go |
| `Field.Ptr` | §5.5 | form_test.go, forms_test.go, nesting_test.go, stores_test.go |
| `Field.Raw` | §5.3 | api_test.go, forms_test.go, review_test.go |
| `Field.Required` | §5.3 | cond_test.go, conv_test.go, forms_test.go, nesting_test.go … |
| `Field.RequiredWith` | §5.4 | forms_test.go, review_test.go |
| `Field.Set` | §5.5 | conv_test.go, forms_test.go, nesting_test.go, review_test.go … |
| `Field.SetPtr` | §5.5 | forms_test.go, review_test.go |
| `Field.Value` | §2 | api_test.go, cond_test.go, conv_test.go, example_test.go … |
| `Field.When` | §5.1 | cond_test.go, form_test.go, nesting_test.go |
| `FieldError` | §5.1 | api_test.go |
| `FieldError.Error` | §4.2 | api_test.go, cond_test.go, conv_test.go, file_test.go … |
| `FieldError.Unwrap` | §5.9 | api_test.go |
| `Float` | §5.3 | api_test.go, forms_test.go, review_test.go |
| `Form` | §5 | api_test.go, cond_test.go, form_test.go, forms_test.go … |
| `Form.Active` | §5.1 | form_test.go, nesting_test.go |
| `Form.Bind` | §5.1 | cond_test.go, conv_test.go, forms_test.go, money_test.go … |
| `Form.Bool` | §5.3 | api_test.go, forms_test.go, review_test.go |
| `Form.Check` | §5.1 | file_test.go, forms_test.go, nesting_test.go, order_test.go … |
| `Form.Checkbox` | §5.3 | api_test.go, cond_test.go, forms_test.go, nesting_test.go … |
| `Form.Control` | §5.1 | api_test.go, conv_test.go, form_test.go, nesting_test.go |
| `Form.Controls` | §5.1 | nesting_test.go |
| `Form.Date` | §5.3 | api_test.go, forms_test.go, review_test.go, time_test.go |
| `Form.Enum` | §5.3 | api_test.go, cond_test.go, forms_test.go, nesting_test.go … |
| `Form.Err` | §2 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Form.Errors` | §4.2 | form_test.go, forms_test.go, nesting_test.go, stores_test.go |
| `Form.Fail` | §5.1 | forms_test.go, golden_test.go, nesting_test.go, review_test.go |
| `Form.File` | §5.2 | api_test.go, cond_test.go, file_test.go, forms_test.go … |
| `Form.Files` | §5.2 | api_test.go, cond_test.go, review_test.go |
| `Form.Float` | §5.3 | api_test.go, forms_test.go, review_test.go |
| `Form.Group` | §5.3 | api_test.go, nesting_test.go |
| `Form.Int` | §5.3 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Form.Label` | §5.1 | api_test.go, forms_test.go, nesting_test.go |
| `Form.List` | §5.8 | api_test.go, nesting_test.go |
| `Form.Number` | §5.3 | api_test.go, conv_test.go |
| `Form.Parsed` | §5.3 | api_test.go, conv_test.go |
| `Form.Submit` | §5.1 | form_test.go, forms_test.go, ids_test.go, nesting_test.go … |
| `Form.Text` | §5.3 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Form.Texts` | §5.3 | api_test.go, cond_test.go, forms_test.go, review_test.go |
| `Form.Time` | §5.3 | api_test.go, forms_test.go, review_test.go, time_test.go |
| `Form.Valid` | §5.1 | forms_test.go, nesting_test.go, stores_test.go |
| `Form.Validate` | §5.1 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `Form.When` | §5.1 | cond_test.go, form_test.go, nesting_test.go |
| `FromRequest` | §5.1 | review_test.go |
| `FromValues` | §5.2 | cond_test.go, conv_test.go, forms_test.go, money_test.go … |
| `IndexPlaceholder` | §5.8 | api_test.go |
| `Integer` | §5.3 | api_test.go |
| `Kind` | §5.4 | api_test.go, nesting_test.go, review_test.go, stores_test.go |
| `Kind.String` | §5.10 | api_test.go, contract_test.go, form_test.go, id_fuzz_test.go … |
| `KindCheckbox` | §5.10 | api_test.go |
| `KindDate` | §5.10 | api_test.go |
| `KindDateTime` | §5.10 | api_test.go |
| `KindFile` | §5.10 | api_test.go |
| `KindFiles` | §5.10 | api_test.go |
| `KindGroup` | §5.10 | api_test.go, review_test.go |
| `KindHidden` | §5.10 | api_test.go |
| `KindList` | §5.10 | api_test.go |
| `KindMultiSelect` | §5.10 | api_test.go, review_test.go |
| `KindMultiline` | §5.10 | api_test.go |
| `KindNumber` | §5.10 | api_test.go |
| `KindPassword` | §5.10 | api_test.go |
| `KindSelect` | §5.10 | api_test.go |
| `KindText` | §5.10 | api_test.go |
| `KindToggle` | §5.10 | api_test.go |
| `List` | §5.8 | api_test.go, nesting_test.go |
| `List.Add` | §5.8 | id_fuzz_test.go, id_test.go, nesting_test.go, provider_test.go … |
| `List.Control` | §5.1 | api_test.go, conv_test.go, form_test.go, nesting_test.go |
| `List.Err` | §2 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `List.Items` | §— | form_test.go, nesting_test.go |
| `List.Label` | §5.1 | api_test.go, forms_test.go, nesting_test.go |
| `List.MaxItems` | §5.8 | nesting_test.go, slices_test.go |
| `List.MinItems` | §5.8 | nesting_test.go, slices_test.go |
| `List.When` | §5.1 | cond_test.go, form_test.go, nesting_test.go |
| `New` | §2 | api_test.go, cond_test.go, contract_test.go, conv_test.go … |
| `Not` | §4.2 | cond_test.go |
| `Source` | §3 | through its values and methods |

## `ui/forms/rule` (§6)

| Identifier | Interface | Tests |
|---|---|---|
| `AtLeast` | §6.2 | money_test.go, order_test.go |
| `AtMost` | §6.2 | money_test.go, order_test.go |
| `Between` | §6.2 | nesting_test.go, order_test.go |
| `Check` | §5.1 | file_test.go, forms_test.go, nesting_test.go, order_test.go … |
| `ContentType` | §6.2 | file_test.go |
| `CountryCode` | §6.2 | strings_test.go |
| `CurrencyCode` | §6.2 | strings_test.go |
| `Each` | §4 | slices_test.go |
| `Email` | §6.2 | example_test.go, id_test.go, strings_test.go |
| `ErrBetween` | §6.3 | nesting_test.go, order_test.go, rule_test.go |
| `ErrContentType` | §6.3 | file_test.go, rule_test.go |
| `ErrCountryCode` | §6.3 | rule_test.go, strings_test.go |
| `ErrCurrencyCode` | §6.3 | rule_test.go, strings_test.go |
| `ErrEmail` | §6.3 | rule_test.go, strings_test.go |
| `ErrMatches` | §6.3 | rule_test.go, strings_test.go |
| `ErrMax` | §5.8 | conv_test.go, order_test.go, rule_test.go |
| `ErrMaxItems` | §5.8 | nesting_test.go, rule_test.go, slices_test.go |
| `ErrMaxLen` | §6.3 | rule_test.go, strings_test.go |
| `ErrMaxSize` | §6.3 | file_test.go, rule_test.go |
| `ErrMin` | §5.9 | conv_test.go, forms_test.go, money_test.go, order_test.go … |
| `ErrMinItems` | §5.9 | forms_test.go, nesting_test.go, rule_test.go, slices_test.go |
| `ErrMinLen` | §6.3 | forms_test.go, rule_test.go, slices_test.go, stores_test.go … |
| `ErrMinSize` | §6.3 | file_test.go, rule_test.go |
| `ErrNotAfter` | §6.3 | rule_test.go, time_test.go |
| `ErrNotBefore` | §6.3 | rule_test.go, time_test.go |
| `ErrOneOf` | §6.3 | order_test.go, rule_test.go |
| `ErrPhone` | §6.3 | rule_test.go, strings_test.go |
| `ErrPostCode` | §6.3 | rule_test.go, strings_test.go |
| `ErrURL` | §6.3 | rule_test.go, strings_test.go |
| `Matches` | §6.2 | strings_test.go |
| `Max` | §5.8 | conv_test.go, nesting_test.go, order_test.go |
| `MaxItems` | §5.8 | nesting_test.go, slices_test.go |
| `MaxLen` | §6.2 | nesting_test.go, slices_test.go, strings_test.go |
| `MaxSize` | §6.2 | file_test.go |
| `Min` | §5.3 | cond_test.go, conv_test.go, forms_test.go, nesting_test.go … |
| `MinItems` | §5.8 | nesting_test.go, slices_test.go |
| `MinLen` | §6.2 | api_test.go, forms_test.go, nesting_test.go, slices_test.go … |
| `MinSize` | §6.2 | file_test.go |
| `New` | §2 | api_test.go, cond_test.go, contract_test.go, conv_test.go … |
| `NotAfter` | §6.2 | time_test.go |
| `NotBefore` | §6.2 | forms_test.go, time_test.go |
| `OneOf` | §6.2 | order_test.go |
| `Phone` | §6.2 | strings_test.go |
| `PostCode` | §6.2 | strings_test.go |
| `Rule` | §5.4 | rule_test.go, strings_test.go, time_test.go |
| `Rule.Attrs` | §5.10 | file_test.go, nesting_test.go, order_test.go, rule_test.go … |
| `Rule.Err` | §2 | api_test.go, cond_test.go, conv_test.go, forms_test.go … |
| `URL` | §6.2 | rule_test.go, strings_test.go |

## `ui/forms/formstest` (§7)

| Identifier | Interface | Tests |
|---|---|---|
| `AssertLabelled` | §7 | nesting_test.go, stores_test.go |
| `Errors` | §4.2 | form_test.go, forms_test.go, nesting_test.go, stores_test.go |
| `Submit` | §5.1 | form_test.go, forms_test.go, ids_test.go, nesting_test.go … |
