# Beginner track

Warm-ups that filter out surface familiarity. Interviewers expect
crisp, correct answers: hesitation here is expensive.

## Syntax & types

**Q: What are the zero values of slice, map, channel, and pointer, and
what can you do with each?**

Strong answer: all are `nil`. A nil slice supports `len`, `range`,
`append`: reads are safe, no backing array. A nil map supports reads
(zero value returned) but panics on writes: declare with `make` or a
literal when you will write. A nil channel blocks forever on every
operation (feature in `select`, footgun elsewhere). A nil pointer
panics on dereference. The design reasoning: Go guarantees usable zero
values where possible; where it can't, it panics loudly instead of
misbehaving quietly.: [05-zero-values](../01-go-fundamentals/05-zero-values.md)

**Q: Why no implicit type conversions?**

Strong answer: explicit narrowing is reviewable: every potential loss
is visible in code. Cost: verbosity; benefit: an entire class of silent
precision bugs disappears. Mention constants: untyped constants
*do* adapt, which is why `var f float64 = 1<<40` compiles.

**Q: Arrays vs slices?**

Strong answer: array is a fixed-size value: length is part of the
type, assignment copies. Slice is a (pointer, len, cap) header over a
backing array: assignment shares the backing array. Growth: `append`
may allocate a new array when cap is exceeded. Follow-up they'll ask:
what does passing a slice to a function let the callee do? (Mutate
shared backing array; also `append` past cap replaces the caller's
view: hence "slices are not reference-safe under growth.")

## Maps

**Q: Why is map iteration order randomized?**

Strong answer: deliberate: the runtime shuffles the start so programs
cannot accidentally depend on hash order, which changes across
runtimes/architectures. Sorted output: collect keys, `slices.Sort`,
range keys.

**Q: How do maps behave under concurrency?**

Strong answer: concurrent reads are fine; any write concurrent with
another read/write is a *fatal* runtime error (`concurrent map writes`,
not a recoverable panic). Fixes: mutex+map, sharded map, or sync.Map
for its two niche shapes. Follow-up: why fatal instead of panic?,
because detecting it generally is undecidable; corruption is already
possible.

## Functions & structs

**Q: Multiple return values: show the idiom and the trap.**

Strong answer: idiom is `(value, error)` with zero value on failure.
Trap: named returns + `defer` can silently overwrite; and `:=`
shadowing in if-init clauses creates a *new* err that never gets
checked.

**Q: Struct embedding vs inheritance?**

Strong answer: embedding promotes fields/methods: no subtyping, no
virtual dispatch, no super. The embedded value is a field; method
resolution is compile-time name promotion. "Inheritance in Go" is a
category error; composition with interfaces is the replacement.,
[04-functions-methods-interfaces](../04-functions-methods-interfaces/) (when it ships)

## Rapid-fire

| Q | One-line answer |
|---|---|
| `:=` vs `var`? | `:=` declares+infers inside functions; `var` everywhere, needed for zero values and specific types |
| const vs var? | compile-time vs runtime; `iota` for enumerations |
| string vs []byte? | immutable UTF-8 bytes vs mutable bytes; conversions copy |
| for-range over string gives? | byte index + rune (not byte!) |
| defer order? | LIFO at function exit; args evaluated at defer time |
| new(T) vs &T{}? | equivalent for structs; &T{} with fields is idiomatic |
| switch fallthrough? | absent by default; keyword exists but usually a smell |
| make vs new? | make for slices/maps/channels (initialized); new for any T (zeroed pointer) |

## Live-code warm-ups

1. Reverse a slice in place. They're checking: two-pointer, no
   allocation, and knowing `s[i], s[j] = s[j], s[i]`.
2. Merge two maps. Checking: read-into-existing vs allocate, and
   handling the "same key" policy question.
3. Count rune frequencies in a string. Checking: `for _, r := range s`
   vs byte indexing; map value increment on the nil map *read* side.

## Common failure patterns at this level

- Reciting "slices are references": they're value headers over shared
  backing arrays; the distinction has real bugs attached.
- Writing `if err != nil { return err }` without wrapping: no context
  for the caller ([05-errors](../05-errors/)).
- Believing `defer` runs at block scope: it's function scope.
e.
