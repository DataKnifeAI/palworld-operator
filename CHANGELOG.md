# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Opt-in `spec.update.autoUpdateImage`: GHCR tag discovery, pin `repo:vX.Y.Z.W`, status fields (`desiredImage`, `runningVersion`, `latestAvailableVersion`, `updateAvailable`), `onlyWhenEmpty`, optional cron `checkSchedule` / `applySchedule` + `timeZone` (default UTC), optional REST `/v1/api/announce` pre-roll notice (`notifyPlayers`).
- `DedicatedServerName` persistence: learn REST `worldguid`, seed `GameUserSettings.ini` (spec or status pin) so Recreate / auto-update keeps the world.
- [docs/FAQ.md](docs/FAQ.md) (+ site FAQ section): incapable version, passwords, world pin, image updates, local vs cluster, sizing.

### Changed

- Sample CR / Compose default image pin: `ghcr.io/pocketpairjp/palserver:v1.0.1.100619` (prefer explicit tags over stale `:latest`).

### Planned / known gaps

- Finish cluster smoke (#12): client join via Gateway, PVC retain across restart, graceful stop.
- Fake-client Reconcile unit tests (#10), optional envtest (#11), negative/ops status messages (#13).

## [0.1.0] — 2026-07-13

First public MVP cut (`Makefile` `VERSION=0.1.0`). Early release — usable for hosting, not production-hardened.

### Added

- `PalworldServer` CRD + reconciler: Deployment, PVC, ConfigMap INI seed, Services, status.
- Envoy Gateway path (UDP game/query; optional REST TCPRoute).
- Optional `spec.generateSecrets` (fill-if-missing Opaque Secret).
- Resource auto-selection tiers + `spec.resources` override.
- Docker Compose local / minimal PC path (`compose/`, `make compose-up`, [docs/LOCAL.md](docs/LOCAL.md)).
- GitHub Actions lint/test/build; GitLab Harbor publish; Pages site.
- Docs: CONNECT, ARCHITECTURE, PALWORLD_SERVER (incl. DedicatedServerName caveat), GITLAB_MIRROR.

### Known limitations

See Unreleased planned gaps above. Prefer pinning `spec.serverImage` to a Pocketpair version tag in any lasting world.
