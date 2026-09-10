# 06 · Packages & Modules

**Status: outline: chapters are planned work (see
[ROADMAP.md](../ROADMAP.md)).** Foundations already covered in
[01 §3 Program structure & modules](../01-go-fundamentals/03-program-structure.md).

## Planned chapters

1. **Package design**: naming, cohesion, the internal/ tool, API
   surface minimization
2. **Go modules in production**: go.mod/go.sum mechanics, versioning,
   MVS (minimal version selection), replace/workspaces, private modules
3. **Multi-module repositories**: when the split earns itself; go.work
   workflows
4. **Dependency hygiene**: vendoring tradeoffs, dependency review,
   govulncheck in CI ([.github/workflows/security.yml](../.github/workflows/security.yml))
5. **Reproducible builds**: version-pinned toolchains, `-trimpath`,
   build provenance
6. **Semantic versioning for Go libraries**: v2+ path rules, the
   reality of breaking changes under the Go 1 promise
