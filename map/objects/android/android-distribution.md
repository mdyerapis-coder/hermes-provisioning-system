---
type: object
cluster: android
universe: live
status: verified
date: 2026-08-06
commit: 0821982
entity: internal/androiddist
---

# Android Distribution — staging + channel promotion

Approval-gated pipeline that takes a `Release` to an immutable staged APK (`StagePlan` → `StageResult`), then promotes a staged version to a `Channel` pointer (`ChannelPlan` → `Channel`).

## Why this shape

Every mutating step splits into `Plan*` (pure, writes a JSON plan) and `Apply*`/`Promote*` (requires a separately-produced `Approval` record) — mirrors the safety-gate model in `docs/architecture.md` (plan → human approval → execute). Neither `ApplyStage` nor `PromoteChannel` runs without a matching approval file on disk.

## Shape

- `StageManifest`/`StagePlan`/`StageResult` — `internal/androiddist/types.go:32-146`
- `ChannelPlan`/`Channel` — `internal/androiddist/types.go:148-184`
- `Approval`, `Release`/`ReleaseMetadata`/`ReleaseSpec` — `internal/androiddist/types.go:98-136`
- `PlanStage`, `ApplyStage` — `internal/androiddist/stage.go:26`, `stage.go:95`
- `PlanChannel`, `PromoteChannel`, `ShowChannel` — `internal/androiddist/channel.go:11,68,176`
- `LoadStageManifest`, `WriteJSON` — `internal/androiddist/stage.go:14,82`

Citations: `internal/androiddist/stage.go:26`, `internal/androiddist/channel.go:68`

## Connected to

- **owns:** its own JSON plan/result files on disk (under the provisioning repository root)
- **owned-by:** `internal/cli/android_distribution.go`
- **joins:** `internal/androidapk` (validates the APK during staging), `internal/assets` (checksum verify)
- **looks-like-but-is-not:** not `internal/manifest.Manifest` (OS-provisioning manifest) — a distinct, APK-only manifest family

## If you change this

- **Hits:** `internal/cli/android_distribution.go`, `internal/androidapk/validate.go` (staging calls it), `internal/androiddist/distribution_test.go`, `internal/androiddist/permissions_test.go`
- **Does not hit:** `internal/manifest` (OS manifest validation is untouched), `internal/server` (server does not serve staged APKs in v0.1)

## Surfaces

| Surface | Role |
|---|---|
| `hps android stage plan/apply` | writes |
| `hps android channel plan/promote/show` | writes/reads |

## See

- Source: `internal/androiddist/stage.go`
