# HPS Provisioning Manifests

HPS manifests are versioned, declarative JSON documents. They identify an operating-system target and the immutable assets required to prepare it. A manifest is data only: validation does not download, write, install, execute, partition, or reboot anything.

## Schema identity

Every current manifest must declare:

```json
{
  "apiVersion": "hps.hermes/v1",
  "kind": "ProvisioningManifest"
}
```

Unknown JSON fields are rejected. The strict decoder is intentional: misspelled or unsupported fields must fail rather than being silently ignored.

## Metadata

- `metadata.name` uses lowercase DNS-style characters and hyphens.
- `metadata.version` is required and identifies the manifest revision.
- `metadata.description` is optional human-readable context.

## Operating-system target

`spec.os` requires:

- `distribution`
- `release`
- `architecture`

`variant` is optional. The target is descriptive in v0.2 and does not trigger installation.

## Assets

Each entry in `spec.assets` requires:

- a unique lowercase `name`;
- a non-empty `type` such as `iso`, `kernel`, `initrd`, or `kickstart`;
- an HTTP or HTTPS `source`;
- a unique repository-relative `destination` that cannot escape `/srv/hermes`;
- a 64-character SHA-256 digest;
- a `required` boolean.

Sources and destinations are validated but not accessed by `hps manifest validate`.

## Validate a manifest

```bash
hps manifest validate --file examples/manifests/fedora44-kde.json
```

## Verify a staged asset

```bash
hps asset verify \
  --file /srv/hermes/fedora/44/Fedora-KDE-Live-44-x86_64.iso \
  --sha256 VERIFIED_DIGEST
```

A mismatch exits non-zero and prints both the expected and calculated digest.

## Security boundaries

- Manifests must never contain credentials or private tokens.
- Example URLs and digests are not trusted release metadata.
- Asset acquisition is not implemented in v0.2.
- Official checksums or signatures must be verified before any installer asset is staged for use.
- Destructive workflows remain outside the manifest validator and require future explicit approval gates.
