# Program structure & modules

## Why Does This Matter?

Everything you build starts from the same skeleton: a module, a `main`
package, and an import path that means something. Getting this right in
week one prevents the reorganization everyone else does in year two.

## Mental Model

A **module** is the unit of versioning and dependency management; it is
announced by a `go.mod` at its root. A **package** is the unit of
compilation and visibility inside that module. A **repository** is just
storage: it may hold one module, many modules, or be part of nothing.

```text
repository (git)
└── module (go.mod, import path = github.com/you/project)
    ├── package main        ← executables (one per binary)
    ├── package payments    ← libraries, lowercase path, snake ok
    └── internal/...        ← compiler-enforced privacy
```

The import path is the contract: `github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/01-go-fundamentals/examples/hello`
is both *where it lives* and *how others import it*.

## The smallest real program

```go
// examples/hello/main.go
package main

import "fmt"

func main() {
	fmt.Println("hello, handbook")
}
```

Three mandatory facts, each a favorite interview warm-up:

1. `package main` + `func main()` is the executable entry point. The
   linker starts there.
2. Imports are per-file and unused imports are **errors**: dead imports
   are noise a team should not pay for.
3. `main` is called on the main goroutine and returns nothing; to exit
   with a status use `os.Exit`, which skips defers (see
   [defer chapter](07-defer-panic-recover.md)).

## go.mod: the module contract

```go
module github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers

go 1.27
```

- `module` declares the import path prefix for everything below.
- `go 1.27` sets the language/stdlib floor. The `go` command uses it to
  reject features your floor lacks (see [meta/versioning.md](../meta/versioning.md)).
- Dependencies appear as `require` blocks with versions; `go.sum` pins
  their cryptographic content.

Create one per module: `go mod init <path>`. Add/prune deps:
`go mod tidy`: run it, commit both files, always.

## Internal packages

Any directory named `internal` gets a special rule enforced by the
compiler: only code rooted at the parent of `internal` may import it.

```text
mycompany/
├── go.mod                  (module mycompany/platform)
├── cmd/api/main.go         ✓ can import mycompany/platform/internal/auth
└── internal/auth/auth.go   ← private to this module
```

Other repositories trying to import `internal/...` fail to compile. This
is how Go says "public API" vs "not yours": capitalization covers files,
`internal` covers trees. Use it aggressively: most packages in a real
service should be internal.

## Where does main.go live?

Common production layout (not mandated; see the layout debate note below):

```text
service/
├── cmd/api/main.go        ← one binary
├── cmd/worker/main.go     ← another binary
├── internal/
│   ├── domain/            ← types + business rules, no I/O
│   ├── postgres/          ← repository implementations
│   └── httpapi/           ← handlers, middleware
├── go.mod
└── README.md
```

`cmd/` is convention, not magic: the compiler does not know it exists.
The layout matters because *imports point inward*: `cmd` and `internal/*`
may import `domain`; `domain` imports neither. That one-way flow is what
keeps a service testable for years. The full treatment is
[14-backend-development](../14-backend-development/).

## Comments and naming: the 20% that matters

- **Comments are prose, complete sentences.** Doc comments start with the
  name: `// Load returns the cached value for key.`
- **Exported = documented.** Anything another package can call deserves a
  doc comment; `go doc` and pkgsite render it.
- **Names are short and local.** `i` in a loop, `r` for a reader within a
  function, `readerTimeout` in config. Long names do not add clarity;
  scope does.
- **MixedCaps, no underscores.** `HTTPServer`, `userID`: initialisms are
  capitalized uniformly.
- **Get it? No `Get` prefix.** `user.Name()`, not `user.GetName()`,
  getters are unidiomatic; return values directly.

## Common Mistakes

- **One giant `main.go`.** Wiring, business logic, and HTTP in one file
  can be fine for a CLI, never for a service.
- **`utils` or `common` packages.** Names that mean nothing attract
  everything. Name packages for what they provide: `timeutil` beats `utils`.
- **Cyclic imports.** A design smell; break cycles by extracting the
  shared types into a lower package.
- **Reorganizing into `models/`, `handlers/`, `controllers/`**: importing
  your framework's vocabulary instead of your domain's.
- **Forgetting `go mod tidy` after refactor.** CI catches it; make it
  muscle memory first.

## Idiomatic Go

```go
// Package ratelimit provides a token-bucket limiter for outbound calls.
package ratelimit

import (
	"context"
	"time"
)

// Limiter allows at most rate events per interval.
type Limiter struct {
	rate     int
	interval time.Duration
}

// Allow reports whether an event is permitted, waiting at most the
// context's remaining budget.
func (l *Limiter) Allow(ctx context.Context) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
		return l.tryTake(), nil
	}
}

func (l *Limiter) tryTake() bool { return true } // stub; see 08-concurrency
```

Note what is absent: no constructor ceremony, no interface declared
before a second implementation exists, doc comments on every exported
symbol, context as the first parameter.

## Performance Considerations

- Package layout affects build granularity: fewer, larger packages
  recompile less in big repos than hundreds of micro-packages.
- Import cycles are impossible, so refactors that add cycles fail fast at
  compile time: use that as a design check.

## Testing Strategy

Tests live next to code (`auth_test.go` beside `auth.go`), same package
by default. This handbook's CI runs `go test ./... -race`; the pattern is
introduced properly in [10-testing](../10-testing/).

## Interview Questions

1. *What is the difference between a module and a package?*: Module:
   versioned dependency unit (go.mod). Package: compilation and visibility
   unit. A module contains packages.
2. *How does `internal` enforce privacy?*: The compiler (not the VCS)
   rejects imports from outside the `internal`'s parent tree.
3. *Why are unused variables and imports errors?*: Both correlate with
   bugs; the language treats dead code as a defect, and it keeps builds
   incremental.

## Practice Exercises

1. Create a scratch module with `go mod init scratch`, add an
   `internal/secret` package, and try importing it from a second module to
   watch the compiler refuse.
2. Write a doc comment for a function, then run `go doc` on it and read it
   as a user would.
3. Rename three names in an old project of yours to be shorter because
   their scope shrank.

## Further Reading

- [Organizing a Go module](https://go.dev/doc/modules/layout): official
- [Effective Go: names](https://go.dev/doc/effective_go#names)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): naming and doc-comment norms
