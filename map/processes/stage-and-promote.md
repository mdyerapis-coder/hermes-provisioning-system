---
type: process
status: verified
date: 2026-08-06
commit: 0821982
consumes: [android/android-distribution.md, android/apk-identity-validation.md, assets/asset-verification.md]
produces: [android/android-distribution.md]
---

# stage-and-promote

Takes an approved Android release through immutable staging and then channel promotion — the only multi-step, approval-gated movement that actually runs end-to-end today.

## Input → Movement → Output

Input: a `Release` description plus a human-signed `Approval` record. Movement: `PlanStage` produces a `StagePlan` JSON file; an operator reviews it and writes an `Approval`; `ApplyStage` consumes plan+approval to write immutable staged files; the same plan→approve→apply shape repeats for `PlanChannel` → `PromoteChannel` to move a channel pointer. Output: a staged, checksum-verified APK under the repository, and (separately) an updated channel pointer file.

## Why this shape

If plan and apply were one step, there would be no artifact for a human to review before an irreversible write — the safety-gate model (`docs/architecture.md`, gates 5-6) requires "produce a plan" and "record explicit human approval" as separate, auditable steps.

## Steps

1. `hps android stage plan` → `PlanStage` (`internal/androiddist/stage.go:26`) validates the `Release`/`StageManifest` and calls `androidapk.Validate` (`internal/androidapk/validate.go:55`) to check APK identity before writing a plan.
2. Operator reviews the `StagePlan` JSON and produces an `Approval` record (`internal/androiddist/types.go:98`).
3. `hps android stage apply` → `ApplyStage` (`internal/androiddist/stage.go:95`) refuses to run without a matching plan+approval pair; verifies asset checksums via `internal/assets.VerifyFile` (`internal/assets/verify.go:22`).
4. `hps android channel plan` → `PlanChannel` (`internal/androiddist/channel.go:11`) computes the pointer move for a channel.
5. `hps android channel promote` → `PromoteChannel` (`internal/androiddist/channel.go:68`) applies it, again gated on an `Approval`.

## If you change this

- **Hits:** `internal/androiddist/stage.go`, `internal/androiddist/channel.go`, `internal/androidapk/validate.go`, `internal/assets/verify.go`, `internal/cli/android_distribution.go`
- **Does not hit:** `internal/server` (staged APKs are not served by `hps serve` in v0.1), `internal/manifest` (OS-provisioning manifests are a separate family)

## Surfaces

| Surface | Role |
|---|---|
| operator | writes `Approval`, reviews plans |
| `hps android stage/channel *` | drives every step |

## See

- Objects: `objects/android/android-distribution.md`, `objects/android/apk-identity-validation.md`, `objects/assets/asset-verification.md`
- Source: `internal/androiddist/stage.go`
