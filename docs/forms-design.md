# ui/forms: API design

Written 2026-11-16 by epistemer and Claude; brought into the repository on 2026-11-21 with the changes listed in section 15.

## 1. Purpose and reader

This document specifies the public API of forms, github.com/epistemer/ui/forms, the neozp package that declares, binds, validates and describes HTML forms. It is written for whoever implements it. The kernel, the module's other first package, has its own design: [brief.md](brief.md).

**What is decided.** Sections 3 to 10 are the design. The invariants in section 3 are fixed. Signatures elsewhere are normative in shape until interface.md is locked: a name or parameter order may change if the compiler or a test shows a better one, and the change is recorded in docs/status.md. After the lock, interface.md is the contract.

**What was open.** Section 14 lists the choices the first draft made without epistemer's decision, and what was decided.

**Out of scope.**

- HTML widgets, CSS and client scripts. Those belong to ui/inputs, which imports forms.
- Tables, filters and bulk actions. They will consume forms later.
- Moving zp's pages onto it.
- Code generation. This design needs none (section 3).

## 2. Background: what this replaces

zp's forms have five design flaws that no implementation fix can remove; this API exists to remove them. Everything else wrong with them is either a consequence of these or an implementation detail.

**zp's system, in three places.**

- A validator whose fluent chain binds from the request, runs rules and returns one error, in one call per field.
- Per page, a result struct, a Validate method built from those chains and returning a map of errors keyed by field name, and a form data literal that declares every field again for rendering, as a slice of any. There are 32 such Validate methods.
- Widgets and client scripts that read those literals.

**The five flaws.**

