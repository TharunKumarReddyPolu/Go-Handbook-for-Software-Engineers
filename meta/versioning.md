# Interpreting version-specific content

Go releases every six months (currently February and August). Go is famous
for its compatibility promise: almost all programs keep compiling forever.
That promise is why this handbook can teach stable concepts confidently,
but it also means some claims need a version stamp.

## How this handbook handles versions

1. **The default.** Chapters describe behavior that has been stable for
   years: slices, maps, goroutines, channels, interfaces, `database/sql`,
   `net/http`. These need no version stamp.
2. **Version stamps.** When a feature or behavior is tied to a specific
   release, chapters say so explicitly, using the form "Introduced in Go X".
   Examples: generics (Go 1.18), loop-variable semantics (changed in
   Go 1.22), generic methods (Go 1.27), `for range f` iteration over
   function iterators (Go 1.23), `sync.WaitGroup.Go` (Go 1.25).
3. **The go directive.** Every module in this handbook declares its minimum
   version in `go.mod` via the `go` directive. A file can only use features
   from that version or earlier; the `go` command enforces this for standard
   library symbols. When you copy an example into your own project, check
   your `go.mod` first.

## A note on generic methods

Generic methods were introduced in **Go 1.27** (August 2026): a method may
now declare its own type parameters, e.g. `func (r *Ring[T]) Map[U any](f
func(T) U) Ring[U]`. This is new enough that:

- older tutorials and AI training data will not mention it, and may even
  deny it exists;
- interface methods may **not** declare type parameters, and interface
  methods cannot be implemented by generic methods: that restriction is
  unchanged;
- most production code will not need them for years. Prefer plain generic
  functions until a method-local type parameter genuinely removes friction.

## When you read older material

Much of the best Go writing predates several of these changes. A non-exhaustive
list of behavior that differs by version:

| Topic | Old behavior | New behavior | Since |
|---|---|---|---|
| Loop variables | One variable per loop, shared by closures (classic capture bug) | Fresh variable per iteration | Go 1.22 |
| `time` timer channels | Buffered; stale values possible | Unbuffered, synchronous | Go 1.23 |
| Range over integer | Not available | `for i := range n` | Go 1.22 |
| Range over iterator func | Not available | `for x := range seq` | Go 1.23 |
| `sync.WaitGroup` | `Add`/`Done`/`Wait` only | Additional `wg.Go(func(){...})` helper | Go 1.25 |
| Generic methods | Not available | Methods may declare type parameters | Go 1.27 |

## Rules of thumb for contributors

- Cite the version when behavior changed, was introduced, or was removed.
- Verify claims against the release notes for that version
  (<https://go.dev/doc/devel/release>) before writing them down.
- Do not write "as of Go 1.x" for stable, long-standing behavior: it
  invites silent rot.
- If unsure whether a claim is version-dependent, test it with the toolchain
  and cite the test in the PR description.
