# Intermediate track

Where interviewers separate "wrote some Go" from "understood Go."
Expect follow-ups two levels deep on each answer.

## Interfaces

**Q: What does an interface value look like in memory?**

Strong answer: a two-word header — (type pointer, value pointer/word).
Assigning a concrete value into an interface may allocate (the value
escapes into the header). The consequences: nil-checking gotchas (next
question), reflection's power, and why huge structs in interfaces
churn GC. — [04-functions-methods-interfaces](../04-functions-methods-interfaces/) (when it ships)

**Q: Why does nil behave strangely with interfaces?**

Strong answer: an interface holding a typed nil pointer is non-nil —
the type word is set. `var err error = (*MyErr)(nil); err != nil` is
true. Failure mode: a function returning `(*MyErr)(nil)` as `error`
defeats every `err != nil` check. Fix: return literal nil; the type
conversion is where the bug is born. — [08-faq-notes](../08-concurrency/08-faq-notes.md)

**Q: Why are interfaces satisfied implicitly?**

Strong answer: decoupling — providers never import consumers, so
behavior can be attached to types you don't own (io.Writer across the
ecosystem). Interfaces declared at the *consumer*, small (1-3 methods),
satisfied by the types that arrive.

**Q: Method sets — when does a type satisfy an interface?**

Strong answer: value receivers join both T and *T's method sets;
pointer receivers join only *T's. So `var r io.Reader = buf` compiles
for *bytes.Buffer but a value receiver type only satisfies via its
value. Follow-up: why Go rejects calling pointer-receiver methods on
addressable values sometimes vs others — addressability rules.

## Pointers

**Q: When pointer receiver vs value receiver?**

Strong answer: pointer when mutation is needed or the type is big /
contains a mutex / must be non-copyable; value for small, immutable,
value-semantic types. Consistency per type — don't mix on one type.
The senior note: value receivers can be *faster* (stack, no indirection
for escape analysis); don't default to pointers reflexively. —
[02-go-language](../02-go-language/) (when it ships)

**Q: Show a bug where a loop variable and a pointer interact.**

Strong answer: pre-Go-1.22, `for _, v := range xs { go f(&v) }` took
the address of one shared variable — every goroutine saw the last
element. Fixed since 1.22 (per-iteration variables) but interviewers
want both worlds and the `v := v` historical workaround. —
[06-control-flow](../01-go-fundamentals/06-control-flow.md)

## Errors

**Q: %w vs %v; errors.Is vs errors.As vs `==`.**

Strong answer: %w wraps (inspectable chain), %v formats (identity
lost). `==` checks surface only; `errors.Is` walks the chain for a
value; `errors.As` walks it for a type and extracts. Failure mode of
`==`: one `fmt.Errorf("...: %w")` away from broken. —
[05-errors](../05-errors/01-errors-are-values.md)

**Q: Design the error model for an HTTP service.**

Strong answer: three families (validation/domain/infra), classified at
creation, mapped at the boundary, logged once. The grade is in
"mapped at the boundary" — domain layers never import net/http. —
[02-error-design](../05-errors/02-error-design.md)

## Generics

**Q: When do generics earn their place? When not?**

Strong answer: yes — same algorithm across types with identical code
(containers, slices/maps utilities, constrained math). No — single
concrete use, faking inheritance, avoiding interface design. The
comparison they want: interface{} (erases type, runtime assertions)
vs generics (compile-time checked, no boxing) vs concrete types
(simplest; duplicate when few). — [07-generics](../07-generics/) (when it ships)

**Q: What's a constraint? What's comparable?**

Strong answer: a constraint is the interface-like type-set a type
argument must satisfy (`[T constraints.Ordered]`). `comparable` = types
supporting ==. Follow-up: why can't a map key be `any`? (any isn't
comparable — map keys must be.)

## Modules & tooling

**Q: go.mod vs go.sum? internal/?**

Strong answer: go.mod declares requirements + versions; go.sum pins
cryptographic content for reproducibility. internal/ is
compiler-enforced privacy — importable only within its parent tree.
— [03-program-structure](../01-go-fundamentals/03-program-structure.md)

**Q: What does go vet catch that tests don't?**

Strong answer: printf-format mismatches, lock copies, unreachable
code, struct tags. Tests check behavior you wrote tests for; vet
checks patterns across everything.

## Testing

**Q: Table-driven tests — why did they win in Go?**

Strong answer: adding a case is a row, failures name themselves via
t.Run, and the table documents the spec including edge cases. Follow-up:
when not to — heterogeneous setups forced into one shape. —
[01-fundamentals](../10-testing/01-fundamentals.md)

**Q: Fake vs mock — pick one for a payments dependency and justify.**

Strong answer: fake (in-memory, contract-faithful, including error
sentinels) for behavior tests; recording mock only where the
interaction IS the requirement (charge submitted exactly once). —
[02-doubles-and-httptest](../10-testing/02-doubles-and-httptest.md)

## Live-code drills

1. Implement `Map`, `Filter` over slices — first with any/interface{},
   then generics; discuss the tradeoffs live.
2. Write `func MultiError(errs ...error) error` that Is/As traverse
   correctly (know errors.Join exists — say it).
3. Make a non-thread-safe cache safe three ways (mutex, sharded,
   sync.Map) and say when each wins.