|  | Flaw | Evidence in zp | What the new API must guarantee |
|---|---|---|---|
| A | The chain executes; it does not describe. A field exists only during a POST. | Every field is declared three times (struct field, chain, widget), linked by a string literal. Required-ness is stated three ways (236 occurrences of the widget's flag alone). Create and edit pages duplicate each other. | A field is declared once, as a value that can bind, validate and be rendered. |
| B | Every rule exists on every type. | A generic validator gives a bool a maximum size and a date a URL check. Both compile and do nothing: the type switch has no matching case. Optional values are skipped by a reflection nil check in each rule. | A rule that does not apply to a field's type fails to compile. Optionality is handled once. |
| C | No codec. Values are strings at both ends. | Parsing lives in the binder, formatting at each call site. Enums have no binder: structs carry the enum as a string, and 27 custom rules only check that it is valid. | One codec per field owns both parse and format. Enums and domain types are first-class. |
| D | Cross-field conditions are written twice, in two languages. | Visibility is a dependency graph for the client (46 graphs, keyed by field-name strings). The same condition is rewritten as a server closure: 53 of 109 custom rules read another field, and depend on statement order. In one form, one field's condition names a constant of another enum than its siblings', and nothing catches it. | One condition value, typed against the field it reads, drives both server validation and client visibility. |
| E | Compound fields are not forms. | Field sets post client-built JSON in a hidden input. The client coerces numeric strings through a JavaScript Number before they reach a decimal. Sub-field names must match Go JSON tags. Sub-fields carry no value and no error; the group reports one opaque error. Children are a slice of any, and each container's type switch accepts a different subset. | A group is a form. Sub-fields bind, validate and report errors like any field. No any in the public API. |

**Not design flaws.** Parsing the request body repeatedly, twenty copy-pasted binders, 30-line type switches and coupling to a web framework are implementation problems. They disappear in a rewrite but do not shape the API. The manual model-to-form mapping in edit handlers is typed and compiler-checked, and stays.

**The abandoned rewrite.** A rewrite in zp aimed correctly at A, C and E. It failed on B: it made one struct per value type, each with its own rule methods, which produced ten pairs of value and pointer files (two of them 3,155 near-identical lines). It never addressed D, and it put rendering inside the input. After 129 commits one test page used it.

## 3. Decisions and invariants

The design rests on decisions epistemer has made and ten invariants that follow from the five flaws. An implementation that breaks an invariant is wrong even if its tests pass.

**Decisions already made.**

- **Go 1.27 or later.** The API uses generic methods. epistemer confirmed by test that a method may declare its own type parameters but cannot narrow the receiver's. Go 1.27's specification allows it (checked 2026-11-21, Go 1.27.4).
- **The fluent chain stays.** What changes is what the chain produces: a field value, not an immediate result.
- **Rules are typed values** passed to the chain, as in .Is(rule.MaxLen(255)). They are not methods on the field.
- **Type changes happen in the chain** through a generic As step.
- **No code generation.** Reconsider only if a later requirement has no other answer.
- **Rules carry built-in errors** (2026-11-21), declared in rule and overridable with .Err(err) (section 6.3).
- **Choices are models' option.SelectOption** (2026-11-21, models ADR 0006): Enum and Options take what models generates (sections 5.1, 5.3).
- **forms lives in the ui module** as github.com/epistemer/ui/forms (2026-11-21).

**Invariants.**

| # | Invariant | Removes flaw |
|---|---|---|
| I1 | A field is declared once. Its name, label, type, rules and conditions live in that one declaration. | A |
| I2 | Declaring a form reads nothing. A form can be rendered without a request. | A |
| I3 | Every field is bound before any rule runs, so a rule may read any other field. | D |
| I4 | A rule that does not apply to a field's type does not compile. | B |
| I5 | Presence is state, not type. There is one Field[T]; there are no pointer variants. | B |
| I6 | One codec per field owns both parsing and formatting. | C |
| I7 | A cross-field condition is one value. The server evaluates it and the client receives it serialised. | D |
| I8 | A group is a form. Nested fields bind, validate and report errors individually. | E |
| I9 | The public API has no any, no []any and no map[string]error. | E |
| I10 | Mistakes by the programmer panic when the form is declared. Input from a user never panics. | all |

**Dependencies.** forms imports the standard library, github.com/epistemer/errors, github.com/epistemer/i18n and github.com/epistemer/models/option (which imports only i18n). It imports no templ, no web framework and no decimal library. Every user-facing error is an errors declaration; every label is an i18n.Ter.

**Packages.** All under github.com/epistemer/ui.

| Package | Holds | Imports |
|---|---|---|
| forms | Form, Field[T], List[G], Conv, Cond, Source, Control | forms/rule, errors, i18n, models/option |
| forms/rule | Rule[T] and the rule catalogue | errors only |
| forms/formstest | Helpers for testing a form without HTTP | forms |

forms/rule replaces the separate validate package from the earlier package list. It does not import forms, so it can be split into a module of its own later if non-form callers need it.

## 4. Core model

A form is a Go struct that embeds *forms.Form and holds one typed handle per field. A constructor declares the fields; each declaration registers itself with the form in order. One instance serves one request.

```go
type StoreForm struct {
	*forms.Form
	Name        *forms.Field[string]
	Address     *forms.Field[string]
	Type        *forms.Field[models.StoreKind]
	Description *forms.Field[string]
	Icon        *forms.Field[*multipart.FileHeader]
}

func NewStoreForm() *StoreForm {
	f := &StoreForm{Form: forms.New()}
	f.Name = f.Text("name").
		Label(models.L10nStore_Name).
		Required().
		Is(rule.MinLen(3), rule.MaxLen(255))
	// ... the other fields, section 11
	return f
}
```

**Why this shape.**

- The struct field gives the handler typed access: f.Name.Value() is a string, f.Type.Value() is the enum.
- Registration at declaration gives the form its ordered field list with no reflection and no second list. This is the only place codegen could have helped, and it is not needed.
- The struct field and its assignment are linked by the compiler, not by a string. The wire name "name" appears once.

**Why one instance per request.** A field holds its submitted text, its parsed value and its error. Keeping that state in the field makes the handle the single thing a handler and a template need. Construction is a few allocations. Dependencies such as a service or a database handle are constructor parameters, captured by rule closures.

### 4.1 Form

```go
func New() *Form

// Lifecycle.
func (f *Form) Bind(src Source)
func (f *Form) Validate(ctx context.Context) bool
func (f *Form) Submit(ctx context.Context, r *http.Request) bool // FromRequest, Bind, Validate

// Outcome.
func (f *Form) Valid() bool
func (f *Form) Err() error      // section 9
func (f *Form) Fail(err error)  // a form-level error, for example from the service layer

// A check that belongs to no single field. Runs after all field rules pass.
func (f *Form) Check(fn func(ctx context.Context) error)

// Traversal for renderers, section 10.
func (f *Form) Controls() iter.Seq[Control]
```

A Form carries no action, method, CSRF token or HTMX attribute. Those are transport and belong to the handler and to ui.

### 4.2 Source

```go
type Source interface {
	Values(name string) []string
	Files(name string) []*multipart.FileHeader
	Names() iter.Seq[string] // every submitted name; lists need it (section 8)
}

func FromRequest(r *http.Request) (Source, error) // parses the body once
func FromValues(v url.Values) Source              // tests, query-string filters
```

FromRequest takes *http.Request, not a framework context, which is what handlers' generated admin handlers receive. A request that cannot be parsed makes Submit record a form-level error and return false.

### 4.3 Lifecycle

1. **Declare.** The constructor builds fields. Nothing is read. Programming mistakes panic here: a duplicate or malformed name, a condition cycle, a builder call after Bind.
2. **Set (optional).** A handler may give fields initial values with Set, for an edit page.
3. **Bind.** For each field in declaration order the form records the submitted text, decides whether the field is present, and runs its conversion chain. A conversion failure is recorded as that field's error. No rule runs. Names the form did not declare are ignored. A second Bind on the same form panics.
4. **Validate.**
   1. Resolve which fields are active (section 7). An inactive field has its value, text and error cleared.
   2. An active field with a conversion error keeps it.
   3. An active field that is absent gets its required error if it is required; otherwise its rules are skipped.
   4. An active, present field runs its rules in order and stops at the first failure.
   5. If no field has an error, the form's Check functions run in order; the first failure becomes the form-level error.
5. **Describe.** Controls can be read at any point: before Bind it shows initial values, after a failed Validate it shows the submitted text and the errors.

A form is not safe for concurrent use.

## 5. Fields and codecs

There is one field type, Field[T]. A constructor on the form fixes the wire name and the first codec; builder methods add everything else; As changes T.

### 5.1 Constructors

```go
func (f *Form) Text(name string) *Field[string]
func (f *Form) Texts(name string) *Field[[]string] // a repeated name: checkbox group, multi-select
func (f *Form) Int(name string) *Field[int]
func (f *Form) Float(name string) *Field[float64]
func (f *Form) Bool(name string) *Field[bool]     // three states: absent, true, false
func (f *Form) Checkbox(name string) *Field[bool] // two states: a missing name is false
func (f *Form) Time(name, layout string) *Field[time.Time]
func (f *Form) Date(name string) *Field[time.Time] // layout 2006-01-02
func (f *Form) File(name string) *Field[*multipart.FileHeader]
func (f *Form) Files(name string) *Field[[]*multipart.FileHeader]

// Generic methods (Go 1.27).
func (f *Form) Number[T Integer | Float](name string) *Field[T] // f.Number[int32]("depth")
func (f *Form) Enum[E EnumValue](name string, values []E) *Field[E]
func (f *Form) Parsed[T encoding.TextMarshaler, PT interface {
	*T
	encoding.TextUnmarshaler
}](name string) *Field[T] // f.Parsed[decimal.Decimal]("fixed")

type EnumValue interface {
	~string
	option.SelectOption // Value() string, Label() i18n.TKer
}
```

- **Names** match [a-z][a-z0-9_]*. Dots and brackets are reserved for paths (section 8). A bad or duplicate name panics.
- **Enum** fits what neozp already generates: every enum value of an app's models is an option.SelectOption (models' Label() and Value(), A3), and sqlc's All<Type>Values() gives the typed slice. Enum rejects a value outside values and supplies the field's options. sqlc's Valid() is not required: the membership check already rejects anything else. This removes zp's string-typed enum fields and the 27 custom rules that only checked validity.
- **Parsed** covers any type with text marshalling, including decimal.Decimal, without forms importing it. An application wraps it once: func Decimal(f *forms.Form, name string) *forms.Field[decimal.Decimal].
- **Int returns Field[int]** on purpose: rule.Min(0) then infers int from the untyped constant. For other widths write the type once: f.Number[int64]("id").Is(rule.Min[int64](1)).

