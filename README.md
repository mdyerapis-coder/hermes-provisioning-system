# Hermes Provisioning System (HPS)

Hermes Provisioning System is a manifest-driven provisioning, recovery, HTTP boot, and deployment platform for the Hermes ecosystem.

> **Project status:** early development. The current CLI provides read-only validation and diagnostics plus a safe embedded HTTP file server. Disk partitioning, operating-system installation, rebuild, and recovery execution are deliberately not implemented yet.

## Current commands

```text
hps version
hps validate [--repo /srv/hermes] [--server http://host/hermes/]
hps doctor   [--repo /srv/hermes] [--server http://host/hermes/]
hps serve    [--root /srv/hermes] [--listen 127.0.0.1:8080]
```

The embedded server provides:

- `GET /healthz` — liveness check
- `GET /api/v1/status` — build and repository status
- `/` — files from the configured provisioning repository

The default listener is loopback-only. Expose it to the LAN explicitly:

```bash
hps serve --root /srv/hermes --listen 0.0.0.0:8080
```

## Expected runtime repository

The first validation profile expects:

```text
/srv/hermes/
├── bootstrap/
├── fedora/
├── kickstarts/
├── recovery/
└── scripts/
```

Additional assets such as PXE binaries, manifests, Windows media, Clonezilla, and SystemRescue will be added in later milestones.

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
| `HPS_LISTEN` | Embedded HTTP server address | `127.0.0.1:8080` |

## Roadmap

1. **v0.1:** CLI, validation, doctor, embedded server, CI.
2. **v0.2:** Versioned manifests, checksums, asset acquisition, Fedora installer staging.
3. **v0.3:** Kickstart generation and modular post-install bootstrap.
4. **v0.4:** Controlled rebuild and recovery planning with explicit human approval gates.
5. **v1.0:** Production HTTP/PXE provisioning, recovery media, audit evidence, and release packaging.

## Safety principles

- Destructive actions must require explicit confirmation and a machine-readable approval record.
- HPS must identify the target host and target disk before any write operation.
- Validation must complete before installation, recovery, deployment, or repair.
- Secrets must never be committed to this public repository.
- Merges, releases, deployments, infrastructure changes, and repairs remain human-approved.

## Licence

Apache-2.0. See [`LICENSE`](LICENSE).
