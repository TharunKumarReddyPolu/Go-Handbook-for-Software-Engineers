# 06 · Packages & Modules

**Status: complete.** Six chapters covering the design and operations
of packages, modules, and dependencies. Every chapter follows the
standard contract (see [CONTRIBUTING](../CONTRIBUTING.md)). The
mechanics basics live in
[01 Section 3 Program structure & modules](../01-go-fundamentals/03-program-structure.md);
this section is the design and production layer.

## Chapters

1. **[Package design](01-package-design.md)**: cohesion over layers,
   naming calibration, `internal/` with teeth, API surface
   minimization, import discipline
2. **[Go modules in production](02-modules-in-production.md)**: MVS,
   the go/toolchain directives, replace vs go.work, private modules,
   the checksum database
3. **[Multi-module repositories](03-multi-module-repositories.md)**:
   the three layouts and their honest tradeoffs, sibling dependency
   resolution, path-filtered CI
4. **[Dependency hygiene](04-dependency-hygiene.md)**: the adoption
   review, vendoring tradeoffs decided, govulncheck's reachability
   analysis, the dependency budget
5. **[Reproducible builds](05-reproducible-builds.md)**: pinned
   toolchains, `-trimpath`, VCS stamping, provenance, container
   checklists
6. **[Semantic versioning for Go](06-semantic-versioning.md)**: v2+
   path rules, what breaks, the Go 1 promise, deprecation as a
   process