**To verify first.** Explicit instantiation on a method call (f.Number[int32](...)) and inference of PT in Parsed are assumed from how generic functions behave. If the 1.27 compiler rejects either, make that constructor a package-level function, forms.Parsed[T](f, name), and record it.

### 5.2 Presence

A field is present or absent after Bind. This replaces the pointer variants of zp's validator and of its abandoned rewrite.

| Constructor | Absent when |
|---|---|
| Text | the name is missing or the value is empty after trimming. Surrounding white space is trimmed by default; .Raw() keeps it, for passwords |
| Texts, Files | no non-empty item was submitted |
| Int, Float, Number, Time, Date, Enum, Parsed, Bool | the name is missing or the value is empty |
| Checkbox | never. A missing name is false. Required on a checkbox means it must be ticked |
| File | no file was uploaded under the name |

### 5.3 Builder methods

Each returns the field, so they chain. All are legal until Bind.

```go
func (f *Field[T]) Label(l i18n.Ter) *Field[T]       // default: the name
func (f *Field[T]) Placeholder(p i18n.Ter) *Field[T] // default: the label
func (f *Field[T]) Help(h i18n.Ter) *Field[T]

func (f *Field[T]) Required() *Field[T]               // reports forms.ErrRequired
func (f *Field[T]) RequiredWith(err error) *Field[T]  // reports err instead
func (f *Field[T]) InvalidWith(err error) *Field[T]   // shown when conversion fails; default forms.ErrInvalid

func (f *Field[T]) Is(rules ...rule.Rule[T]) *Field[T] // section 6
func (f *Field[T]) When(c Cond) *Field[T]              // section 7
func (f *Field[T]) Options(opts ...option.SelectOption) *Field[T]
func (f *Field[T]) Hint(k Kind) *Field[T]              // section 10

func (f *Field[T]) As[U any](c Conv[T, U]) *Field[U]
```

Required is the single statement of required-ness. The renderer reads it from the field; no widget takes an IsRequired argument.

Options gives the choices for a select or radio group and also validates membership, comparing the submitted text with each option's Value(). Enum sets them itself; Options is for choices loaded at run time, such as rows a query returned, and takes models' All<Type>Options() as it is: f.Options(models.AllOrderStatusOptions()...). An application type becomes a choice by implementing Value() string and Label() i18n.TKer.

### 5.4 State

```go
func (f *Field[T]) Set(v T) *Field[T] // an initial value; marks the field present
func (f *Field[T]) SetPtr(p *T) *Field[T] // Set(*p) when p is not nil; for nullable columns
func (f *Field[T]) Value() T          // the zero T when absent or in error
func (f *Field[T]) Get() (T, bool)
func (f *Field[T]) Ptr() *T           // nil when absent: for nullable columns
func (f *Field[T]) Err() error
func (f *Field[T]) Fail(err error)    // attach an error after Validate, e.g. a unique violation
```

Set is how an edit page fills a form. It replaces a separate edit-result struct per page, and it is typed: f.Kind.Set(st.Kind) takes the enum, not a string.

When a submitted value fails conversion, the field keeps the submitted text, so the page shows abc back in a number input and not 0.

### 5.5 As and Conv

```go
type Conv[T, U any] struct {
	Parse  func(T) (U, error)
	Format func(U) T
}
```

As appends a stage to the field's conversion chain and returns a handle of the new type.

- **The chain is a pipeline:** text, then T, then U. Rules added before As check the T; rules added after check the U. They run in that order and stop at the first failure.
- **The old handle is dead.** Using it after As panics. Keep only the returned one.
- **Label, help, required-ness, When and Options carry over.** Options compare wire text, so they do not depend on the type.
- **A Parse error** that is an errors declaration is shown as it is. Any other error is wrapped as the cause of the field's invalid error, so logs keep it and the user sees a translated message.

```go
// The shop stores a cooldown as a duration and edits it as whole seconds.
var seconds = forms.Conv[int, time.Duration]{
	Parse:  func(n int) (time.Duration, error) { return time.Duration(n) * time.Second, nil },
	Format: func(d time.Duration) int { return int(d / time.Second) },
}

f.Cooldown = f.Int("cooldown").Is(rule.Min(0)).As(seconds) // *forms.Field[time.Duration]
```

## 6. Rules

A rule is a value of type rule.Rule[T]. Field[T].Is accepts only Rule[T], so the compiler rejects a rule for the wrong type. Rules never see a pointer and never check for nil: an absent field does not run them.

f.Bool("active").Is(rule.MaxLen(5)) // does not compile: Rule[string] is not Rule[bool]

### 6.1 The type

```go
package rule

type Rule[T any] struct{ /* unexported */ }

// New makes a custom rule. A nil return passes.
func New[T any](check func(ctx context.Context, v T) error) Rule[T]

// Err returns a copy that reports err when the rule fails.
func (r Rule[T]) Err(err error) Rule[T]

// Check runs rules in order and returns the first failure. Usable without a form.
func Check[T any](ctx context.Context, v T, rules ...Rule[T]) error

// Attrs lists the HTML attributes a rule implies, such as maxlength="255".
func (r Rule[T]) Attrs() iter.Seq2[string, string]
```

### 6.2 The catalogue for v0.1

