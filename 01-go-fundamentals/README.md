# 01 · Go Fundamentals

The starting point of the handbook. This section builds the vocabulary,
the toolchain habits, and the mental model that every later chapter assumes,
including how Go looks from inside Java, Python, C++, or JavaScript.

## Objectives

By the end of this section you can:

- Explain why Go exists and what it optimizes for
- Drive the toolchain: `go run`, `go build`, `go install`, `go env`, `go mod`
- Structure a program and a module correctly
- Predict the zero value of any type without looking it up
- Write control flow that a Go reviewer would not rewrite
- Use defer/panic/recover correctly, and know when not to use them
- Map your current language's habits onto Go's, and unlearn the ones that do not transfer

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Why Go exists](01-why-go-exists.md) | The problem Go was built to solve |
| 2 | [Toolchain & workflow](02-toolchain-and-workflow.md) | go run, go build, go env, the daily loop |
| 3 | [Program structure & modules](03-program-structure.md) | package main, go.mod, project layout |
| 4 | [Variables, types & constants](04-variables-types-constants.md) | declaration, inference, conversions |
| 5 | [Zero values](05-zero-values.md) | Go's most under-taught design decision |
| 6 | [Control flow](06-control-flow.md) | if, for, switch, range: all of Go's loops |
| 7 | [Defer, panic & recover](07-defer-panic-recover.md) | ordered cleanup, the panic contract |
| 8 | [Coming from other languages](08-coming-from-other-languages.md) | Java / Python / C++ / JS → Go |

## Examples

Runnable code lives in `examples/`:

- `examples/hello/main.go`: smallest real program with a module
- `examples/zero/main.go`: printing the zero value of every core type


