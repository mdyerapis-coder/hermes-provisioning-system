---
type: object
cluster: validate
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/validate/validate.go
---

# Repository / Host Validation ("doctor")

`Repository(root string) Result` and `Endpoint(ctx, rawURL string) Check` — the checks behind `hps validate` and `hps doctor`: is the repo tree shaped correctly, is an optional provisioning server endpoint reachable.

## Why this shape

Returns a `Result` of individual `Check`s rather than a single bool/error, so the CLI can print every check's pass/fail instead of stopping at the first failure — the "doctor" pattern of surfacing everything wrong at once.

## Shape

- `Check`, `Result`, `(r Result) OK() bool` — `internal/validate/validate.go:15-27`
- `Repository(root string) Result` — `internal/validate/validate.go:40`
- `Endpoint(ctx, rawURL string) Check` — `internal/validate/validate.go:76`

Citations: `internal/validate/validate.go:40`

## Connected to

- **owns:** nothing
- **owned-by:** `internal/cli/cli.go` (`runValidate`, `runDoctor`)
- **joins:** nothing structurally (pure filesystem + HTTP checks)
- **looks-like-but-is-not:** not `internal/manifest.Validate()` — that validates one manifest's contents; this validates repository tree shape / host reachability

## If you change this

- **Hits:** `internal/cli/cli.go` (both `validate` and `doctor` read this), `internal/validate/validate_test.go`
- **Does not hit:** `internal/manifest`, `internal/androiddist` (different validation domains)

## Surfaces

| Surface | Role |
|---|---|
| `hps validate` / `hps doctor` | invokes directly |

## See

- Source: `internal/validate/validate.go`