The constraint of each constructor is what makes misuse a compile error.

| Applies to | Constructors |
|---|---|
| string | MinLen(n), MaxLen(n) (counted in runes), Matches(re), Email(), URL(), Phone(), CountryCode(), CurrencyCode(), PostCode(country) |
| T cmp.Ordered | Min(v), Max(v), Between(lo, hi) |
| T with Cmp(T) int, such as decimal.Decimal | AtLeast(v), AtMost(v) |
| T comparable | OneOf(vs...) |
| time.Time | NotBefore(t), NotAfter(t) |
| *multipart.FileHeader | MinSize(bytes), MaxSize(bytes), ContentType(types...) |
| []T | MinItems(n), MaxItems(n), Each(rules...) which lifts Rule[T] to Rule[[]T] |

zp overloads MinSize and MaxSize for string length and file size, and MinLen for slices. Here each name has one meaning and one type.

### 6.3 Errors

Every catalogue rule has a declared default error in rule, built with the errors package, so a rule needs no error argument:

```go
var (
	ErrMaxLen = errors.Key1[int]("errors.forms.rule.max_len", "Must be at most %d characters", errors.Invalid)
	ErrEmail  = errors.Key("errors.forms.rule.email", "Must be an email address", errors.Invalid)
)

func MaxLen(n int) Rule[string] // reports ErrMaxLen.With(n)
```

A form that wants its own wording overrides it: rule.MaxLen(255).Err(ErrStoreNameTooLong).

Keys follow errors' convention (its D11), errors.<area>.<name>: errors.forms.rule.<name> for the catalogue and errors.forms.<name> for the errors forms itself declares (section 9), so rule.ErrMinItems and forms.ErrMinItems never share a key. Keys are permanent: renaming one orphans its translations.

This differs from the example first approved, .Is(rule.Max(255, err)), where the error is a parameter. zp declares 10 to 25 untranslated errors.New values per form only to feed those parameters. Defaults remove them, and the messages are translatable because their keys are i18n keys. epistemer confirmed the defaults on 2026-11-21.

### 6.4 Custom rules

```go
// A database check. s is a constructor parameter of the form.
f.Name.Is(rule.New(func(ctx context.Context, name string) error {
	if taken, _ := s.PromotionNameTaken(ctx, name); taken {
		return ErrPromotionNameTaken
	}
	return nil
}))

// A rule that delegates to another neozp package. a is the app's *auth.Auth,
// a constructor parameter; CheckPassword returns declared errors people can
// read, so they pass through unchanged.
f.Password.Is(rule.New(func(ctx context.Context, pw string) error {
	return a.CheckPassword(ctx, pw, f.Email.Value())
}))

// A rule that reads another field. Legal because of invariant I3.
f.Confirm.Is(rule.New(func(_ context.Context, v string) error {
	if v != f.Password.Value() {
		return ErrPasswordMismatch
	}
	return nil
}))
```

- Rules run in the order given and stop at the first failure for that field. Put database rules last so they run only on values that already passed the cheap rules.
- A rule must not change the form.
- A rule that depends on another field being a particular value is usually a condition, not a rule (section 7).

## 7. Conditions

A condition is one value, forms.Cond, built from a typed field handle. The server evaluates it during Validate; the renderer receives it as JSON for the client. Writing it once is what removes flaw D.

```go
type Cond struct{ /* unexported */ }

func (f *Field[T]) Eq(v T) Cond     // the field's value equals v
func (f *Field[T]) In(vs ...T) Cond // the field's value is one of vs
func (f *Field[T]) Filled() Cond    // the field is present

func All(cs ...Cond) Cond
func Any(cs ...Cond) Cond
func Not(c Cond) Cond

func (c Cond) MarshalJSON() ([]byte, error)
```

### 7.1 When

field.When(c) makes the field **active** only while c holds. Groups and lists take When as well.

```go
f.Fixed = f.Parsed[decimal.Decimal]("fixed").
	When(f.Type.Eq(models.PromotionTypeFixed)).
	Required().
	Is(rule.AtLeast(decimal.Zero), rule.AtMost(ten))
```

- **Required plus When is a conditional requirement.** This one line replaces a rules.DependencyGraph entry and a server closure of the form if r.TypeString == "fixed" && d == nil { return err }.
- **Eq takes a T.** f.Type.Eq(models.DiscountBasePercentage) on a Field[models.PromotionType] does not compile. zp's string-keyed version of that mistake did.
- **An inactive field is cleared on the server:** its value, submitted text and error are dropped, its rules do not run, and Value() returns the zero value. A hidden input's stale value cannot reach the handler.
- **On the client** the widget hides and disables an inactive field. That behaviour belongs to ui.

### 7.2 Evaluation

- Comparison is on wire text: Eq(v) holds when the field is active, present, and its formatted value equals the formatted v. The client compares the same strings, so the two sides cannot disagree.
- A condition that reads an inactive or absent field is false for Eq, In and Filled.
- Conditions are resolved after Bind, on demand, with the result remembered. Declaration order does not matter.
- A cycle panics at the When call that closes it.
- A condition may read a field in the same form or in an enclosing form. It may not reach into a list's items from outside; that panics.

### 7.3 Wire format

Keep zp's wire format, so its client evaluator can move to ui/inputs with few changes.

```go
{"typ":"c","k":"type","op":"eq","v":"fixed"}
{"typ":"l","op":"and","args":[ ... ]}
```

- k is the field's full wire path (section 8).
- In is written as an or of eq nodes.
- Filled adds one operator, {"typ":"c","k":"...","op":"filled"}.
- zp's gt, lt, gte and lte are not in v0.1. Add them when a real form needs one.

