# Control flow

## Why Does This Matter?

Go has three loop forms, one conditional, and one multi-way branch,
deliberately fewer constructs than any mainstream language. The discipline
shows up in reviews: Go code has a recognizable shape because control flow
options are limited. This chapter covers the syntax *and* the shape.

## Mental Model

- **if**: condition, no parens, braces mandatory, may start with a statement.
- **for**: the only loop keyword; three forms cover while, C-style, and
  range.
- **switch**: no fallthrough by default (opposite of C); may switch on
  nothing (true-switch); `case` values need not be constants.
- **range**: iterate anything with a defined iteration: slices, maps,
  channels, strings, integers (Go 1.22+), and functions (Go 1.23+).

## if: with the initialization clause

```go
if n, err := strconv.Atoi(s); err != nil {
	return 0, fmt.Errorf("parse %q: %w", s, err)
} else if n > 100 {
	return 0, ErrTooBig
}
// n and err are out of scope here: deliberate
```

The init clause scopes the variables to the conditional: the idiomatic
way to keep `err` handling adjacent to the call that produced it. The
controversial `else` after an init clause is a signal: usually you want
early returns instead.

```go
// idiomatic: happy path unindented
if err != nil {
	return err
}
// ... continue
```

## for: all three forms

```go
// C-style
for i := 0; i < 10; i++ { ... }

// while
for x < 100 { x = step(x) }

// infinite
for { ... }  // exit via break, return, or panic
```

### range, and its version history

| Since | Form | Variables |
|---|---|---|
| forever | slice/array | `i`, or `i, v` |
| forever | map | `k`, or `k, v` (order randomized) |
| forever | string | `i, r`: r is a **rune**, not byte |
| forever | channel | `v`: receives until closed |
| Go 1.22 | integer | `i := range n`: 0..n-1 |
| Go 1.23 | function iterator | `k, v := range seq` |

Two footguns worth permanent memory:

1. **Map order is randomized** by design (the runtime shuffles the start)
   so nobody accidentally depends on it. Sort keys if you need order.
2. **Loop variable capture**: since Go 1.22, each iteration gets a fresh
   variable, so the classic "all closures see the last value" bug is
   fixed for modules with `go >= 1.22`. On older versions the bug is real
   and the fix is `v := v` inside the loop. Know both worlds; interviewers
   love this one (see [meta/versioning.md](../meta/versioning.md)).

## switch: no fallthrough, and the true-switch

```go
// expression switch
switch op {
case "add", "+":    // multiple values per case
	return add(a, b)
case "mul":
	return mul(a, b)
default:
	return nil, fmt.Errorf("unknown op %q", op)
}

// no condition = switch true: the clean if-else chain
switch {
case err != nil:
	return err
case retries > 3:
	return ErrRetriesExhausted
case ctx.Err() != nil:
	return ctx.Err()
}

// type switch: on interfaces
switch v := x.(type) {
case string:
	return len(v)
case []byte:
	return len(v)
case nil:
	return 0
default:
	return -1
}
```

Fallthrough exists (`fallthrough` keyword) but using it is a code smell
in Go: cases are semantically independent by default. `break` is implied
at each case end; `break label` exits an enclosing loop from inside a
switch, which is one of the few justified uses of labels.

## Common Mistakes

```go
// MISTAKE: ranging a map and appending: order will bite you in tests.
for k, v := range m {      // randomized order
	out = append(out, k+"="+v)
}
// idiomatic: collect keys, sort, then range keys.

// MISTAKE: modifying a slice while ranging it.
for i, v := range s {
	if v == 0 {
		s = append(s[:i], s[i+1:]...) // i now points past the wrong element
	}
}
// idiomatic: build a new slice, or range backwards.

// MISTAKE: shadowing in if-init chains (see ch. 04).
// MISTAKE: using for-range on a map when you meant "sorted output".
```

## Idiomatic Go

```go
// early return over else-nesting
func classify(code int) string {
	switch {
	case code < 300:
		return "ok"
	case code < 400:
		return "redirect"
	case code < 500:
		return "client error"
	default:
		return "server error"
	}
}
```

- Guard clauses return immediately; the happy path stays at indent level 1.
- A `switch` with 3+ conditions beats an if-else ladder.
- Labels: rare, named, justified. `break outer` with a comment why.

## Performance Considerations

- `for range` over a slice with index-only access (`for i := range s`)
  avoids copying elements: relevant for large structs; `for i, v := range`
  copies each element into `v` (in Go < 1.22 this variable was reused per
  iteration; since 1.22 it is fresh but still a copy).
- Maps have no cheap "in order" path: if you need sorted iteration on a
  hot path, maintain a sorted key slice instead.
- Switch on integer/string constants compiles to efficient dispatch;
  long if-else chains of function calls do not.

## Concurrency Considerations

`for range` over a channel is the receiving half of every pipeline:

```go
for v := range ch {   // ends when ch is closed
	process(v)
}
```

Forgetting to close the channel is the top cause of goroutine leaks
(covered in [08-concurrency](../08-concurrency/07-pitfalls.md)).

## Security Considerations

Unbounded loops that process untrusted input (`for { read() }`) need a
termination bound: max iterations, timeouts via context, or size caps,
otherwise a hostile input becomes a DoS. See [21-security](../21-security/).

## Testing Strategy

Test branch boundaries, not just branches: for `classify`, test 299, 300,
399, 400, 499, 500. Boundary bugs are the ones that reach production.

## Interview Questions

1. *How many loop constructs does Go have?*: One keyword, three forms;
   range over integers (1.22) and functions (1.23) are recent additions.
2. *Why is map iteration order randomized?*: To prevent accidental
   dependence on hash order; deliberate runtime shuffling.
3. *What changed about loop variables in Go 1.22, and what bug did it
   fix?*: Per-iteration variables; the classic closure-capture bug. Know
   the pre-1.22 workaround (`v := v`).

## Practice Exercises

1. Rewrite a nested if-else from your old code as a `switch` with
   guard clauses. Compare readability line by line.
2. Range over the string "héllo" with both byte and rune access; print
   indices. Explain why byte indices skip.
3. Write a function that iterates a map and returns keys in sorted
   order; add a table-driven test that would fail with unsorted output.

## Further Reading

- [The Go spec: for statements](https://go.dev/ref/spec#For_statements)
- [Range over function iterators](https://go.dev/blog/range-functions): Go 1.23
- [Go 1.22 release notes: loopvar](https://go.dev/doc/go1.22#language): per-iteration loop variables
