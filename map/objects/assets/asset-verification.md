---
type: object
cluster: assets
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/assets/verify.go
---

# Asset Verification (SHA-256)

`VerifyFile(path, expected string) (Verification, error)` — the single checksum primitive used wherever a downloaded/staged file's integrity must be proven.

## Why this shape

Kept as one dependency-free function so every package that needs a checksum (`internal/manifest` validation, `internal/androiddist` staging) can call it without pulling in APK- or manifest-specific logic — one home for "does this file's SHA-256 match."

## Shape

- `Verification` — `internal/assets/verify.go:13`
- `VerifyFile(path, expected string) (Verification, error)` — `internal/assets/verify.go:22`

Citations: `internal/assets/verify.go:22`

## Connected to

- **owns:** nothing
- **owned-by:** `internal/cli/asset.go` (`hps asset verify`)
- **joins:** `internal/manifest` (`validateSHA256` mirrors the same digest format), `internal/androiddist` (staging verifies APK checksums)
- **looks-like-but-is-not:** not APK identity validation (`internal/androidapk` checks package id/signer, not just bytes)

## If you change this

- **Hits:** `internal/cli/asset.go`, `internal/assets/verify_test.go`, `internal/manifest/manifest.go` (same digest format convention)
- **Does not hit:** `internal/androidapk` (identity checks are independent of raw-file checksums)

## Surfaces

| Surface | Role |
|---|---|
| `hps asset verify` | invokes directly |
| `internal/androiddist` staging | invokes as a precondition |

## See

- Source: `internal/assets/verify.go`