**Conformance vectors.** forms ships testdata/cond/*.json: each file holds a condition, a set of submitted values and the expected result. The Go evaluator is tested against them, and ui's client evaluator must pass the same files. This is the only guarantee that the two evaluators agree.

## 8. Groups and lists

A group is a nested form under a name; a list is a repeatable group. Both are built by a function that receives a sub-form and declares fields on it exactly as a top-level constructor does. Members are ordinary fields: they bind, validate and report errors one by one.

```go
// Group declares a nested form and returns what build returns.
func (f *Form) Group[G any](name string, build func(g *Form) G) G

// List declares a repeatable nested form.
func (f *Form) List[G any](name string, build func(g *Form) G) *List[G]

// On a sub-form, these describe the group itself.
func (f *Form) Label(l i18n.Ter) *Form
func (f *Form) When(c Cond) *Form
func (f *Form) Active() bool // after Validate

type List[G any] struct{ /* unexported */ }

func (l *List[G]) Label(t i18n.Ter) *List[G]
func (l *List[G]) When(c Cond) *List[G]
func (l *List[G]) MinItems(n int) *List[G]
func (l *List[G]) MaxItems(n int) *List[G]
func (l *List[G]) Items() []G
func (l *List[G]) Add() G // append an item; used to give a list initial values
func (l *List[G]) Err() error
type PercentageForm struct {
	*forms.Form
	Value, Max, Min *forms.Field[decimal.Decimal]
}

func newPercentage(g *forms.Form) *PercentageForm {
	p := &PercentageForm{Form: g}
	p.Value = Decimal(g, "value").Required().Is(rule.AtLeast(zero), rule.AtMost(hundred))
	p.Max = Decimal(g, "max")
	p.Min = Decimal(g, "min").Required()
	return p
}

f.Percentage = f.Group("percentage", newPercentage) // *PercentageForm
f.Percentage.When(f.Type.Eq(models.PromotionTypePercentage))
```

### 8.1 Wire names

Nested fields post under path names. There is no client-built JSON and no hidden aggregate input.

| Declared as | Posted name |
|---|---|
| field name on the root | name |
| field value in group percentage | percentage.value |
| field type in item 0 of list constraints | constraints[0].type |
| deeper nesting | a.b[2].c |

This fixes three things at once: decimals never pass through a JavaScript Number, sub-field names are declared in Go and not tied to JSON tags, and every sub-field has an addressable error.

Texts and Files are not lists. They read a repeated plain name, tags=a&tags=b.

### 8.2 Binding a list

1. Scan Source.Names() for names that start with <path>[. Collect the distinct indices.
2. An index is one to six decimal digits. Anything else under that prefix is ignored.
3. Sort the indices numerically. They need not be contiguous: the client may have removed rows.
4. For each index, call build with a sub-form whose path is <path>[i], then bind it.
5. After binding, renumber the items 0..n-1 in that order. Paths in errors and in a re-rendered page use the new numbers.

A list binds at most MaxItems items, or forms.DefaultMaxItems (256) when none is set. More than that is a list-level error and the extra items are not built.

MinItems and MaxItems report default errors declared in forms. A list-level error sits at the list's own path.

### 8.3 The template row

A client that adds rows needs a blank row to clone. List exposes one through its control (section 10): an item built with the index placeholder forms.IndexPlaceholder, the string __i__, so its fields are named constraints[__i__].type. The client clones it and substitutes the next index everywhere, which also rewrites the conditions inside it.

### 8.4 Conditions inside items

An item's fields refer to their siblings through the handles created in build:

```go
func newConstraint(g *forms.Form) *ConstraintForm {
	c := &ConstraintForm{Form: g}
	c.Type = g.Enum("type", models.AllConstraintTypeValues()).Required()
	monetary := c.Type.Eq(models.ConstraintTypeMonetary)
	c.MinAmount = Decimal(g, "min_amount").When(monetary)
	c.MaxAmount = Decimal(g, "max_amount").When(monetary)
	// ...
	return c
}
```

Each item has its own handles, so each item's conditions read that item's type. This replaces the per-item dependency graphs of zp's JSON input.

### 8.5 From sub-form to domain type

forms does not build shop.Percentage or shop.Constraint. The application writes a method on its sub-form, func (p *PercentageForm) Percentage() shop.Percentage, that reads the typed handles. That keeps the mapping explicit and compiler-checked.

## 9. Errors

An error belongs to the field that reported it and is read from that field's handle. There is no map keyed by name. A field holds at most one error: the first failure.

```go
// FieldError ties an error to the node that reported it.
type FieldError struct {
	Path  string   // "name", "percentage.value", "constraints[2].type"; "" for the form itself
	Label i18n.Ter
	Err   error
}

func (e *FieldError) Error() string // "<path>: <err>", for logs
func (e *FieldError) Unwrap() error

// Err is nil when the form is valid. Otherwise it joins one *FieldError per
// failing node: the form-level error first, then fields in declaration order.
func (f *Form) Err() error

func (f *Form) Errors() iter.Seq[*FieldError]
```

**Where each kind of error lands.**

| Source | Stored on | Default declaration |
|---|---|---|
| Request cannot be parsed | the form (path "") | forms.ErrUnreadable |
| Conversion fails | the field | forms.ErrInvalid, or the field's InvalidWith |
| Value is not among the options | the field | forms.ErrNotAnOption |
| Required field is absent | the field | forms.ErrRequired, or the field's RequiredWith |
| A rule fails | the field | the rule's error (section 6.3) |
| MinItems, MaxItems, too many items | the list | forms.ErrMinItems, forms.ErrMaxItems (errors.Key1[int]) |
| A Check function fails | the form or group that declared it | what the function returned |
| Fail(err) | the form or field it was called on | what was passed |

**Working with the errors package.** Every default is declared with errors.Key or errors.Key1 and the kind errors.Invalid.

- errors.T(ctx, field.Err()) gives the translated text for one field. This is what a widget prints.
- errors.All(form.Err()) yields one message per failing node, because Err is a join and FieldError unwraps.
- errors.Is(form.Err(), rule.ErrMaxLen) works, which makes tests direct.
- errors.KindOf(form.Err()) is Invalid, whose status is 400. Re-rendering the form with that status or another (zp answers 422) is the handler's choice; whether htmx swaps the response is ui/inputs' concern.

**Undeclared errors.** A custom rule should return an errors declaration. If it returns a plain errors.New, the errors package shows its generic message to the user. Do not work around that inside forms: the generic message is the signal that a declaration is missing.

**Service-layer errors.** After a successful Validate, a handler that gets an error from the service calls f.Fail(err) for a general failure or f.Name.Fail(err) when the error is about one field, then renders the form again. This replaces the separate err error parameter that every zp page template takes.

## 10. Rendering contract

forms renders nothing. It exposes each node of a form as a Control: a non-generic, read-only view that holds everything a widget needs and nothing typed by T. The widget package, ui/inputs, is the only consumer.

Control is a concrete struct, not an interface. Go 1.27 does not let generic methods satisfy interfaces, and the builder methods on Field[T] already use the names a view would want (Label, Help). A separate view type avoids both problems.

```go
type Control struct{ /* unexported */ }

