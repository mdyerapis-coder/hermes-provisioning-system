# Hermes Provisioning System — system map

A walkable graph of this repository so a cold agent can answer "what is X" and "what else moves if I change X" without slurping the tree. The subject tree is the source of truth; this map cites it and never becomes a second spec.

## Universes

| Universe | Meaning |
|---|---|
| **live** | In force. Implement and cite against these. |
| **leftover** | Still present, no longer the main path. Touch only if that path is in scope. |
| **ghost** | Named or filed, not wired (planned packages in docs/architecture.md that don't exist yet: `internal/kickstart`, `internal/bootstrap`, `internal/approval`, `internal/evidence`, `internal/recovery`, `internal/pxe`). Do not implement against these. |

## Name collisions

| Product term | Code/file name |
|---|---|
| the CLI | `hps` binary, `cmd/hps/main.go` + `internal/cli` |
| the manifest (OS provisioning) | `Manifest`, `internal/manifest/manifest.go` |
| the manifest (Android staging) | `StageManifest`/`Release`, `internal/androiddist/types.go` — a **different** manifest family, do not conflate |
| doctor / validate | `internal/validate` (repo tree + host/endpoint checks — not manifest content validation) |

## How to walk

1. Open `map/CLAUDE.md` (this file) — the L0 catalog.
2. Open `map/objects/_index.md` — one line per noun.
3. Open the object card for the noun you are changing.
4. Open `map/effects/CONTEXT.md` — what else moves.

## Routing

| If you are changing | Open |
|---|---|
| CLI dispatch / flags | `map/objects/cli/hps-cli.md` |
| OS provisioning manifest | `map/objects/manifest/provisioning-manifest.md` |
| Android APK validate / stage / channel | `map/objects/android/*.md`, `map/processes/stage-and-promote.md` |
| checksum verification | `map/objects/assets/asset-verification.md` |
| `hps validate` / `hps doctor` | `map/objects/validate/repository-host-validation.md` |
| the embedded server | `map/objects/server/embedded-http-server.md` |
| anything | `map/effects/CONTEXT.md` first |

## Twins

`map/AGENTS.md` and `map/routing.md` are byte-identical twins of this catalog, generated for tools that ignore `CLAUDE.md`. Never hand-edit the twins; regenerate from `CLAUDE.md`.
