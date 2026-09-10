# 26 · Go Interview Preparation

Organized by difficulty, graded on reasoning rather than recall. Every
answer here points back to the handbook chapter with the depth — this
section is the *revision layer*, not the source.

## How to use this section

- Answers that only name facts are junior answers. Answers that name
  the fact, the *why*, and the failure mode it prevents are senior
  answers.
- Scenario questions ("What would you do if...") are graded on
  process: clarify constraints, measure, propose, and state tradeoffs —
  not on jumping to an implementation.
- Interviewers probe depth with follow-ups ("why?", "what breaks?",
  "how would you prove it?"). Prepare the second layer of every answer.

## Chapters

| # | Chapter | Level |
|---|---|---|
| 1 | [Beginner track](01-beginner.md) | syntax, types, slices, maps, functions, structs |
| 2 | [Intermediate track](02-intermediate.md) | interfaces, pointers, errors, generics, modules, testing |
| 3 | [Advanced track](03-advanced.md) | goroutines, channels, sync, memory, GC, scheduler, races |
| 4 | [Senior track & scenarios](04-senior-scenarios.md) | architecture, reliability, scalability, production debugging |

## Question format

Each question shows a strong answer skeleton — the structure a strong
candidate fills in live:

> **Q:** Why does nil behave strangely with interfaces?
>
> **Strong answer:** An interface value is a (type, value) pair. A nil
> interface has neither; a non-nil interface holding a typed nil pointer
> has a type, so it is not nil. The failure mode: passing a typed-nil
> into an interface-returning function surprises `err != nil` checks.
> The fix: return literal nil; check at value level before wrapping.
> — depth in [08-concurrency/08-faq-notes](../08-concurrency/08-faq-notes.md)

## Cross-references

- Live-code drill targets: [08-concurrency examples](../08-concurrency/) (stages 1–5)
- Benchmark/profiling stories: [19-performance](../19-performance/)
- Design exercises: [24-system-design](../24-system-design/) (when it ships)
- Distributed-systems depth: [16-distributed-systems](../16-distributed-systems/) (when it ships)
