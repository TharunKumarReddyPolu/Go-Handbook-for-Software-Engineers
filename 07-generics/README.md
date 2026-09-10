# 07 · Generics

**Status: outline — chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Decision-level content already in
[FAQ](../FAQ.md) and [26 §2](../26-go-interview-preparation/02-intermediate.md).

## Planned chapters

1. **Why generics exist** — the interface{} era's costs (erasure,
   runtime assertions, boxing); what the type-parameter model fixed
2. **Type parameters & constraints** — syntax, inference, type sets,
   `comparable`, `any`, operator constraints
3. **Generic data structures** — containers done right (and the
   built-ins that still win)
4. **Generic APIs** — designing constraints that read like
   documentation; stdlib `slices`/`maps` as the reference style
5. **When generics hurt** — readability as the constraint that wins;
   the "second use" rule; anti-pattern gallery
6. **Version notes** — generics since Go 1.18; **generic methods since
   Go 1.27** (methods may declare type parameters; interface methods
   still may not) — see [meta/versioning.md](../meta/versioning.md)

## Standing rule for this section

Generics are introduced *after* concrete and interface-based solutions
in every example, because the correct engineering question is never
"can I use generics?" but "did the duplication earn them?"
