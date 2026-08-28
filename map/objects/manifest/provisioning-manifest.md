---
type: object
cluster: manifest
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/manifest/manifest.go
---

# Provisioning Manifest

The declarative unit describing what to provision: `Manifest{Metadata, Spec}` where `Spec` holds `OperatingSystem` and a list of `Asset`.

## Why this shape

Manifests are validated strictly before anything executes — SHA-256 required on every asset, sources/destinations checked for path-traversal shape — because the safety-gate model (`docs/architecture.md`, gates 3-4) requires a validated manifest before any future destructive step. `Validate()` is a method on the struct, so a manifest can't be used without passing through it.

## Shape

- `Manifest{Metadata, Spec}`, `Spec{OperatingSystem, Assets []Asset}` — `internal/manifest/manifest.go:21`
- `(m Manifest) Validate() error` — `internal/manifest/manifest.go:60`
- `validateSource`, `validateDestination`, `validateSHA256` — `internal/manifest/manifest.go:120-160`
- Loading: `internal/manifest/load.go`

Citations: `internal/manifest/manifest.go:21`, `internal/manifest/manifest.go:60`

## Connected to

- **owns:** nothing external
- **owned-by:** `internal/cli/manifest.go` (`manifest validate`); future `internal/bootstrap` (planned, not built — see `docs/architecture.md`)
- **joins:** `internal/assets` (`validateSHA256` mirrors the digest shape `assets.VerifyFile` checks)
- **looks-like-but-is-not:** not the Android `Release`/`StageManifest` types in `internal/androiddist` — a separate, APK-specific manifest family

## If you change this

- **Hits:** `internal/cli/manifest.go`, `internal/manifest/manifest_test.go`, `docs/manifests.md`
- **Does not hit:** `internal/androiddist` types (different manifest family entirely)

## Surfaces

| Surface | Role |
|---|---|
| `hps manifest validate` | reads |
| operator | authors manifest JSON |

## See

- Source: `internal/manifest/manifest.go`
