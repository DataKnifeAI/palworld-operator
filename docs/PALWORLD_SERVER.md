# Palworld dedicated server — operator research notes

Sources:
- [Deploy dedicated server](https://docs.palworldgame.com/getting-started/deploy-dedicated-server) (official)
- [Configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/) (official)
- [Official Docker image (Pocketpair)](https://github.com/pocketpairjp/palworld-dedicated-server-docker) — `ghcr.io/pocketpairjp/palserver`
- [thijsvanloef/palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker) (community alternative)

## Official distribution

Pocketpair publishes an **official dedicated server container** on GHCR:

| Item | Value |
|------|-------|
| Image | `ghcr.io/pocketpairjp/palserver` |
| Source | [pocketpairjp/palworld-dedicated-server-docker](https://github.com/pocketpairjp/palworld-dedicated-server-docker) |
| Docs | [Palworld Server Guide](https://tech.palworldgame.com/) / [requirements](https://tech.palworldgame.com/getting-started/requirements) |
| Tags | Versioned (e.g. `v1.0.0.100427`) and `latest` |

SteamCMD App ID **2394010** remains the underlying dedicated server; the official image packages that build. Non-container installs (Steam / SteamCMD on Linux or Windows) are still documented by Pocketpair.

**No DataKnifeAI custom server-image repository is required** while this official image is maintained. Mirror to Harbor only if cluster policy prefers a private registry (optional ops step, not the operator default).

### Official image layout

| Path | Purpose |
|------|---------|
| `/pal/Package/PalServer.sh` | Server entrypoint (via `/pal/helper.sh` in compose samples) |
| `/pal/Package/DefaultPalWorldSettings.ini` | Defaults template (do **not** edit for live config) |
| `/pal/Package/Pal/Saved` | Persist this directory (saves + `Config/LinuxServer/`) |

Compose sample mounts host `./Saved` → `/pal/Package/Pal/Saved` and passes CLI args (`-port=8211`, multithreading flags). Gameplay settings go in `PalWorldSettings.ini` under the Saved mount — same INI model as bare SteamCMD.

## Ports

| Port | Proto | Role | Operator notes |
|------|-------|------|----------------|
| 8211 | UDP | Game traffic | Primary client connect; expose via UDPRoute |
| 27015 | UDP | Steam query | Community browser / Steam; expose via UDPRoute when listing |
| 25575 | TCP | RCON | Enable via INI / community image env for graceful stop |
| 8212 | TCP | REST API | Useful for health/player logging; **do not** public-forward |

Official compose examples expose **8211/UDP** only; query/RCON/REST are still part of the dedicated server when enabled in settings.

## Persistence

| Path | Purpose |
|------|---------|
| `Pal/Saved/SaveGames/` | World saves (must be on PVC) |
| `Pal/Saved/Config/LinuxServer/` | `PalWorldSettings.ini`, related INI |
| Official image mount | `/pal/Package/Pal/Saved` |
| Community image mount | `/palworld` (install + saves + backups) |

Recommended PVC size: **≥ 100Gi** default in the CRD draft (saves + server files + backups grow; Windrose uses 35Gi as a floor for a lighter game).

## Container image options

| Image | Role |
|-------|------|
| **`ghcr.io/pocketpairjp/palserver`** | **Operator default** — official Pocketpair image (Windrose-style: prefer publisher image) |
| `harbor.dataknife.net/library/palserver:...` | Optional Harbor mirror of the official image (cluster pull policy) |
| `thijsvanloef/palworld-server-docker` | Optional community image: env-driven config, backups, RCON helpers |
| `johnnyknighten/palworld-server` | Another community env→INI option |

Operator default: `ghcr.io/pocketpairjp/palserver:latest` (pin a version tag or digest in production).

### Why not a DataKnifeAI `palworld-server-docker` repo?

A separate DataKnifeAI image project (Dockerfile + Harbor CI) is only warranted if Pocketpair stopped publishing containers. That is **not** the case today — use the official GHCR image and keep the operator focused on Kubernetes reconciliation.

## Configuration models

### Official image (default)

- **CLI args** for port / threading (compose `command:` → `-port=8211`, `-UseMultithreadForDS`, …)
- **INI** for name, passwords, RCON, crossplay, balance: `Pal/Saved/Config/LinuxServer/PalWorldSettings.ini`
- Operator should generate/mount that INI (ConfigMap or template) similar to Windrose’s `ServerDescription.json`, not rely on community env vars

### Community image (optional)

Env vars map to INI / launch options. Highly recommended: `PUID`, `PGID`, `PORT`, `PLAYERS`.

| Variable | Default | Maps to |
|----------|---------|---------|
| `SERVER_NAME` | — | Display name |
| `SERVER_DESCRIPTION` | — | Description |
| `SERVER_PASSWORD` | — | Join password |
| `ADMIN_PASSWORD` | — | Admin / RCON |
| `PLAYERS` | 16 | Max players (1–32) |
| `PORT` | 8211 | Game UDP port |
| `QUERY_PORT` | 27015 | Steam query |
| `RCON_ENABLED` | false* | Enable RCON (*enable for K8s graceful stop) |
| `RCON_PORT` | 25575 | RCON TCP |
| `REST_API_ENABLED` | true | REST API |
| `REST_API_PORT` | 8212 | REST TCP |
| `MULTITHREADING` | false | Up to ~4 threads useful |
| `COMMUNITY` | false | Community browser listing |
| `PUBLIC_IP` / `PUBLIC_PORT` | auto | Community listing; set to Gateway address/port in K8s |
| `UPDATE_ON_BOOT` | true | Required on first install |
| `BACKUP_ENABLED` | true | Cron backups inside container |
| `CROSSPLAY_PLATFORMS` | Steam,Xbox,PS5,Mac | Crossplay allow-list |

Passwords must come from Kubernetes Secrets, not CR plaintext.

## Resource guidance

Community/hosting consensus (not official Pocketpair SLAs):

- Light private: ~8–16 GiB RAM, 4 vCPU
- Typical multiplayer with bases: **16 GiB+** RAM
- UE5 + Pal AI scales with player count, base count, and `BaseCampWorkerMaxNum`

Operator should auto-select resources from `maxPlayers` with override via `spec.resources` (same UX as Windrose).

## Graceful lifecycle

- Enable RCON (INI or community env) so shutdown can save cleanly on SIGTERM
- Set `terminationGracePeriodSeconds` high enough (e.g. 60–120s)
- Prefer careful update policy in prod (unexpected image/Steam updates mid-session)

## Crossplay

Dedicated servers support Steam / Xbox / PS5 / Mac via `CrossplayPlatforms` (INI) or community image `CROSSPLAY_PLATFORMS`.
