# 05 · Error Handling

Go's most distinctive design decision, and the one that changes your code
the most coming from any exception-based language. Errors here are values:
part of every signature, visible at every call site, unavoidable.

## Objectives

By the end of this section you can:

- Design errors that carry context without losing their identity
- Choose between sentinel errors, typed errors, and opaque errors deliberately
- Wrap with `%w` and unwrap with `errors.Is`/`errors.As` correctly
- Map domain errors to HTTP responses without leaking internals
- Classify errors as retryable vs non-retryable and act on it
- Log errors without duplicating or destroying information

## Chapters

| # | Chapter | Focus |
|---|---|---|
| 1 | [Errors are values](01-errors-are-values.md) | The error interface, New, Errorf, wrapping, Is/As |
| 2 | [Error design in production](02-error-design.md) | Domain/validation errors, HTTP mapping, retryability, logging |

## Examples

`examples/` contains a working module used by both chapters:

- `examples/errorslib/`: a small error package: domain errors, wrapping,
  retryability classification, with tests
- `examples/service/`: a fake "payment" service mapping domain errors to
  HTTP responses, with tests