func (f *Field[T]) Control() Control
func (l *List[G]) Control() Control
func (f *Form) Control() Control             // the group itself
func (f *Form) Controls() iter.Seq[Control]  // its members, in declaration order

func (c Control) Kind() Kind
func (c Control) Path() string            // the HTML name attribute
func (c Control) Label() i18n.Ter
func (c Control) Placeholder() i18n.Ter
func (c Control) Help() i18n.Ter
func (c Control) Required() bool
func (c Control) Active() bool            // false: render hidden and disabled
func (c Control) When() (Cond, bool)      // the condition to serialise for the client
func (c Control) Text() []string          // submitted text if bound, else the formatted value
func (c Control) Options() iter.Seq[Choice]
func (c Control) Attrs() iter.Seq2[string, string] // from rules: maxlength, min, max, pattern, accept
func (c Control) Err() error

// Groups and lists.
func (c Control) Children() iter.Seq[Control] // a group's members, or a list's items as groups
func (c Control) Template() (Control, bool)   // a list's blank row, section 8.3

type Choice struct {
	Value    string
	Label    i18n.Ter
	Selected bool
}
```

### 10.1 Kind

Kind is a closed set. Each constructor sets a default; Field.Hint overrides it.

| Kind | Default for | Typical override |
|---|---|---|
| KindText | Text, Parsed |  |
| KindMultiline, KindPassword, KindHidden |  | Text(...).Hint(...) |
| KindNumber | Int, Float, Number |  |
| KindCheckbox | Checkbox |  |
| KindToggle | Bool |  |
| KindDate, KindDateTime | Date, Time |  |
| KindSelect | Enum, any field with Options |  |
| KindMultiSelect | Texts with Options |  |
| KindFile, KindFiles | File, Files |  |
| KindGroup, KindList | Group, List |  |

Input subtypes such as email, url and tel are not kinds. The rules Email(), URL() and Phone() contribute a type attribute through Attrs.

### 10.2 How ui/inputs is expected to use it

This is a recommendation for the next package, stated here because it shaped the contract.

- **Typed widget calls in templates.** inputs.Date(f *forms.Field[time.Time]) and inputs.Select[T any](f *forms.Field[T]) take the typed handle and read its Control(). A date widget on a text field does not compile. Layout stays in the template, where zp hard-codes it in a type switch.
- **An automatic renderer.** inputs.Auto(f *forms.Form) walks Controls() and switches on Kind. Table filters and bulk-action forms in view need this. Because Kind is closed, the switch can be tested for exhaustiveness.

Neither needs a type switch on any, which zp's form data struct forces today.

### 10.3 What Control deliberately lacks

- **The typed value.** A widget prints Text(). Only handlers read Value().
- **File previews and current-file URLs.** An edit page passes the existing file's URL to the file widget as an argument.
- **Widget parameters** such as a search endpoint for an autocomplete. Those are arguments of the widget call, not properties of the field.

- **The label of a chosen row** in a remote select (an id the options were not preloaded for). models declares listing.Labeler for that (ADR 0006, D4); ui/inputs' widget takes one as an argument, and forms stays out of it.

## 11. Worked examples

Two forms of the toy shop, shaped like real forms of zp: one form serving create and edit, and one with conditions, a group and a list. Both must compile in examples/shop as part of the acceptance tests (section 13). Model and enum names are those models would generate for the shop's schema; handlers are written as handlers' admin handlers, with //h: directives and its generated URL builders.

### 11.1 Store: one form for create and edit

In zp, a form like this is two result structs, two Validate methods and two form data literals that differ in three lines.

```go
type StoreForm struct {
	*forms.Form
	Name        *forms.Field[string]
	Address     *forms.Field[string]
	Kind        *forms.Field[models.StoreKind]
	Description *forms.Field[string]
	Logo        *forms.Field[*multipart.FileHeader]
}

func NewStoreForm() *StoreForm {
	f := &StoreForm{Form: forms.New()}
	f.Name = f.Text("name").
		Label(models.L10nStore_Name).
		Required().
		Is(rule.MinLen(3), rule.MaxLen(255))
	f.Address = f.Text("address").
		Label(models.L10nStore_Address).
		Required().
		Is(rule.MaxLen(255))
	f.Kind = f.Enum("kind", models.AllStoreKindValues()).
		Label(models.L10nStore_Kind).
		Required()
	f.Description = f.Text("description").
		Label(models.L10nStore_Description).
		Hint(forms.KindMultiline).
		Is(rule.MaxLen(255))
	f.Logo = f.File("logo").
		Label(models.L10nStore_Logo).
		Is(rule.MaxSize(5 << 20))
	return f
}
```

```go
// Create: the logo is mandatory.
//
//h:key stores.create
//h:method POST
//h:path /stores
//h:desc Create a store
func (h *H) StoreCreate(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	f := NewStoreForm()
	f.Logo.Required()
	if !f.Submit(ctx, r) {
		return h.invalid(w, r, pages.StoreNew(f))
	}
	st, err := h.s.CreateStore(ctx,
		f.Name.Value(), f.Description.Ptr(), f.Address.Value(), f.Kind.Value(), f.Logo.Value())
	if err != nil {
		f.Fail(err)
		return h.invalid(w, r, pages.StoreNew(f))
	}
	http.Redirect(w, r, adminroutes.Store(st.ID), http.StatusSeeOther)
	return nil
}

