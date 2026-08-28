---
type: object
cluster: android
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/androidapk/validate.go
---

# APK Identity Validation

`Validate(ctx, Options) (Result, error)` — verifies an APK's package id, version code/name, and signer SHA-256 against declared expectations, by shelling out to Android build tools and parsing their output.

## Why this shape

Shelling out (`runTool`) rather than parsing the APK binary format directly keeps `parseBadging`/`parseSignerSHA256` as the only APK-format-aware code, small and tested, so the rest of the distribution pipeline only ever deals with a validated `Result`, never a raw APK.

## Shape

- `Options` (apk path, expected package/version/signer) — `internal/androidapk/validate.go:29`
- `Result` (verified identity) — `internal/androidapk/validate.go:41`
- `Validate(ctx, Options) (Result, error)` — `internal/androidapk/validate.go:55`
- `resolveTool`, `runTool`, `parseBadging`, `parseSignerSHA256` — `internal/androidapk/validate.go:182-282`

Citations: `internal/androidapk/validate.go:55`, `internal/androidapk/validate.go:236`

## Connected to

- **owns:** nothing
- **owned-by:** `internal/cli/android.go` (`android validate`); `internal/androiddist` (staging validates identity before accepting a release)
- **joins:** `internal/assets` (raw-file checksum is a separate, earlier step)
- **looks-like-but-is-not:** not APK *signing* — only verifies an existing signature's digest, never signs

## If you change this

- **Hits:** `internal/cli/android.go`, `internal/androiddist/stage.go` (consumes validation before staging), `internal/androidapk/validate_test.go`
- **Does not hit:** `internal/androiddist/channel.go` (channel promotion trusts an already-staged, already-validated APK; does not re-run APK validation)

## Surfaces

| Surface | Role |
|---|---|
| `hps android validate` | invokes directly |
| staging pipeline | invokes as a precondition |

## See

- Source: `internal/androidapk/validate.go`
