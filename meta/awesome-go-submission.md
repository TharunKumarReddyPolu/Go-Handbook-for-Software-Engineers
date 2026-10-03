# awesome-go submission kit

Status: **ready, parked**. The PR opens after **2027-02-09**, when the
repository crosses awesome-go's 5-month history requirement (first commit
2026-09-09). Everything below was verified against
[avelino/awesome-go's CONTRIBUTING.md](https://github.com/avelino/awesome-go/blob/main/CONTRIBUTING.md)
in October 2026; re-check it before submitting, the rules change.

## Requirements scorecard (as of 2026-10-03)

| Requirement | Status |
|---|---|
| Public, accessible, not archived | pass |
| `go.mod` at root | pass |
| SemVer release (`vX.Y.Z`) | pass (`v1.0.0`) |
| pkg.go.dev page loads | pass (verified live) |
| Open source license | pass (Apache-2.0) |
| README + CI/CD | pass |
| Category minimum (3 items) | pass (Tutorials has ~40) |
| Coverage report link | pass (Codecov wired into CI) |
| 5 months of history | **clears 2027-02-09** |

## The entry

Insert into the `### Tutorials` section of awesome-go's `README.md`,
alphabetically between `Go database/sql tutorial` and `Go in 7 days`:

```md
- [Go-Handbook-for-Software-Engineers](https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers) - A practical handbook covering Go fundamentals through production, distributed, and financial systems, with runnable examples verified by CI.
```

Format rules honored: exact repo name as link text, description ends
with a period, non-promotional, one item per PR.

## The PR

Title: `Add Go-Handbook-for-Software-Engineers`

Body (their template plus the links reviewers ask for):

```md
Forge link: https://github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers
pkg.go.dev: https://pkg.go.dev/github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers
Coverage: https://app.codecov.io/gh/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers
```

## Submission steps

1. Fork `avelino/awesome-go`, create a branch.
2. Add the entry line at the position above; touch nothing else.
3. Open the PR against `avelino/awesome-go` with the title and body
   above, filling their PR template.
4. CI runs their blocking checks automatically (alphabetical order,
   single item, format, SemVer, pkg.go.dev reachability). Fix any red
   check before a maintainer looks.
5. Maintainers review manually: category fit, usefulness, description
   accuracy. They wait 15 days for interaction; answer promptly.

## After acceptance

Add the mentioned badge to this README's badge row:

```md
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)
```

## Maintenance obligations once listed

- Keep shipping: at least one SemVer release per year while active, or
  demonstrate stability with no stale bug reports.
- Keep the coverage link current (CI uploads on every push already).
- Respond to issues and PRs within roughly two weeks.
