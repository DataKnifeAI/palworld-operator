# Palworld dedicated server notes

Operator-relevant detail for the official Pocketpair image and optional community images.

Sources:
- [Deploy dedicated server](https://docs.palworldgame.com/getting-started/deploy-dedicated-server)
- [Configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/)
- [Official Docker image (Pocketpair)](https://github.com/pocketpairjp/palworld-dedicated-server-docker) — `ghcr.io/pocketpairjp/palserver`
- [thijsvanloef/palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker) (community alternative)

## Official distribution

| Item | Value |
|------|-------|
| Image | `ghcr.io/pocketpairjp/palserver` |
| Source | [pocketpairjp/palworld-dedicated-server-docker](https://github.com/pocketpairjp/palworld-dedicated-server-docker) |
| Docs | [Palworld Server Guide](https://tech.palworldgame.com/) / [requirements](https://tech.palworldgame.com/getting-started/requirements) |
| Tags | Versioned (e.g. `v1.0.0.100427`) and `latest` |

SteamCMD App ID **2394010** is the underlying dedicated server; the official image packages that build.

**No DataKnifeAI custom server-image repository is required** while this official image is maintained. Optional Harbor mirror of the game image is an ops step — see [GITLAB_MIRROR.md](GITLAB_MIRROR.md). The **operator** image publishes separately to `harbor.dataknife.net/library/palworld-operator`.

### Official image layout

| Path | Purpose |
|------|---------|
| `/pal/Package/PalServer.sh` | Server entrypoint (via `/pal/helper.sh` in compose samples) |
| `/pal/Package/DefaultPalWorldSettings.ini` | Defaults template (do **not** edit for live config) |
| `/pal/Package/Pal/Saved` | Persist this directory (saves + `Config/LinuxServer/`) |

Compose samples mount `./Saved` → `/pal/Package/Pal/Saved` and pass CLI args (`-port=8211`, multithreading). Gameplay settings live in `PalWorldSettings.ini` under the Saved mount.

## Ports

| Port | Proto | Role | Operator notes |
|------|-------|------|----------------|
| 8211 | UDP | Game traffic | Primary client connect; expose via UDPRoute |
| 27015 | UDP | Steam query | Community browser / Steam; UDPRoute when listing |
| 25575 | TCP | RCON | Enable for graceful stop/save |
| 8212 | TCP | REST API | Useful for ops; **do not** public-forward casually |

Official compose examples often expose **8211/UDP** only; query/RCON/REST still exist when enabled in settings.

## Persistence

| Path | Purpose |
|------|---------|
| `Pal/Saved/SaveGames/` | World saves (must be on PVC) |
| `Pal/Saved/Config/LinuxServer/` | `PalWorldSettings.ini`, related INI |
| Official image mount | `/pal/Package/Pal/Saved` |
| Community image mount | `/palworld` (install + saves + backups) |

Recommended PVC size: start at **50–100Gi** (worlds grow with bases/Pals). Stop the server before mutating settings files; shutdown overwrites in-memory settings.

## Container image options

| Image | Role |
|-------|------|
| **`ghcr.io/pocketpairjp/palserver`** | **Operator default** — official Pocketpair image |
| `harbor.dataknife.net/library/palserver:...` | Optional Harbor mirror |
| `thijsvanloef/palworld-server-docker` | Optional community image (env-driven config) |
| `johnnyknighten/palworld-server` | Another community env→INI option |

Pin a version tag or digest in production. A separate DataKnifeAI game-image project is only warranted if Pocketpair stopped publishing containers.

## Configuration models

### Official image (default)

- **CLI args** for port / threading (`-port=8211`, `-UseMultithreadForDS`, …)
- **INI** for name, passwords, RCON, crossplay, balance: `Pal/Saved/Config/LinuxServer/PalWorldSettings.ini`
- Operator generates/mounts that INI (ConfigMap), not community env vars

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
| `PUBLIC_IP` / `PUBLIC_PORT` | auto | Set to Gateway address/port in K8s |
| `UPDATE_ON_BOOT` | true | Required on first install |
| `BACKUP_ENABLED` | true | Cron backups inside container |
| `CROSSPLAY_PLATFORMS` | Steam,Xbox,PS5,Mac | Crossplay allow-list |

Passwords must come from Kubernetes Secrets, not CR plaintext.

### Credentials: bring-your-own vs auto-generate

| Mode | Spec | Secret |
|------|------|--------|
| **Bring-your-own** | `adminPasswordSecretRef` + `serverPasswordSecretRef` | You create the Opaque Secret first |
| **Auto-generate** | `generateSecrets: true` (omit refs, or keep refs pointing at the managed Secret) | Operator creates `{metadata.name}-secrets` (override with `credentialsSecretName`) |

Auto-gen behavior:

- Creates an Opaque Secret owned by the `PalworldServer` (OwnerReference)
- Fills missing/empty keys `server-password` (join) and `admin-password` (RCON/admin) with random strong passwords
- **Never overwrites** existing non-empty keys
- Status sets `credentialsSecretName` and `credentialsGenerated: true` — **no plaintext** in status

Read passwords (placeholder names match the sample CR):

```shell
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.server-password}' | base64 -d; echo
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.admin-password}' | base64 -d; echo
```

For auto-gen, substitute the Secret name from
`.status.credentialsSecretName` (default `{cr-name}-secrets`).

### CR field mapping

| Concern | Official (INI / CLI) | Community env | CR field |
|---------|----------------------|---------------|----------|
| Display name | `ServerName` in INI | `SERVER_NAME` | `spec.serverName` |
| Max players | `ServerPlayerMaxNum` | `PLAYERS` | `spec.maxPlayers` |
| Game port | `-port=` CLI | `PORT` | `spec.gamePort` (default 8211) |
| Query port | INI / server args | `QUERY_PORT` | `spec.queryPort` (default 27015) |
| RCON | `RCONEnabled` / `RCONPort` | `RCON_*` | `spec.rcon` |
| REST API | INI | `REST_API_*` | `spec.restAPI` |
| Passwords | INI fields | `SERVER_PASSWORD`, `ADMIN_PASSWORD` | Secret refs **or** `spec.generateSecrets` |
| Community list | INI + public bind | `COMMUNITY`, `PUBLIC_*` | `spec.community` + gateway |
| Crossplay | `CrossplayPlatforms` | `CROSSPLAY_PLATFORMS` | `spec.crossplayPlatforms` |

## Resource guidance

Community/hosting consensus (not official Pocketpair SLAs):

| Players | Suggested memory | Notes |
|---------|------------------|-------|
| 1–8 | 8–16 Gi | Light private world |
| 8–16 | 16–24 Gi | Typical dedicated |
| 16–32 | 24–32+ Gi | Public / large bases; UE5 scales with structures |

CPU: prefer multi-core; official CLI includes `-UseMultithreadForDS` (community: `MULTITHREADING=true`). Override via `spec.resources`. Sample CR uses modest requests for ~8Gi nodes.

## Graceful lifecycle

- Enable RCON so shutdown can save cleanly on SIGTERM
- Set `terminationGracePeriodSeconds` high enough (e.g. 60–120s)
- Prefer careful update policy in prod (unexpected image/Steam updates mid-session)

## Updating the game server (Steam / patches)

Pocketpair’s official image and community SteamCMD images behave differently. This operator defaults to the official image.

### Official image (`ghcr.io/pocketpairjp/palserver`) — **pin / bump the image tag**

Official Pocketpair guidance ([Updating the Dedicated Server](https://github.com/pocketpairjp/palworld-dedicated-server-docker#updating-the-dedicated-server)):

1. **Back up** the Saved PVC (or snapshot) before updating.
2. **Stop** the server (scale / roll the Deployment — the operator uses `Recreate`).
3. **Change the image tag** in `spec.serverImage` to the game version (e.g. `ghcr.io/pocketpairjp/palserver:v1.0.0.100427`), or move `latest` only after confirming the published tag matches the client.
4. **Start** again and confirm REST `/v1/api/info` `version` and that `worldguid` / days still match the previous world.

| Mechanism | Official Pocketpair image | Community (e.g. thijsvanloef) |
|-----------|---------------------------|--------------------------------|
| How game bits update | **New container image tag** published by Pocketpair | **SteamCMD** inside the container on start |
| `UPDATE_ON_BOOT` / `spec.updateOnBoot` | **Not used** (operator only injects that env for community images) | Controls SteamCMD update/install on boot |
| Pinning a version | Tag or digest on `spec.serverImage` | `TARGET_MANIFEST_ID` / skip-update flags on community images |
| Rebuild required? | No — pull Pocketpair’s published image | No — SteamCMD downloads app **2394010** into the volume |

**Operator practice:** prefer an explicit version tag (or digest) in production; treat `latest` as convenience only. After a game patch, bump `spec.serverImage` when Pocketpair publishes a matching `palserver` tag ([GHCR package](https://github.com/pocketpairjp/palworld-dedicated-server-docker/pkgs/container/palserver)). Do **not** expect an in-container SteamCMD update from the official image.

`spec.updateOnBoot` remains for optional community images; it does not make the official image self-update.

### World selection across restarts

Palworld loads the world named in `Pal/Saved/Config/LinuxServer/GameUserSettings.ini` under `[/Script/Pal.PalGameLocalSettings]` → `DedicatedServerName=<SaveGames/0 folder name>`. That folder name is the world GUID (also reported by REST `worldguid`).

If `DedicatedServerName` is missing or wrong after a roll, the server creates a **new** empty world and leaves the old folder on the PVC. Keep `GameUserSettings.ini` on the Saved PVC (the operator’s `seed-settings` init only overwrites `PalWorldSettings.ini`). After any reschedule, confirm REST `worldguid` still matches the intended `SaveGames/0/<guid>/` directory.

## Crossplay

Dedicated servers support Steam / Xbox / PS5 / Mac via `CrossplayPlatforms` (INI) or community image `CROSSPLAY_PLATFORMS`.

## Connecting from the game client

Player-facing join flow (Join Multiplayer Game, `connectionAddress:connectionPort`, join vs admin password, community browser): see [CONNECT.md](CONNECT.md).
