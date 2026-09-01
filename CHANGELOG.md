# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Opt-in `spec.mods`: dedicated `{name}-mods` PVC at `/pal/Package/Mods` plus default Paks **subpath** overlays (`~WorkshopMods`, `LogicMods`) so Linux `.pak` files do not hide `Pal-LinuxServer.pak`. Official Workshop loader remains Windows-only; UE4SS/Win64 is not this image. Optional `activeModList`; `-workshopdir` off unless `useWorkshopDirArg`.

### Planned / known gaps

- Finish cluster smoke (#12): client join via Gateway, graceful stop / SIGTERM save integrity (PVC retain observed on live auto-update Recreate).
- Fake-client Reconcile unit tests (#10), optional envtest (#11), negative/ops status messages (#13).

## [0.2.0-beta.1] — 2026-08-31

Public beta cut (`Makefile` `VERSION=0.2.0-beta.1`). First tagged release of the post-MVP operator: auto-update, `optionSettings`, and world-pin persistence. Usable for hosting, not production-hardened.

### Added

- `spec.optionSettings` (`map[string]string`): extra [PalWorldSettings.ini OptionSettings](https://docs.palworldgame.com/settings-and-operation/configuration/) merged into the ConfigMap INI and re-seeded on every pod start so balance/feature keys survive PVC overwrite. Management CR fields override the same keys; community images get best-effort env mapping (official image preferred for full INI).
- Opt-in `spec.update.autoUpdateImage`: GHCR tag discovery, pin `repo:vX.Y.Z.W`, status fields (`desiredImage`, `runningVersion`, `latestAvailableVersion`, `updateAvailable`), `onlyWhenEmpty`, optional cron `checkSchedule` / `applySchedule` + `timeZone` (default UTC), optional REST `/v1/api/announce` pre-roll notice (`notifyPlayers`).
- Multi-stage pre-update announce schedule (`spec.update.notifySchedule`, default 60m→10s) with `status.plannedApplyTime` / `announcedNotifyStages`; non-blocking reconcile requeues at stage boundaries; short final countdown immediately before apply. Legacy `notifyLeadTime` remains a single-stage fallback.
- `DedicatedServerName` persistence: learn REST `worldguid`, seed `GameUserSettings.ini` (spec or status pin) so Recreate / auto-update keeps the world.
- [docs/FAQ.md](docs/FAQ.md) (+ site FAQ section): incapable version, passwords, world pin, settings reset, image updates, local vs cluster, sizing.

### Changed

- Docs / site: `spec.optionSettings` as a passthrough map, official [OptionSettings](https://docs.palworldgame.com/settings-and-operation/configuration/) link, reserved CR keys, apply = patch + roll (world/PVC intact), example `bExistPlayerAfterLogout: "True"`.
- Sample CR / Compose default image pin: `ghcr.io/pocketpairjp/palserver:v1.0.1.100619` (prefer explicit tags over stale `:latest`).
- `notifyLeadTime` deprecated in favor of `notifySchedule` (still honored as a single stage when schedule is empty).

### Known limitations

See Unreleased planned gaps. Prefer pinning `spec.serverImage` to a Pocketpair version tag in any lasting world.

## [0.1.0] — 2026-07-13

First public MVP cut (`Makefile` `VERSION=0.1.0`). Early release — usable for hosting, not production-hardened. GitHub tag `v0.1.0-beta.1` later captured this cut plus in-progress post-MVP work; `v0.2.0-beta.1` is the versioned beta.

### Added

- `PalworldServer` CRD + reconciler: Deployment, PVC, ConfigMap INI seed, Services, status.
- Envoy Gateway path (UDP game/query; optional REST TCPRoute).
- Optional `spec.generateSecrets` (fill-if-missing Opaque Secret).
- Resource auto-selection tiers + `spec.resources` override.
- Docker Compose local / minimal PC path (`compose/`, `make compose-up`, [docs/LOCAL.md](docs/LOCAL.md)).
- GitHub Actions lint/test/build; GitLab Harbor publish; Pages site.
- Docs: CONNECT, ARCHITECTURE, PALWORLD_SERVER (incl. DedicatedServerName caveat), GITLAB_MIRROR.

### Known limitations

See Unreleased planned gaps. Prefer pinning `spec.serverImage` to a Pocketpair version tag in any lasting world.

[Unreleased]: https://github.com/DataKnifeAI/palworld-operator/compare/v0.2.0-beta.1...HEAD
[0.2.0-beta.1]: https://github.com/DataKnifeAI/palworld-operator/compare/v0.1.0-beta.1...v0.2.0-beta.1
[0.1.0]: https://github.com/DataKnifeAI/palworld-operator/releases/tag/v0.1.0-beta.1
