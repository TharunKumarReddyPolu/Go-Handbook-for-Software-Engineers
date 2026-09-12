# Multiple & named returns

## Why Does This Matter?

Go functions return multiple values natively, and the entire error
handling model is built on that: `(value, error)` instead of exceptions
or out-parameters. The syntax is trivial; the engineering questions are
not: when to name results, how naked returns age, and how shadowing
turns a tidy `err` into a silent swallow of the wrong variable.

## Mental Model

A function's results are a tuple. The `:=` and `=` forms on the left
unpack it:

```go
f, err := os.Open(path)   // two results, both consumed
_, ok := m[k]             // deliberate discard
```

Result tuples are not first-class values: you cannot assign
`f, err` to one variable or pass the tuple around. Return exactly what
the caller unpacks.

**Named results** give the tuple components names in the signature:

```go
func Split(s string) (before, after string, found bool) {
    i := strings.Index(s, "=")
    if i < 0 {
        return "", "", false
    }
    return s[:i], s[i+1:], true
}
```

The names are documentation at the call site and interface definition
alike. This is their primary value.

## Naked returns: a short leash

Named results are pre-declared variables, so `return` can be bare:

```go
func parse(raw string) (u *User, err error) {
    u, err = decode(raw)
    if err != nil {
        return            // returns u (nil here), err
    }
    u.Name, err = normalize(u.Name)
    return
}
```

The rule of thumb that keeps this safe: naked returns are fine in
short functions (under ~10 lines, one screen), and in `defer` blocks
that repair a result. In anything longer they oblige the reader to
track variable assignments across the whole body. Long functions
should return values explicitly.

## Named results and defer: the repair pattern

The combination earns its complexity in exactly one place: deferred
closures that transform an outgoing error.

```go
func call(ctx context.Context) (err error) {
    defer func() {
        if errors.Is(err, ErrTransient) {
            err = fmt.Errorf("call failed after retries: %w", err)
        }
    }()
    err = attempt(ctx)
    return err
}
```

Without named results, `defer` cannot touch what the function returns.
This is the mechanism behind panic-to-error conversion too (see
[01-fundamentals](../01-go-fundamentals/README.md)): recover in a
deferred closure, assign to the named `err`.

## Shadowing: the silent killer

```go
func load(name string) (*Config, error) {
    cfg := &Config{}
    if err := json.Unmarshal(data, cfg); err != nil { // err is function-scoped
        return nil, err
    }
    if v := os.Getenv("MODE"); v != "" {
        cfg.Mode = v
    }
    if err := validate(cfg); err != nil { // new err each if-statement
        return nil, err
    }
    return cfg, nil
}
```

The trap version:

```go
var err error
if cond {
    err := doThing()     // := declares a NEW err, scoped to the if
    if err != nil {
        return err       // fine, returns the inner one
    }
}
return err               // BUG: outer err is always nil
```

`:=` redeclares in the inner scope whenever either side of the `:=` is
new. The compiler does not warn: both variables exist legitimately.
`go vet -shadow` (via `golang.org/x/tools/go/analysis/passes/shadow`)
flags it; code review must too.

## Common Mistakes

- **Naked returns in long functions**: result provenance becomes
  archaeology.
- **`:=` inside `if` when you meant `=`**: the shadowing trap above;
  the classic variant is `if err := f(); err != nil { return err }`
  followed by *using* `err` after the if (it is out of scope or the
  wrong one).
- **Naming results you never use**: `(data []byte, err error)` with an
  explicit return anyway: the names are then noise, and unflushed
  zero-values can leak out via naked returns.
- **Returning named nil pointers from constructors**: a typed-nil
  interface trap when the caller stores the result in an interface
  (see [06-interfaces-and-embedding](06-interfaces-and-embedding.md)).
- **Discarding errors with `_`** at call sites that matter: one
  underscore per deliberate decision, with a comment saying why.

## Idiomatic Go

```go
// Unpack with intent; keep the happy path flat.
conn, err := dial(ctx, addr)
if err != nil {
    return fmt.Errorf("dial %s: %w", addr, err)
}
defer conn.Close()

// Name results when the signature is the documentation.
func ParseDuration(s string) (d time.Duration, ok bool)

// Defer-repair for error decoration only, not control flow.
defer func() { err = errors.Join(err, closer.Close()) }()
```

## Performance Considerations

- Multiple small results are returned in registers on modern Go
  (register ABI, Go 1.17+); the tuple costs effectively nothing
  compared to allocating a result struct.
- A result struct may be clearer for many values, but weigh it: the
  tuple keeps the hot path allocation-free. Measure before wrapping
  (see [19-performance/01](../19-performance/01-measure-first.md)).

## Concurrency Considerations

Named results are single variables: two goroutines cannot share them,
but a closure (including the defer-repair pattern) both reads and
writes them. If you launch goroutines that assign to named results,
that is a race unless synchronized; prefer collecting via channels or
WaitGroup-guarded locals (patterns in
[08-concurrency/05-patterns](../08-concurrency/05-patterns.md)).

## Testing Strategy

Table-driven tests assert on *all* results, including the ok flag: the
comma-ok contract is part of your API. A test that only checks `err
== nil` misses the `found=false` path that callers rely on. The
`examples/seqops` package demonstrates asserting full result tuples.

## Interview Questions

1. *When are named results worth it?*: Documentation-heavy signatures,
   defer-repair patterns, and panic recovery; not as a license for
   naked returns in long functions.
2. *How does `:=` shadow?*: Re-declares in the inner scope when at
   least one name is new; both variables exist, the inner wins inside.
3. *Are multiple returns a tuple you can pass around?*: No: results
   unpack at the call site; first-class tuples would need a struct.
4. *Why does Go return errors instead of throwing?*: Errors are
   values in the signature: control flow is explicit, and the happy
   path is visibly interleaved with handling (see
   [05-errors](../05-errors/README.md)).

## Practice Exercises

1. Rewrite a long function that uses naked returns into explicit
   returns; count the bugs you find in the original (there is
   usually one).
2. Write a function `First[T any](s []T, pred func(T) bool) (T, bool)`
   and a test suite covering the not-found path explicitly.
3. Demonstrate the shadowing bug in a scratch file; fix it two ways
   (rename, and `=` instead of `:=`), and run the shadow analyzer on
   both.

## Further Reading

- [Spec: Return statements](https://go.dev/ref/spec#Return_statements)
- [Effective Go: multiple returns](https://go.dev/doc/effective_go#multiple-returns)
- [errors.Join (Go 1.20+)](https://pkg.go.dev/errors#Join)
