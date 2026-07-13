# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned / known gaps

- Operator does not yet pin/manage `DedicatedServerName` (`GameUserSettings.ini`) — world drift risk on restart without a pin ([docs](docs/PALWORLD_SERVER.md#world-selection-across-restarts)).
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
