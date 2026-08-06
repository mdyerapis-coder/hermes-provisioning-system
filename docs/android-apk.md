# Android APK validation and distribution

HPS treats Android packages as externally built and signed release artefacts. The Android application repository owns source compilation and signing. HPS verifies the resulting immutable files and can stage or promote them only through separately approved plans.

## Trust boundary

HPS must never receive or store an Android signing private key. A release record must be produced outside HPS and must pin all of the following:

- APK SHA-256 digest;
- Android application ID;
- version code;
- version name;
- signing-certificate SHA-256 fingerprint.

The expected values must come from an independent CI/release record, not from the APK being validated. Reading a fingerprint from an APK and immediately trusting that same value does not establish identity.

## APK validation

```bash
hps android validate \
  --apk FILE \
  --sha256 APK_DIGEST \
  --package APPLICATION_ID \
  --version-code CODE \
  --version-name NAME \
  --signer-sha256 CERTIFICATE_DIGEST
```

The validator performs these read-only checks:

1. Rejects symbolic links, non-regular files, and names without an `.apk` extension.
2. Hashes the APK and compares it with the expected SHA-256 digest.
3. Uses `aapt2 dump badging` to extract the package, version code, and version name.
4. Uses `apksigner verify --print-certs` to verify the APK signature and extract signing-certificate metadata.
5. Requires exactly one distinct signer identity.
6. Re-hashes the APK after tool execution to detect ordinary changes during validation.

HPS discovers Android SDK Build Tools from `PATH`, `ANDROID_HOME`, or `ANDROID_SDK_ROOT`. Exact executable paths can be provided with `--aapt2` and `--apksigner`, or with `HPS_AAPT2` and `HPS_APKSIGNER`.

## Hermes Android identities

The supplied Hermes Android project currently defines:

| Build type | Application ID | Version code | Version name |
|---|---|---:|---|
| Release | `com.hermesandroid.app` | `1` | `0.1.0-phase1` |
| Debug | `com.hermesandroid.app.debug` | `1` | `0.1.0-phase1` |

Debug signing keys are local-development identities and must not be promoted to a stable release channel. Stable distribution requires a separately controlled release-signing identity whose certificate fingerprint is pinned in release metadata.

## Approval-gated distribution

The distribution workflow is documented in [`android-distribution.md`](android-distribution.md). It provides:

- strict `AndroidStageManifest` inputs;
- read-only stage and channel plans;
- `HPSApproval` records bound to exact plan SHA-256 values;
- immutable `/srv/hermes/android/<channel>/<version>/` directories;
- generated `release.json` and `SHA256SUMS`;
- atomic `current.json` channel promotion;
- stable-channel rejection of `.debug` application IDs.

Code availability does not authorize a live staging or promotion action. Those remain separate human approval gates.

## Safety boundary

HPS does not:

- build or sign an APK;
- possess an Android signing private key;
- install an APK on an Android device;
- publish to Google Play;
- contact an Android device over ADB;
- silently stage or promote a release without exact-plan approval.
