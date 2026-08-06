# Security Policy

Hermes Provisioning System can eventually perform high-impact operations such as disk preparation, operating-system installation, recovery, and network boot. Security reports are therefore treated as operationally significant.

## Supported versions

HPS is pre-release software. Security fixes are applied to the latest development branch until the first stable release policy is published.

## Reporting a vulnerability

Do not open a public issue containing credentials, private infrastructure details, exploit code against a live system, or sensitive logs.

Use GitHub's private vulnerability reporting feature for this repository when available. Include:

- the affected commit or release;
- the command or component involved;
- the expected and observed behaviour;
- a minimal reproduction that does not expose real secrets;
- the potential impact;
- any suggested mitigation.

## Security boundaries

- Secrets must not be committed to this repository.
- Destructive operations must require explicit human approval.
- Target host and target disk identity must be verified before writes.
- Network services default to loopback unless exposure is explicitly requested.
- Downloaded installer assets will require checksums or signatures before use.
- Recovery, deployment, infrastructure changes, and repairs must produce auditable evidence.
