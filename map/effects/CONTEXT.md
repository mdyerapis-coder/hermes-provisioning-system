# Effects — change-impact index

If you are changing X, open these cards first. First-order hits only. If the index and a card disagree, fix the card.

## Changing the CLI

| Change | Open |
|---|---|
| add/change a subcommand | `objects/cli/hps-cli.md`, the `internal/*` package it backs |

## Changing manifests / assets

| Change | Open |
|---|---|
| OS provisioning manifest | `objects/manifest/provisioning-manifest.md`, `objects/assets/asset-verification.md` |
| checksum verification | `objects/assets/asset-verification.md`, `objects/manifest/provisioning-manifest.md`, `objects/android/android-distribution.md` |

## Changing Android distribution

| Change | Open |
|---|---|
| APK identity checks | `objects/android/apk-identity-validation.md`, `objects/android/android-distribution.md`, `processes/stage-and-promote.md` |
| staging / channel promotion | `objects/android/android-distribution.md`, `processes/stage-and-promote.md`, `objects/android/apk-identity-validation.md`, `objects/assets/asset-verification.md` |

## Changing validation / server

| Change | Open |
|---|---|
| `hps validate` / `hps doctor` | `objects/validate/repository-host-validation.md`, `objects/cli/hps-cli.md` |
| embedded server | `objects/server/embedded-http-server.md`, `objects/cli/hps-cli.md` |

## Does not hit

- Changing `internal/androiddist` does not hit `internal/manifest` (separate manifest families; OS-provisioning manifests are untouched by APK staging).
- Changing `internal/server` does not hit `internal/androiddist` — staged APKs are not served by `hps serve` in v0.1 (non-goal in `docs/architecture.md`).
- Changing `map/` does not hit `internal/*` runtime code.
