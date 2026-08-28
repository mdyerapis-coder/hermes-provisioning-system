---
type: object
cluster: server
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/server/server.go
---

# Embedded HTTP Server

`New(cfg Config) (http.Handler, error)` builds the handler behind `hps serve` — health/status endpoints and repository asset serving, wrapped in a `securityHeaders` middleware.

## Why this shape

Binds `127.0.0.1:8080` by default (`docs/architecture.md`, "Network model") — LAN exposure requires an explicit CLI flag, not a server default, so the safe-by-default posture survives even if a caller forgets a flag.

## Shape

- `BuildInfo`, `Config` — `internal/server/server.go:13-25`
- `New(cfg Config) (http.Handler, error)` — `internal/server/server.go:26`
- `securityHeaders(next http.Handler) http.Handler` — `internal/server/server.go:66`

Citations: `internal/server/server.go:26`

## Connected to

- **owns:** nothing (stateless handler over the repository root it's given)
- **owned-by:** `internal/cli/cli.go` (`runServe`)
- **joins:** the repository filesystem tree (`/srv/hermes` by convention, `docs/architecture.md`)
- **looks-like-but-is-not:** not a reverse proxy — `docs/architecture.md` notes NGINX may still sit in front of it in production

## If you change this

- **Hits:** `internal/cli/cli.go` (`runServe`), `internal/server/server_test.go`
- **Does not hit:** `internal/androiddist` (server does not serve staged APKs in v0.1 — non-goal)

## Surfaces

| Surface | Role |
|---|---|
| `hps serve` | starts |
| health/status API clients | reads |

## See

- Source: `internal/server/server.go`
