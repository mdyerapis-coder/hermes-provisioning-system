# Approval-gated Android APK distribution

HPS stages Android APKs as immutable, externally built and signed release artefacts. It does not compile Android source, hold signing private keys, install applications, or publish to Google Play.

## Runtime layout

```text
/srv/hermes/android/
├── debug/
│   ├── 0.1.0-phase1/
│   │   ├── hermes-android-debug.apk
│   │   ├── release.json
│   │   └── SHA256SUMS
│   └── current.json
├── beta/
│   └── ...
└── stable/
    └── ...
```

A version directory is created once and cannot be replaced by HPS. Channel promotion atomically changes only `current.json`. The `stable` channel rejects application IDs ending in `.debug`.

## Trust inputs

The stage manifest must pin values established outside HPS:

- absolute local APK path;
- APK SHA-256;
- Android application ID;
- version code and version name;
- signing-certificate SHA-256 fingerprint;
- source repository and source commit provenance.

Do not copy identity values from an untrusted APK and immediately approve them. The digest, signer fingerprint, and provenance must be independently reviewed.

## Stage manifest

Start from `examples/android/stage-manifest.example.json` and replace every placeholder. `metadata.version` must exactly match `spec.versionName`.

```json
{
  "apiVersion": "hps.hermes/v1",
  "kind": "AndroidStageManifest",
  "metadata": {
    "name": "hermes-android-debug",
    "version": "0.1.0-phase1"
  },
  "spec": {
    "channel": "debug",
    "apk": "/absolute/path/to/app-debug.apk",
    "fileName": "hermes-android-debug.apk",
    "package": "com.hermesandroid.app.debug",
    "versionCode": "1",
    "versionName": "0.1.0-phase1",
    "sha256": "REPLACE_WITH_64_HEX_DIGEST",
    "signerSha256": "REPLACE_WITH_64_HEX_CERTIFICATE_DIGEST",
    "provenance": {
      "sourceRepository": "SOURCE_REPOSITORY",
      "commit": "SOURCE_COMMIT",
      "buildId": "CI_BUILD_ID",
      "builder": "BUILDER_IDENTITY"
    }
  }
}
```

Unknown fields, symbolic-link paths, traversal paths, invalid digests, unsafe file names, and incomplete provenance are rejected.

## 1. Create a stage plan

Planning validates the APK with `aapt2` and `apksigner`, resolves the exact repository and source paths, and writes a new plan file. It does not change `/srv/hermes`.

```bash
hps android stage plan \
  --manifest stage-manifest.json \
  --repo /srv/hermes \
  --out stage-plan.json
```

The command prints the SHA-256 of the exact serialized plan. Review the complete plan, including source path, destination, package, version, digest, signer, size, and provenance.

## 2. Create an exact-plan approval

Approval is a separate JSON file. The `planSha256` value must match the exact plan file bytes.

```json
{
  "apiVersion": "hps.hermes/v1",
  "kind": "HPSApproval",
  "action": "android-stage",
  "planSha256": "EXACT_STAGE_PLAN_SHA256",
  "approved": true,
  "approvedBy": "HUMAN_APPROVER",
  "approvedAt": "2026-08-06T06:00:00Z",
  "expiresAt": "2026-08-07T06:00:00Z"
}
```

An approval is rejected when it is missing, false, expired, unexpectedly future-dated, for the wrong action, or bound to a different plan digest.

## 3. Apply immutable staging

```bash
hps android stage apply \
  --plan stage-plan.json \
  --approval stage-approval.json \
  --repo /srv/hermes
```

Before writing, HPS revalidates the repository, source path, APK digest, package, versions, signature, signer fingerprint, size, plan, and approval. It then writes the APK, `SHA256SUMS`, and finally `release.json` as the release commit marker. An existing version directory is never replaced.

Staging does not update a channel pointer.

## 4. Plan channel promotion

```bash
hps android channel plan \
  --repo /srv/hermes \
  --channel debug \
  --version 0.1.0-phase1 \
  --out channel-plan.json
```

The command verifies the staged APK and `release.json`, then produces an exact promotion plan without modifying `current.json`.

## 5. Approve and promote

Use the same approval structure with:

```json
"action": "android-channel-promote"
```

and the exact channel-plan SHA-256.

```bash
hps android channel promote \
  --repo /srv/hermes \
  --plan channel-plan.json \
  --approval channel-approval.json
```

Promotion rechecks the release manifest and APK digests, then atomically replaces only:

```text
/srv/hermes/android/CHANNEL/current.json
```

## 6. Inspect the current pointer

```bash
hps android channel show \
  --repo /srv/hermes \
  --channel debug
```

The command rejects a pointer whose referenced release manifest has changed.

## Operational boundaries

- Code merge does not authorize staging or channel promotion on a live host.
- A debug signing certificate is not a production trust anchor.
- Beta and stable release signing must be controlled outside HPS.
- Plan and approval files may contain internal paths and approver identity; keep them outside the public web root.
- NGINX MIME, attachment, dotfile-denial, and HTTP verification work is tracked separately so live server changes receive independent review.
