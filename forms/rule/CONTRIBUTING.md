# Adding a Rule to `forms/rule`

This contributor note specifies how to add a validation rule to the `forms/rule` catalogue.

## 1. Declare the default error in `forms/rule/errors.go`

Every catalogue rule declares a default error in `forms/rule/errors.go` so callers need not supply an error parameter when calling constructors:

- **Key:** Follow the naming convention `errors.forms.rule.<name>` (e.g., `errors.forms.rule.min_len`, `errors.forms.rule.max`). Keys are permanent: renaming one orphans existing translations.
- **Message text:** Use lower-case text with no final period (e.g. `"must be at most %d characters"`). The UI layer capitalizes messages for display.
- **Kind:** Always `errors.Invalid`.
- **Arguments:** Use typed declarations from `github.com/epistemer/errors`:
  - `Key`: no arguments (e.g., `ErrEmail`, `ErrURL`).
  - `Key1[int]`: counts, lengths, item limits (e.g., `ErrMinLen`, `ErrMinItems`).
  - `Key1[string]`: single bounds formatted with `fmt.Sprint` (e.g., `ErrMin`, `ErrMax`).
  - `Key1[i18n.Ter]`: calendar date bounds formatted with `i18n.StaticDate` (e.g., `ErrNotBefore`, `ErrNotAfter`).
  - `Key2[string, string]`: two string bounds (e.g., `ErrBetween`).

```go
var ErrMaxLen = errors.Key1[int]("errors.forms.rule.max_len", "must be at most %d characters", errors.Invalid)
```

## 2. Write the constructor

Rule constructors return a `Rule[T]` value:

- Construct the rule using a `Rule[T]{check: func(ctx context.Context, v T) error { ... }}` struct literal.
- The `check` function returns `nil` when `v` is valid, or the default error (instantiated with `.With(...)` if parameterized) on failure.
- If the rule implies HTML attributes (such as `minlength`, `maxlength`, `min`, `max`, `pattern`, `type`, or `accept`), chain `.with(name, value)`.
- Constructors do not accept an error argument: `Rule.Err(customErr)` already allows callers to override the default error.

```go
func MaxLen(n int) Rule[string] {
	return Rule[string]{
		check: func(_ context.Context, v string) error {
			if utf8.RuneCountInString(v) > n {
				return ErrMaxLen.With(n)
			}
			return nil
		},
	}.with("maxlength", strconv.Itoa(n))
}
```

## 3. Enforce misuse as a compile error

A rule's type parameter `T` (and any generic type constraints on the constructor) is what prevents misuse at compile time:

- `Field[T].Is(...)` accepts only `Rule[T]`.
- Attempting to apply a rule to an incompatible field type (such as passing `rule.MaxLen(5)` to a `Field[bool]`) fails to compile.
- Generic constraints should be as specific as necessary:
  - `T cmp.Ordered` for ordered values (`Min`, `Max`, `Between`).
  - `T interface{ Cmp(T) int }` for types with custom comparisons (`AtLeast`, `AtMost`).
  - `T comparable` for equality sets (`OneOf`).
  - `Rule[[]T]` for slice element validation (`MinItems`, `MaxItems`, `Each`).

## 4. Tests

Every catalogue rule requires:

1. **Table-driven unit test** in `forms/rule/*_test.go`:
   - Test passing and failing inputs using `rule.Check(ctx, v, r)`.
   - Assert failing cases match the expected default error with `errors.Is(err, rule.Err...)`.
   - Verify all implied HTML attributes via `r.Attrs()`.
2. **Compile-time safety test** under `forms/rule/testdata/nocompile/<name>/main.go`:
   - The first line must state the expected compiler error message:
     ```go
     // want: <expected compiler error text>
     ```
   - `TestNoCompile` in `nocompile_test.go` verifies that `go build` fails with that exact text.

## 5. Interface amendment

Adding a rule or modifying an error key changes the public contract of `github.com/epistemer/ui`:

- It constitutes an **interface amendment** (`docs/interface.md` §10).
- Amendments must be decided and approved according to the repository's owner rules and recorded in `docs/interface.md` §10 and `docs/status.md`.
