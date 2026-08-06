# HPS Architecture

## Purpose

Hermes Provisioning System separates provisioning policy from machine-specific execution. The control layer validates manifests, assets, host identity, approvals, and evidence. Execution modules perform narrowly scoped operating-system tasks.

## Initial component model

```text
hps CLI
├── version
├── validate ── repository and endpoint validation
├── doctor   ── host readiness checks
└── serve    ── health, status API, and repository assets
```

The first milestone is intentionally non-destructive. It establishes interfaces and automated tests before disk, installer, or recovery operations are introduced.

## Runtime and source separation

```text
Git repository                         Deployed runtime
hermes-provisioning-system/            /srv/hermes/
├── cmd/                               ├── bootstrap/
├── internal/                          ├── fedora/
├── docs/                              ├── kickstarts/
└── tests                              ├── recovery/
                                       └── scripts/
```

Source code is developed and reviewed in Git. `/srv/hermes` contains generated or downloaded runtime assets and is not the source of truth.

## Safety gates for future destructive actions

A future install, rebuild, recovery, or repair transaction must not start until all gates pass:

1. Resolve and verify host identity.
2. Resolve and verify the target disk by stable hardware identifiers.
3. Validate the selected manifest and all referenced assets.
4. Verify asset checksums or signatures.
5. Produce a human-readable execution plan.
6. Record explicit human approval for that exact plan.
7. Acquire an expiring operation lock.
8. Execute bounded steps with structured logs.
9. Validate the resulting system.
10. Preserve evidence and release the lock.

## Network model

The embedded server binds to `127.0.0.1:8080` by default. LAN exposure must be explicit. Production deployments may continue to use NGINX in front of HPS for static assets while HPS provides health, status, manifests, and orchestration APIs.

PXE and iPXE support will consume the same versioned manifests and asset catalogue used by HTTP-based provisioning. Boot transport must not define provisioning policy.

## Planned packages

- `internal/manifest` — schema, loading, validation, and version resolution.
- `internal/assets` — downloads, checksums, signatures, and cache management.
- `internal/kickstart` — deterministic Fedora Kickstart generation.
- `internal/bootstrap` — modular post-install execution.
- `internal/approval` — approval records and execution-plan binding.
- `internal/evidence` — structured logs, reports, and checksums.
- `internal/recovery` — bounded repair and recovery workflows.
- `internal/pxe` — boot menu and transport configuration.

## Non-goals for v0.1

- Disk writes or partitioning.
- Automatic operating-system installation.
- Automatic repair or deployment.
- Secret retrieval.
- Remote privileged execution.
- Windows deployment.