// Edit: the same form, filled from the model. The logo stays optional.
//
//h:key stores.edit
//h:method GET
//h:path /stores/{id}/edit
//h:desc Edit a store
//h:perm stores.update
func (h *H) StoreEdit(w http.ResponseWriter, r *http.Request, id int64) error {
	st, err := h.s.GetStore(r.Context(), id)
	if err != nil {
		return err // a declared NotFound error; handlers' error writer renders the page
	}
	f := NewStoreForm()
	f.Name.Set(st.Name)
	f.Address.Set(st.Address)
	f.Kind.Set(st.Kind)
	f.Description.SetPtr(st.Description)
	return pages.StoreEdit(id, f, st.LogoURL).Render(r.Context(), w)
}
```

h.invalid is the application's own helper: it writes the status the app chose for a form that failed (400 from errors.Invalid, or 422) and renders the page.

What is gone: both result structs, both Validate methods, ten errors.New values per page, every lookup of an error by field name, every IsRequired: true, and the string round trip of the enum.

What remains on purpose: the edit handler still copies model fields into the form by hand. That mapping is typed, and removing it would need reflection or codegen.

### 11.2 Promotion: conditions, a group and a list

The shape of zp's largest form (505 lines of template): a type enum decides which of several fields and groups apply, and a list of constraints whose own fields depend on each item's type.

```go
type PromotionForm struct {
	*forms.Form
	Name        *forms.Field[string]
	Type        *forms.Field[models.PromotionType]
	Base        *forms.Field[models.DiscountBase]
	MinQuantity *forms.Field[int32]
	Fixed       *forms.Field[decimal.Decimal]
	Absolute    *forms.Field[decimal.Decimal]
	Percentage  *PercentageForm
	Points      *PointsForm
	Constraints *forms.List[*ConstraintForm]
}

