# Contributing to the Go Handbook

Thanks for helping improve this handbook. It succeeds when explanations are
accurate, examples run, and a reader can trust every claim. This guide makes
contributing predictable for both sides.

## Ways to contribute

| Type | Example | Size |
|---|---|---|
| Fix an inaccuracy | A wrong claim about map growth or GC pacing | Small |
| Improve an example | Tighter, more idiomatic, better names | Small |
| Add exercises or interview questions | To an existing chapter | Small |
| Expand an outline section | Write chapters for a section marked [Outline] | Large: open an issue first |
| New section | Something the roadmap does not cover | Discuss first, almost certainly |

For anything beyond a small fix, open an issue using the
[content gap template](.github/ISSUE_TEMPLATE/content-gap.md) before writing.
The handbook deliberately says no to topics that do not earn a full chapter.

## Before you start

1. Read [README.md](README.md): especially "The philosophy" and the chapter
   contract.
2. Skim two or three existing chapters to absorb the format and tone.
3. Check [ROADMAP.md](ROADMAP.md) so planned work does not collide.
4. For style questions, this file is the authority; where it is silent,
   imitate existing chapters.

## The chapter template

Chapters use a consistent structure so readers always know where to look.
Not every section must exist for every topic, but the order is fixed:

```markdown
# Concept Name

## Why Does This Matter?

## Mental Model

## How It Works

## Syntax / API

## Basic Example

## Real-World Example

## Production Example

## Common Mistakes

## Idiomatic Go

## Performance Considerations

## Concurrency Considerations

## Security Considerations

## Testing Strategy

## Interview Questions

## Practice Exercises

## Further Reading
```

Every chapter **must** end with Further Reading. Link only authoritative
sources: go.dev, official project documentation, official repositories.
Never fabricate a reference; if you cannot verify a link, do not include it.

## Writing standards

- **Engineer to engineer.** Direct, concrete, no filler, no motivational
  fluff, no wall-of-text. Short paragraphs and sentences.
- **Answer why before how.** Every concept explains the problem it solves
  before the syntax.
- **Original text.** Do not copy or closely paraphrase Go documentation,
  Effective Go, blog posts, tutorials, or other handbooks. Explain it your
  own way; link the source under Further Reading instead.
- **Honest comparisons.** When Go differs from Java, Python, C++, or
  JavaScript, say what is genuinely better, what is worse, and what is just
  different. No advocacy.
- **Tables and Mermaid diagrams** where they carry weight (comparisons,
  lifecycles, architecture). Never decorative.
- **Version stamps.** See [meta/versioning.md](meta/versioning.md). Use
  "Introduced in Go X" for version-dependent behavior; leave stable behavior
  unstamped.
- **Terminology consistency.** Use the definitions in
  [GLOSSARY.md](GLOSSARY.md). If you need a new term, add it there in the
  same PR.

## Code standards

All Go examples must:

- Compile (`go build ./...` must pass with your file included).
- Be gofmt-formatted; CI enforces it.
- Be `go vet`- and staticcheck-clean.
- Follow idiomatic Go: small interfaces, errors as values, `context` for
  cancellation, no Java-style hierarchies.
- Handle errors explicitly. An example that ignores an error teaches a
  reader to ignore errors.
- Prefer the standard library. Third-party imports need a stated reason in
  the surrounding prose (see 18-kafka-with-go for the expected style of
  justification).
- Be runnable or clearly labeled otherwise. Add a file-level comment
  `// This snippet is intentionally incomplete.` for code shown for
  explanation only.
- Include tests when the logic is non-trivial. CI runs `go test ./... -race`;
  tests that need external services (Kafka broker, Postgres) must skip
  cleanly when the service is absent.

Example files live next to their chapter, typically under `examples/` within
the section, in package `main` for runnable programs or a named package for
testable logic.

## Verifying your changes

Run all of these before opening the PR: CI runs the same checks:

```bash
gofmt -l .                 # must print nothing
go build ./...             # everything compiles
go vet ./...               # no vet findings
go test ./... -race        # all tests pass with race detection
go run ./tools/linkcheck   # internal links resolve
```

Install staticcheck once and run it locally:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

## Content audit checks

Three checks go beyond compilation. They are the same ones the
repository-wide audit uses; run them before opening a PR.

**1. Internal links.** Already in the block above, but it is the audit's
first gate, so it bears repeating:

```bash
go run ./tools/linkcheck   # every internal link must resolve
```

**2. Em-dash scan.** The handbook uses no em dashes (U+2014), anywhere:
prose, code comments, examples. Rewrite the sentence instead; a colon,
semicolon, or parentheses almost always reads better.

```bash
pat=$(printf '\xe2\x80\94')
grep -rn --exclude-dir=.git "$pat" --include="*.md" --include="*.go" .   # must print nothing
```

**3. Version-stamp sweep.** If your change touches version-dependent
behavior, list the stamps in the files you edited and re-verify each one
against the release notes for that version:

```bash
grep -n "Introduced in Go\|since Go 1." 14-backend-development/   # your files here
```

Then confirm each claim at [go.dev/doc/devel/release](https://go.dev/doc/devel/release),
and check the feature you used is at or below the `go` directive in
`go.mod`. Stamp conventions live in [meta/versioning.md](meta/versioning.md).
If your claim is the newest kind ("Introduced in Go X"), consider whether
a compiled example can pin it empirically, the way `Ring.Map` pins the
generic-methods claim in
[03-data-structures/examples/ring](03-data-structures/examples/ring/ring.go):
a toolchain change that invalidates the claim then fails CI instead of
silently rotting the prose.

## Git workflow

1. Fork, then create a branch: `git checkout -b fix/concurrency-leak-example`.
2. Make focused changes. One chapter per PR keeps review fast.
3. Commit with clear messages: `fix: correct channel ownership example in 08-concurrency`.
4. Fill in the [PR template](.github/pull_request_template.md) honestly.

## Review process

- A maintainer reviews for accuracy, originality, and fit: in that order.
- Technical corrections are usually merged quickly; new chapters may go
  through multiple rounds. That is normal and not a rejection.
- If a claim in review is disputed, the tie-breaker is evidence: a
  benchmark, the release notes, or a reproducible test. Cite the version.

## Reporting issues

- Inaccurate content: use the [bug template](.github/ISSUE_TEMPLATE/bug_report.md)
  and quote the exact sentence. Quote-based reports get fixed fastest.
- Missing content: use the [content gap template](.github/ISSUE_TEMPLATE/content-gap.md).

Thank you for making this better.
