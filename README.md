# Hermes Provisioning System (HPS)

Hermes Provisioning System is a manifest-driven provisioning, recovery, HTTP boot, and deployment platform for the Hermes ecosystem.

> **Project status:** early development. The CLI provides repository validation, host diagnostics, strict provisioning manifests, SHA-256 asset verification, Android APK identity verification, approval-gated immutable APK staging, approval-gated channel pointers, and a safe embedded HTTP file server. Disk partitioning, operating-system installation, rebuild, recovery execution, APK installation, and Android signing remain deliberately unimplemented.

## Current commands

```text
hps version
hps validate [--repo /srv/hermes] [--server http://host/hermes/]
hps doctor   [--repo /srv/hermes] [--server http://host/hermes/]
hps serve    [--root /srv/hermes] [--listen 127.0.0.1:8080]
hps manifest validate --file MANIFEST.json
hps asset verify --file FILE --sha256 DIGEST
hps android validate --apk FILE --sha256 DIGEST --package APPLICATION_ID \
  --version-code CODE --version-name NAME --signer-sha256 CERT_DIGEST
hps android stage plan  --manifest RELEASE.json --repo /srv/hermes --out PLAN.json
hps android stage apply --plan PLAN.json --approval APPROVAL.json --repo /srv/hermes
hps android channel plan    --repo /srv/hermes --channel CHANNEL --version VERSION --out PLAN.json
hps android channel promote --repo /srv/hermes --plan PLAN.json --approval APPROVAL.json
hps android channel show    --repo /srv/hermes --channel CHANNEL
```

The embedded server provides:

- `GET /healthz` — liveness check
- `GET /api/v1/status` — build and repository status
- `/` — files from the configured provisioning repository

The default listener is loopback-only. Expose it to the LAN explicitly:

```bash
hps serve --root /srv/hermes --listen 0.0.0.0:8080
```

## Provisioning manifests

HPS v1 manifests are strict JSON documents. Unknown fields are rejected, asset destinations must remain relative to the runtime repository, and every declared asset carries a SHA-256 digest.

```bash
./bin/hps manifest validate \
  --file examples/manifests/fedora44-kde.json
```

The bundled Fedora example contains placeholder release metadata and must not be treated as an approved installer source. See [`docs/manifests.md`](docs/manifests.md).

Verify an already-downloaded file without changing it:

```bash
./bin/hps asset verify \
  --file /path/to/asset.iso \
  --sha256 VERIFIED_DIGEST
```

## Android APK validation and distribution

HPS validates an existing APK against identity values supplied by an independent release record. Validation checks the APK SHA-256 digest, application ID, version code, version name, cryptographic signature, and signing-certificate SHA-256 fingerprint.

```bash
./bin/hps android validate \
  --apk /path/to/app-release.apk \
  --sha256 VERIFIED_APK_DIGEST \
  --package com.hermesandroid.app \
  --version-code 1 \
  --version-name 0.1.0-phase1 \
  --signer-sha256 VERIFIED_CERTIFICATE_DIGEST
```

Android SDK Build Tools must provide `aapt2` and `apksigner`. HPS discovers them from `PATH`, `ANDROID_HOME`, or `ANDROID_SDK_ROOT`, or accepts explicit paths. Debug builds use `com.hermesandroid.app.debug`; release builds use `com.hermesandroid.app`.

Staging and channel promotion use a two-step plan/apply workflow. The apply operation requires a separate `HPSApproval` document bound to the SHA-256 of the exact serialized plan. Existing release directories are never replaced, and channel promotion changes only `current.json`.

See:

- [`docs/android-apk.md`](docs/android-apk.md) for APK identity verification;
- [`docs/android-distribution.md`](docs/android-distribution.md) for staging, approvals, immutable releases, and channel promotion;
- [`examples/android/stage-manifest.example.json`](examples/android/stage-manifest.example.json) for a non-production input template.

## Expected runtime repository

The initial validation profile expects:

```text
/srv/hermes/
├── android/
├── bootstrap/
├── fedora/
├── kickstarts/
├── recovery/
└── scripts/
```

The Android directory is created only by an explicitly approved stage operation. No APK has to be present for the normal repository doctor checks.

Additional assets such as PXE binaries, versioned runtime manifests, Windows media, Clonezilla, and SystemRescue will be added in later milestones.

## Build

HPS currently targets Linux and requires Go 1.26 or newer.

```bash
make check
make build
./bin/hps version
```

Install locally:

```bash
sudo make install
```

## Test against Hermes Station

```bash
./bin/hps doctor \
  --repo /srv/hermes \
  --server http://192.168.1.88/hermes/
```

## Environment

| Variable | Purpose | Default |
|---|---|---|
| `HPS_REPOSITORY` | Provisioning repository root | `/srv/hermes` |
| `HPS_SERVER` | Endpoint checked by `validate` and `doctor` | unset |
| `HPS_LISTEN` | Embedded server address | `127.0.0.1:8080` |
| `HPS_AAPT2` | Optional path to the Android `aapt2` executable | auto-discover |
| `HPS_APKSIGNER` | Optional path to the Android `apksigner` executable | auto-discover |

## Roadmap

1. **v0.1:** CLI, validation, doctor, embedded server, CI — complete.
2. **v0.2:** Versioned manifests and checksum verification — complete.
3. **v0.3:** Distribution foundation — APK validation merged; approval-gated staging and channel promotion under review.
4. **v0.4:** Kickstart generation, modular post-install bootstrap, and controlled recovery planning.
5. **v1.0:** Production HTTP/PXE provisioning, recovery media, audit evidence, and release packaging.

## Safety principles

- Destructive or publishing actions must require explicit confirmation and a machine-readable approval record.
- An Android stage or promotion approval must be bound to the SHA-256 of the exact reviewed plan file.
- HPS must identify the target host and target disk before any disk write operation.
- Validation must complete before installation, recovery, deployment, publication, or repair.
- Installer and application artefacts must be checked against independently verified checksums and signing identities.
- HPS must never possess an Android application signing private key.
- Debug signing identities must not be treated as stable-release trust anchors.
- Secrets must never be committed to this public repository.
- Merges, releases, deployments, infrastructure changes, staging, promotion, and repairs remain human-approved.

## System map

See `map/CLAUDE.md` for a walkable graph of this repository (nouns, verbs, change-impact index) for editing agents.

## Licence

Apache-2.0. See [`LICENSE`](LICENSE).