func NewPromotionForm(s *service.Service) *PromotionForm {
	f := &PromotionForm{Form: forms.New()}

	f.Name = f.Text("name").Required().
		Is(rule.MinLen(3), rule.MaxLen(255), uniquePromotionName(s))

	f.Type = f.Enum("type", models.AllPromotionTypeValues()).Required()
	is := f.Type.Eq // every condition below is typed against PromotionType

	f.Base = f.Enum("base", models.AllDiscountBaseValues()).
		When(is(models.PromotionTypePercentage)).Required()

	f.MinQuantity = f.Number[int32]("min_quantity").
		Is(rule.Between[int32](0, 100))

	f.Fixed = Decimal(f.Form, "fixed").
		When(is(models.PromotionTypeFixed)).Required().
		Is(rule.AtLeast(zero), rule.AtMost(ten))

	f.Absolute = Decimal(f.Form, "absolute").
		When(is(models.PromotionTypeAbsolute)).Required().
		Is(rule.AtLeast(zero), rule.AtMost(hundred))

	f.Percentage = f.Group("percentage", newPercentage) // section 8
	f.Percentage.When(is(models.PromotionTypePercentage))

	f.Points = f.Group("points", newPoints)
	f.Points.When(is(models.PromotionTypePoints))

	f.Constraints = f.List("constraints", newConstraint).MinItems(1) // section 8.4

	return f
}
```

A submission for a fixed promotion with one monetary constraint posts these names:

```
name=Gold
type=fixed
fixed=2.50
constraints[0].type=monetary
constraints[0].min_amount=100
constraints[0].max_amount=5000
```

If a stale percentage.value arrives with it, the group is inactive, so the value is cleared and f.Percentage.Active() is false.

What is gone, against zp's form of this shape: 26 errors.New values, the result struct, a 110-line Validate method, a 300-line form data literal, three dependency graphs, the JSON hidden inputs and the validity methods that could only say yes or no.

## 12. What to take from zp

zp and its abandoned rewrite of the validator are private and read-only references for behaviour. Take data, test cases and one wire format; take no architecture. The guide to which files hold what is kept outside this public repository, in the maintainers' notes. In short:

- **Data:** the regular expressions and tables behind Email, Phone, PostCode, CountryCode and CurrencyCode, choosing the more complete copy where the two sources differ.
- **Test cases:** the validator's and the rewrite's table tests (binding edge cases, enums, files, nested binding, optional strings), rewritten against this API.
- **Wire format:** the JSON shape of conditions (section 7.3); the conformance vectors are derived from zp's client evaluator.
- **Ideas the rewrite got right:** a field object that owns its value and error; a source abstraction that parses once; a nested form that binds and validates recursively; a rule catalogue broad enough that custom rules are rare.

Names, paths and identifiers from zp do not enter this repository.

## 13. Build order and acceptance tests

Forms is built in the roadmap's milestones F1 to F6 (docs/roadmap.md), after the module's M0. Each ends with tests that pass, and none depends on a later one. The compiler spike comes first (M0.3) because several signatures here rest on assumptions about Go 1.27.

### 13.1 Steps

1. **Repository** (M0.1). Part of the ui module, Go 1.27. Examples live in the examples/shop module, so decimal never enters the main module's dependencies.
2. **Compiler spike** (M0.3). One throwaway file that proves or disproves each of these, with the result written to docs/status.md:
   - a generic method As[U] on Field[T] that returns *Field[U];
   - explicit instantiation on a method call, f.Number[int32]("n");
   - inference of PT in Parsed;
   - f.Int("n").Is(rule.Min(0)) compiles without a type argument;
   - a method value used as a function, is := f.Type.Eq;
   - Enum[E] with E constrained by ~string plus option.SelectOption accepts a models-style enum;
   - f.Bool("b").Is(rule.MaxLen(5)) fails to compile.

   Where one fails, apply the fallback in section 5.1 and record it.
3. **forms/rule** (F1). Rule[T], New, Err, Check, Attrs, the catalogue in section 6.2 and its default errors. Table tests from the salvaged cases.
4. **Core** (F2). Form, Field[T], Source, the non-generic constructors, presence, Required, Is, Set, Value, Ptr, Bind, Validate, Submit, FieldError. Done when the store example passes.
5. **Conversion** (F3). Conv, As, Enum, Parsed, Number, Options.
6. **Conditions** (F4). Cond, When, evaluation, cycle detection, JSON, the conformance vectors.
7. **Nesting and view** (F5). Group, List, path binding, the template row, Control, Kind, formstest. Done when the promotion example passes.

### 13.2 Tests that must exist

| Area | Test |
|---|---|
| Compile-time safety | Packages under testdata/nocompile/ that must fail to build: a rule on the wrong field type, Eq with a value of another enum type, a rule for T applied after As to U |
| Declaration errors | Each panics with a message naming the field: duplicate name, malformed name, condition cycle, builder call after Bind, use of a handle after As, second Bind |
| Presence | Every row of the table in section 5.2, including a required checkbox and trimmed text |
| Redisplay | A number field given abc reports ErrInvalid and its control's Text() is abc |
| Precision | 0.10000000000000000001 binds into decimal.Decimal and formats back unchanged |
| Options | models' All<Type>Options() passes to Options unchanged; a value outside it reports ErrNotAnOption |
| Conditions | A required field under a false When is not required; a stale value under a false When is cleared; the vectors in testdata/cond/ pass |
| Lists | Sparse indices 0 and 3 bind as two items and renumber to 0 and 1; an error lands at constraints[1].min_amount; item 257 is refused |
| Errors | errors.Is(f.Err(), rule.ErrMaxLen), errors.All(f.Err()) and errors.T(ctx, field.Err()) in two locales, through i18ntest |
| Labels | formstest reports every control whose label is still its wire name, so every visible text of a form can be an i18n key |
| API hygiene | No exported signature contains any or interface{}; go list -deps of forms shows no templ, web framework or decimal |
| Examples | Both forms in section 11 compile and their handlers' paths are exercised through formstest |

### 13.3 Working rules

- zp is read-only and private. Read it for behaviour; write only in this repository, and keep zp's names out of it.
- Do not add a generator, a genx.meta.yaml or struct tags.
- When the compiler forces a signature to change, keep the invariants in section 3, log the change with its reason, and amend interface.md.

## 14. Decisions on the open questions

The first draft listed nine choices made without epistemer's explicit decision. On 2026-11-21 epistemer decided 1 and accepted the defaults of 2 to 9.

| # | Question | Decision | What reversing it costs |
|---|---|---|---|
| 1 | Do rules carry built-in errors, or is the error always a parameter as in .Is(rule.Max(255, err))? | Built-in, overridable with .Err(err) (section 6.3). epistemer | Every catalogue constructor gains an err parameter and every form declares its own errors, as zp does |
| 2 | How do nested fields travel? | Path names such as constraints[0].type (section 8.1) | Client-built JSON in a hidden input needs a JSON Source and brings back number coercion in the browser |
| 3 | Is text trimmed? | Yes, with .Raw() to opt out (section 5.2) | One line per constructor |
| 4 | What does Int return? | Field[int], so rule.Min(0) infers (section 5.1). sqlc's integer columns are int32 and int64, so a field set from a model uses Number[int32] or Number[int64] | Field[int64] matches sqlc's ids but needs rule.Min[int64](0) everywhere |
| 5 | One error per field, or all of them? | One: the first failure (section 9) | Err() on a field becomes a join and widgets print a list |
| 6 | Do database rules run when another field has already failed? | Yes. Only the failing field short-circuits, as in zp | A two-pass Validate that runs context rules only when every cheap rule passed |
| 7 | In which location are dates and times parsed? | UTC. Nothing in neozp carries a time zone yet | A location parameter on Time and Date, or one read from the context |
| 8 | Is rule a subpackage or its own module? | Subpackage ui/forms/rule (section 3) | A module of its own |
| 9 | Do conditions need gt, lt, gte, lte? | Not in v0.1 (section 7.3) | Four more builders and four more operators in the client evaluator |

**Deferred, not open.** Async or remote client validation, and multi-step forms, are out of v0.1; nothing in the design blocks them.

**Dropped.** A JSON source for API endpoints. handlers serves the API through oapi-codegen's strict server, which binds and validates JSON bodies against the OpenAPI file (handlers ADR 0003), so forms has no API caller.

## 15. Changes since the first draft

Applied on 2026-11-21, after checking the draft of 2026-11-16 against neozp as it stands:

- **Module.** forms is github.com/epistemer/ui/forms, in the ui module, not a repository of its own (epistemer).
- **Choices.** models v0.3.0 (amendment A3, ADR 0006) made option.SelectOption the interface UI packages consume. EnumValue is ~string plus SelectOption (sqlc's Valid() is no longer required); Options takes SelectOption values; Option[T] is gone; options carry over through As (epistemer, sections 5.1 and 5.3).
- **Error keys** follow errors' errors.<area>.<name>: errors.forms.rule.<name> and errors.forms.<name> (section 6.3).
- **Handlers.** The examples use handlers' net/http admin handlers, //h: directives and generated URL builders instead of echo.
- **auth.** CSRF is auth's (no token on a form, auth D13); a password rule can delegate to Auth.CheckPassword (section 6.4).
- **Labels.** formstest checks that every control has a label of its own (section 13.2).
- **Public repository.** zp's file names, identifiers and the salvage guide's paths are out of the document; the worked examples use the toy shop (sections 11, 12).
- **Dropped** the JSON source (section 14).
