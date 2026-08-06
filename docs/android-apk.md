# Android APK validation and distribution

HPS treats Android packages as externally built and signed release artefacts. The Android application repository owns source compilation and signing. HPS verifies and, in later milestones, distributes the resulting immutable files.

## Trust boundary

HPS must never receive or store an Android signing private key. A release record must be produced outside HPS and must pin all of the following:

- APK SHA-256 digest;
- Android application ID;
- version code;
- version name;
- signing-certificate SHA-256 fingerprint.

The expected values must come from an independent CI/release record, not from the APK being validated. Reading a fingerprint from an APK and immediately trusting that same value does not establish identity.

## Current command

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

Debug signing keys are commonly local-development identities and must not be promoted to a stable release channel. Stable distribution requires a separately controlled release-signing identity whose certificate fingerprint is pinned in release metadata.

## Planned runtime layout

APK staging is not implemented yet. The intended repository layout is:

```text
/srv/hermes/android/
├── debug/
│   └── current.json
├── beta/
│   └── current.json
├── stable/
│   └── current.json
└── releases/
    └── com.hermesandroid.app/
        └── VERSION_CODE/
            ├── app.apk
            ├── release.json
            └── SHA256SUMS
```

Historical release directories will be immutable. Channel promotion will update only an approved pointer manifest after validation and explicit human approval.

## Safety boundary

The current implementation does not:

- build or sign an APK;
- copy an APK into `/srv/hermes`;
- create or promote a release channel;
- install an APK on an Android device;
- publish to Google Play;
- contact an Android device over ADB.
