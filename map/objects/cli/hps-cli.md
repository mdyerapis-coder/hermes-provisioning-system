---
type: object
cluster: cli
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/cli/cli.go
---

# HPS CLI — command dispatch

`hps` (product name) = `internal/cli` package, entry via `cmd/hps/main.go`. A single `Run` function dispatches subcommands by string match.

## Why this shape

A flat switch in `Run` keeps every subcommand's flag parsing local to its own `run*` function, so adding a command means adding one case + one function — no registry, no reflection. Matches the project's "no framework yet" posture (`docs/architecture.md`).

## Shape

- `internal/cli/cli.go` — `BuildInfo`, `Run(args, build, stdout, stderr) int`; dispatches `validate`, `doctor`, `serve`, `manifest`, `asset`, `android`
- `internal/cli/android.go` — `runAndroid` subcommands: `validate`, `stage`, `channel`
- `internal/cli/android_distribution.go` — staging/channel command bodies
- `internal/cli/asset.go` — `asset verify`
- `internal/cli/manifest.go` — `manifest validate`
- Entry point: `cmd/hps/main.go` calls `cli.Run(os.Args[1:], ...)`

Citations: `internal/cli/cli.go:29`, `cmd/hps/main.go:15`

## Connected to

- **owns:** nothing (calls into other packages, holds no state)
- **owned-by:** `cmd/hps/main.go`
- **joins:** `internal/validate`, `internal/server`, `internal/manifest`, `internal/assets`, `internal/androidapk`, `internal/androiddist`
- **looks-like-but-is-not:** not a command framework — plain `flag.FlagSet` per subcommand, no cobra/urfave

## If you change this

- **Hits:** whichever `internal/*` package backs the new/changed subcommand; `README.md` command list; `internal/cli/cli_test.go`
- **Does not hit:** `internal/server` HTTP routes (server owns its own routing; CLI only starts it)

## Surfaces

| Surface | Role |
|---|---|
| operator (terminal) | invokes `hps <command>` |
| `internal/cli/cli_test.go` | tests dispatch |

## See

- Source: `internal/cli/cli.go`
