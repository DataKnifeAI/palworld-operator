# Palworld dedicated server — operator research notes

Sources:
- [Deploy dedicated server](https://docs.palworldgame.com/getting-started/deploy-dedicated-server) (official)
- [Configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/) (official)
- [thijsvanloef/palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker) (de facto container image)

## Official distribution

- Dedicated server is a **Steam tool / SteamCMD** download (App ID **2394010**), not an official Pocketpair container image.
- Linux path after install: `steamapps/common/PalServer/`
- Config template: `DefaultPalWorldSettings.ini` → copy to `Pal/Saved/Config/LinuxServer/PalWorldSettings.ini`
- Do **not** edit `DefaultPalWorldSettings.ini` (ignored). Stop the server before editing live INI or shutdown overwrites changes.

## Ports

| Port | Proto | Role | Operator notes |
|------|-------|------|----------------|
| 8211 | UDP | Game traffic | Primary client connect; expose via UDPRoute |
| 27015 | UDP | Steam query | Community browser / Steam; expose via UDPRoute |
| 25575 | TCP | RCON | Admin; Docker image uses RCON for graceful `docker stop` |
| 8212 | TCP | REST API | Useful for health/player logging; **do not** public-forward |

## Persistence

| Path | Purpose |
|------|---------|
| `Pal/Saved/SaveGames/` | World saves (must be on PVC) |
| `Pal/Saved/Config/LinuxServer/` | `PalWorldSettings.ini`, related INI |
| Docker image: `/palworld` | Single volume covering install + saves + backups |

Recommended PVC size: **≥ 50Gi** (saves + native backups grow; Windrose uses 35Gi as a floor for a lighter game).

## Container image options

| Image | Notes |
|-------|-------|
| **`thijsvanloef/palworld-server-docker`** | Preferred default: env-driven config, backups, RCON/REST, documented k8s examples |
| `johnnyknighten/palworld-server` | Alternative env→INI generator |
| Custom SteamCMD image | Possible later; more maintenance |

Operator default: `thijsvanloef/palworld-server-docker:latest` (pin digests in production).

## Key environment variables (Docker image)

Highly recommended: `PUID`, `PGID`, `PORT`, `PLAYERS`.

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

- Enable RCON so the Docker entrypoint can save/shutdown on SIGTERM
- Set `terminationGracePeriodSeconds` high enough (e.g. 60–120s; compose examples use `stop_grace_period: 30s` as a minimum)
- Prefer `UPDATE_ON_BOOT` careful in prod (unexpected Steam updates mid-session)

## Crossplay

Dedicated servers support Steam / Xbox / PS5 / Mac via `CrossplayPlatforms` / image `CROSSPLAY_PLATFORMS`.
