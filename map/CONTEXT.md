# map — how to walk

This shelf is a **system map** (ICM form 6): a record library of nouns, a short shelf of verbs, and a change-impact index. It exists so an editing agent can answer "what is X" and "what else moves if I change X" without reading the whole tree.

## The subject

`~/github/hermes-provisioning-system` — Hermes Provisioning System (HPS), a Go CLI. Early development: repo/host validation, an embedded HTTP server, strict manifest + SHA-256 asset verification, and an approval-gated Android APK staging/channel-promotion pipeline. Disk partitioning, OS install, and recovery execution are deliberately unimplemented (see `docs/architecture.md`, "Non-goals for v0.1"). The subject tree is the source of truth; the map cites it (`path:line`), it never restates as-built behaviour.

## Universes

- **live** — in force; implement and cite against these.
- **leftover** — present but not the main path; touch only if that path is in scope.
- **ghost** — named in `docs/architecture.md`'s "Planned packages" but not built yet.

## Node types (closed set)

See `_meta/schema.md`. Cards are copied from `_templates/object.md` / `process.md`.

## Status of nouns

`map/objects/_index.md` — one line per noun, marked `stub | verified | stale`. A card is `verified` only with a date, commit, and citations.

## Change impact

`map/effects/CONTEXT.md` — if you are changing X, open these cards. First-order hits only.

## Rules

- Never hand-edit `map/AGENTS.md` or `map/routing.md` — they are byte-identical twins of `map/CLAUDE.md`.
- Do not create process cards for movements that do not actually run; `stage-and-promote` is the only one that does today.
- If a card and source disagree, the code wins and the card says so.
