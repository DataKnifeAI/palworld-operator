# Palworld dedicated server notes

Operator-relevant detail for the official Pocketpair image and optional community images.

Sources:
- [Deploy dedicated server](https://docs.palworldgame.com/getting-started/deploy-dedicated-server)
- [Configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/)
- [REST API](https://docs.palworldgame.com/category/rest-api/) — replacement for RCON (v1.0.3). Intro URL is [`/api/rest-api/palwold-rest-api`](https://docs.palworldgame.com/api/rest-api/palwold-rest-api) (official typo; spelled-correct 404s)
- [RCON](https://docs.palworldgame.com/api/rcon/) — **deprecated**; scheduled to stop in an upcoming update
- [Installing mods on a server](https://docs.palworldgame.com/settings-and-operation/mod/) — **Windows-only** official Workshop loader
- [Yorkhost PAK / UE4SS notes](https://yorkhost.fr/docs/en/palworld/mods-ue4ss) — community Linux vs Win64 (not Pocketpair)
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
| `/pal/Package/Mods` | **Not in the image.** Official persist is Saved only. Opt-in `spec.mods` mounts a dedicated PVC here (next to `PalServer.sh`) |

Compose samples mount `./Saved` → `/pal/Package/Pal/Saved` and pass CLI args (`-port=8211`, multithreading). Gameplay settings live in `PalWorldSettings.ini` under the Saved mount. The official image does **not** ship a `Mods/` tree (confirmed on `palserver:v1.0.3.101283`).

**Local / minimal PC (this repo):** [`compose/`](../compose/) + [LOCAL.md](LOCAL.md) — Docker Compose, no Kubernetes, resource caps and `.env` password seed.

## Ports

| Port | Proto | Role | Operator notes |
|------|-------|------|----------------|
| 8211 | UDP | Game traffic | Primary client connect; expose via UDPRoute |
| 27015 | UDP | Steam query | Community browser / Steam; UDPRoute when listing |
| 25575 | TCP | RCON (deprecated) | Legacy listener; ClusterIP default-on until Pocketpair removes it. Operator does **not** use RCON commands. Not required for stop/save. |
| 8212 | TCP | REST API | Replacement for RCON. Operator uses REST announce + admin basic auth. **Do not** public-forward casually |
| 8088 | TCP/HTTP | Optional Server Manager | `spec.serverManager` (`spec.modManager` alias); same Gateway VIP; **basic auth required** |

Official compose examples often expose **8211/UDP** only; query/RCON/REST still exist when enabled in settings. REST is the documented admin API; RCON remains ClusterIP-internal as a legacy listener.

## Persistence

| Path | Purpose |
|------|---------|
| `Pal/Saved/SaveGames/` | World saves (must be on PVC) |
| `Pal/Saved/Config/LinuxServer/` | `PalWorldSettings.ini`, related INI |
| Official image mount | `/pal/Package/Pal/Saved` |
| Community image mount | `/palworld` (install + saves + backups) |

Recommended PVC size: start at **50–100Gi** (worlds grow with bases/Pals). Stop the server before mutating settings files; shutdown overwrites in-memory settings.

### Mods — Linux vs Windows (honest)

This operator’s **default image is Linux** [`ghcr.io/pocketpairjp/palserver`](https://github.com/pocketpairjp/palworld-dedicated-server-docker). Pocketpair’s [official dedicated-server mods](https://docs.palworldgame.com/settings-and-operation/mod/) (Workshop loader, `Mods/PalModSettings.ini`, `-workshopdir`) are **Windows-only**. They do **not** load on Linux PalServer.

Live filesystem: the image has **no** `Mods/` directory. The saves PVC is **`/pal/Package/Pal/Saved` only**. Opt-in `spec.mods` is a *second* PVC so files never mix into world saves.

| Kind | Linux (this operator) | Windows PalServer |
|------|----------------------|-------------------|
| Official Workshop (`Mods/`, `PalModSettings.ini`, `-workshopdir`) | **Not loaded** — mount is forward-looking for a future Pocketpair Linux path | **Yes** — [Pocketpair](https://docs.palworldgame.com/settings-and-operation/mod/) |
| Client mods (`bAllowClientMod`) | Join **policy** only — does not install or host client mods | Same INI flag |
| Community `.pak` / `.sig` under `Pal/Content/Paks/` | **Possible** if version-matched ([Yorkhost](https://yorkhost.fr/docs/en/palworld/mods-ue4ss)) | Yes |
| UE4SS (`UE4SS.dll`, Lua under `Pal/Binaries/Win64/`) | **No** — needs Windows/Win64 DLL injection / Proton. Not `PalServer-Linux-Shipping` | Yes (different stack) |

**Client mods** are a join policy, not server content. Set `spec.optionSettings.bAllowClientMod` (`"True"` / `"False"`) so PC clients may or may not connect with their own mods. The server does not download, store, or inject those files. Consoles cannot load PC client mods.

**Server content mods** (Workshop on Windows, or Linux PAK drops) change what the dedicated process serves. They can **lock consoles out** of a crossplay world — keep `spec.crossplayPlatforms` and community listing in mind before adding PAKs.

### Mods PVC (`spec.mods`) — opt-in, two layouts

One dedicated PVC stages files at the real paths without mixing them into the saves volume. Enabling `spec.mods` rolls the game pod (Recreate). Leave it off on a live world unless you accept that roll.

| Kind | Works on official Linux image? | Path |
|------|--------------------------------|------|
| **Pocketpair Workshop** (`Mods/`, `PalModSettings.ini`, `Info.json`) | **No today** — Windows-only loader. PVC still creates `/pal/Package/Mods` (image does not ship it). `PalServer.sh` does not pass `-workshopdir`. | PVC root → `/pal/Package/Mods` |
| **PAK files** (`.pak` + optional `.sig`) | **Yes (community)** — native Linux loads version-matched PAKs under `Pal/Content/Paks/`. Config-only tweaks are `PalWorldSettings.ini` (`spec.optionSettings`), not this PVC. | PVC `paks/~WorkshopMods` and `paks/LogicMods` overlay those **subfolders only** |
| **UE4SS** | **No** — do not mount Win64 over this image. | n/a |

The image already has `Pal/Content/Paks/Pal-LinuxServer.pak`. We **never** replace the whole `Paks/` directory (that would hide the official pak and the server would not start). Overlays are `~WorkshopMods` and `LogicMods` only — the same subfolders Pocketpair’s Windows loader deploys into.

**Backup the saves PVC and pin `spec.serverImage` before adding PAKs.** A version-incompatible `.pak` can prevent the server from starting.

| Item | Value |
|------|-------|
| PVC | `{metadata.name}-mods` (OwnerReference; does not replace `{name}-files`) |
| Mods mount | `/pal/Package/Mods` (creates the directory; official image does not ship it) |
| Workshop packages | `{mods}/Workshop/<any-folder>/Info.json` |
| Settings file | `{mods}/PalModSettings.ini` |
| PAK overlays (default on) | `{mods}/paks/~WorkshopMods` → `/pal/Package/Pal/Content/Paks/~WorkshopMods` |
| | `{mods}/paks/LogicMods` → `/pal/Package/Pal/Content/Paks/LogicMods` |
| Default size | `10Gi` (`spec.mods.storage.size`) |
| StorageClass | `spec.mods.storage.storageClassName`, else `spec.storageClassName` |
| `-workshopdir` | **Off by default.** `useWorkshopDirArg: true` appends `-workshopdir={workshopDir}`. Official Linux args omit it. |

```yaml
spec:
  mods:
    enabled: true
    storage:
      size: 10Gi
      # storageClassName: truenas-csi-nfs   # defaults to spec.storageClassName
    # path: /pal/Package/Mods
    # workshopDir: /pal/Package/Mods/Workshop
    # useWorkshopDirArg: false
    # paksOverlay: true                     # default; set false for Mods/ only
    # activeModList:                        # optional seed of PalModSettings.ini
    #   - GamingCattiva
```

Copy files onto the PVC via a throwaway pod (folder name is organizational; Workshop `PackageName` is in `Info.json`):

```shell
kubectl -n game-servers apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: palworld-mods-copy
spec:
  restartPolicy: Never
  containers:
    - name: copy
      image: busybox:1.37
      command: ["sleep", "3600"]
      volumeMounts:
        - name: mods
          mountPath: /mods
  volumes:
    - name: mods
      persistentVolumeClaim:
        claimName: palworld-server-mods
EOF
# Official Workshop tree (Windows loader / future Linux):
kubectl -n game-servers cp ./MyMod palworld-mods-copy:/mods/Workshop/MyMod
# Linux PAK overlay (does not hide Pal-LinuxServer.pak):
kubectl -n game-servers cp ./MyMod.pak palworld-mods-copy:/mods/paks/~WorkshopMods/MyMod.pak
kubectl -n game-servers delete pod palworld-mods-copy
```

Optional `spec.mods.activeModList` seeds `PalModSettings.ini` on official-image starts. Restart the game Deployment after changing packages.

### Optional Server Manager (`spec.serverManager`)

Authenticated admin UI on the **same Gateway VIP** as game UDP (HTTPRoute, not a LoadBalancer on the game pod). **Off by default.** Journey: **Overview → Controls → Saves → Mods**. The sidecar shares the game pod and proxies Palworld REST on `http://127.0.0.1:8212/v1/api` — REST is **not** public-routed. RCON is [deprecated](https://docs.palworldgame.com/api/rcon/); use REST ([official docs](https://docs.palworldgame.com/api/rest-api/palwold-rest-api) — Pocketpair’s spelling). `spec.modManager` is a deprecated alias for the same sidecar.

| Item | Value |
|------|-------|
| Enable | `spec.serverManager.enabled: true` (or `spec.modManager.enabled`) |
| URL | `http://<gateway.address>:<port>/` — default port **8088** (`spec.serverManager.port`) |
| Auth | HTTP basic auth — username `admin`, password = credentials Secret key `admin-password` |
| Sidecar | `/server-manager` from the **operator** image (`/mod-manager` is kept as a copy) |
| Overview | REST `GET /info`, `/metrics`, `/players` (version, worldguid, FPS, players, days/uptime/basecamps when present) |
| Controls | REST announce / save / shutdown (confirm); Recreate-roll restart |
| Saves | Zip of `SaveGames/` (optional `Config/LinuxServer`; INI passwords redacted). Upload replaces the live world (confirm). Mounts the game PVC at `/saves`. |
| Mods tab | List/upload/download/delete on the mods PVC (`/mods`). Needs `spec.mods.enabled`. |
| Restart | UI button PATCHes the game Deployment (Recreate). **Players disconnect** until Ready. The UI pod restarts too. |
| RBAC | Namespaced Role: `get`/`patch`/`update` **only** that Deployment |

**Do not enable on a live CR without a maintenance window** — adding the sidecar Recreate-rolls the game pod.

The operator image must include `/server-manager` (Dockerfile also copies `/mod-manager`). Rebuild/push Harbor so the sidecar command works. Pin `spec.serverManager.image` to a digest/tag if the game namespace cannot pull `:latest`. Private Harbor may need `spec.imagePullSecrets`.

Linux PalServer still does **not** load official Workshop / `PalModSettings.ini`. Use the Mods tab to drop version-matched `.pak` files under `paks/~WorkshopMods` and `paks/LogicMods`. Path traversal outside the mods and saves mounts is rejected. There is no unauthenticated mode.

```yaml
spec:
  serverManager:
    enabled: true
    port: 8088
    # image: harbor.dataknife.net/library/palworld-operator:latest
  # mods:
  #   enabled: true          # needed only for the Mods tab
  #   storage:
  #     size: 10Gi
```

**Honest expectations:** Linux PAK drops may load if they match the pinned server version. Pocketpair Workshop + `PalModSettings.ini` + `-workshopdir` + UE4SS will not load on this image until Pocketpair ships a Linux loader (or you run a different Windows/Proton stack — out of scope).

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
- **INI** for name, passwords, REST/RCON, crossplay, balance: `Pal/Saved/Config/LinuxServer/PalWorldSettings.ini`
- Operator builds that INI in a ConfigMap and the `seed-settings` init copies it onto the PVC **every pod start** (overwrite is intentional — keep desired settings in the CR)

### Game balance / features (`spec.optionSettings`)

`spec.optionSettings` is a **passthrough** `map[string]string` of [PalWorldSettings.ini OptionSettings](https://docs.palworldgame.com/settings-and-operation/configuration/) keys (balance, features, performance). Values are INI literals (`"True"`, `"1.5"`, `None`, …). Unknown keys are kept for newer game versions.

**Official parameter list (source of truth):** [Configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/). Do not copy Pocketpair’s full key list into this repo.

| Behavior | Detail |
|----------|--------|
| Passthrough | Any OptionSettings key → ConfigMap `PalWorldSettings.ini` |
| Source of truth | ConfigMap rebuilt each reconcile; PVC re-seeded on each roll |
| Official image | Full map → INI (recommended) |
| Community image | Best-effort env mapping for common keys only |

**Reserved / overridden** — these keys come from dedicated CR fields and **always win** over the same key in `optionSettings`:

| INI key | CR field |
|---------|----------|
| `ServerName` / `ServerDescription` / `ServerPlayerMaxNum` | `spec.serverName`, `serverDescription`, `maxPlayers` |
| `ServerPassword` / `AdminPassword` | Secret refs or `spec.generateSecrets` — **never** put passwords in the map |
| `PublicPort` / `PublicIP` | `spec.gamePort` / community public bind |
| `RCONEnabled` / `RCONPort` | `spec.rcon` |
| `RESTAPIEnabled` / `RESTAPIPort` | `spec.restAPI` |
| `CrossplayPlatforms` | `spec.crossplayPlatforms` |

Example:

```yaml
spec:
  optionSettings:
    bExistPlayerAfterLogout: "True"   # sleep in-place on logout
    WorkSpeedRate: "1.5"
    DeathPenalty: None
    # bAllowClientMod: "True"         # join policy only — does not install mods
```

**Apply** (world / PVC stay intact — do **not** delete the CR or PVC; keep `dedicatedServerName` / `worldguid`):

1. Merge-patch the CR (`optionSettings` keys merge; other spec fields are left alone).
2. Wait for reconcile to rebuild the ConfigMap.
3. Roll the **game** Deployment so the `seed-settings` init re-copies `PalWorldSettings.ini` onto the PVC (required for the official image). Players disconnect briefly.

```shell
kubectl -n game-servers patch palworldserver palworld-server --type=merge \
  -p '{"spec":{"optionSettings":{"bExistPlayerAfterLogout":"True"}}}'
kubectl -n game-servers rollout restart deploy/palworld-server
```

### Community image (optional)

Env vars map to INI / launch options. Highly recommended: `PUID`, `PGID`, `PORT`, `PLAYERS`.
`spec.optionSettings` maps only **known** keys to community env vars (e.g. `ExpRate` → `EXP_RATE`); unmapped keys are ignored. For complete OptionSettings coverage use the **official** Pocketpair image.

| Variable | Default | Maps to |
|----------|---------|---------|
| `SERVER_NAME` | — | Display name |
| `SERVER_DESCRIPTION` | — | Description |
| `SERVER_PASSWORD` | — | Join password |
| `ADMIN_PASSWORD` | — | Admin / REST basic auth (legacy RCON used the same password) |
| `PLAYERS` | 16 | Max players (1–32) |
| `PORT` | 8211 | Game UDP port |
| `QUERY_PORT` | 27015 | Steam query |
| `RCON_ENABLED` | false* | Legacy RCON (*operator default-on in K8s for compatibility; not used for stop/save) |
| `RCON_PORT` | 25575 | Legacy RCON TCP |
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
- Fills missing/empty keys `server-password` (join) and `admin-password` (in-game admin / REST basic auth) with random strong passwords
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
| RCON (deprecated) | `RCONEnabled` / `RCONPort` | `RCON_*` | `spec.rcon` (ClusterIP default-on; unused by operator) |
| REST API | INI | `REST_API_*` | `spec.restAPI` |
| Passwords | INI fields | `SERVER_PASSWORD`, `ADMIN_PASSWORD` | Secret refs **or** `spec.generateSecrets` |
| Community list | INI + public bind | `COMMUNITY`, `PUBLIC_*` | `spec.community` + gateway |
| Crossplay | `CrossplayPlatforms` | `CROSSPLAY_PLATFORMS` | `spec.crossplayPlatforms` |
| Balance / features | Extra `OptionSettings=(…)` keys | Known keys → env (partial) | `spec.optionSettings` |
| Workshop / mods | `/pal/Package/Mods` + Paks subpath overlays | `/palworld/Mods` + `/palworld/Pal/Content/Paks/…` | `spec.mods` (opt-in) |
| Server Manager UI | HTTP on Gateway VIP (sidecar) | same | `spec.serverManager` (`spec.modManager` alias) |

## Resource guidance

Community/hosting consensus (not official Pocketpair SLAs):

| Players | Suggested memory | Notes |
|---------|------------------|-------|
| 1–8 | 8–16 Gi | Light private world |
| 8–16 | 16–24 Gi | Typical dedicated |
| 16–32 | 24–32+ Gi | Public / large bases; UE5 scales with structures |

CPU: prefer multi-core; official CLI includes `-UseMultithreadForDS` (community: `MULTITHREADING=true`). Override via `spec.resources`. Sample CR uses modest requests for ~8Gi nodes.

## Graceful lifecycle

Pocketpair documents [RCON as deprecated](https://docs.palworldgame.com/api/rcon/) (v1.0.3; scheduled to stop in an upcoming update). Use the [REST API](https://docs.palworldgame.com/category/rest-api/) instead ([intro](https://docs.palworldgame.com/api/rest-api/palwold-rest-api) — official URL typo; spelled-correct 404s). Auth is HTTP basic, user `admin`, password = `AdminPassword`.

Documented admin APIs:

- [`POST /v1/api/save`](https://docs.palworldgame.com/api/rest-api/save/) — save the world
- [`POST /v1/api/shutdown`](https://docs.palworldgame.com/api/rest-api/shutdown/) — graceful shutdown with wait/message
- [`POST /v1/api/announce`](https://docs.palworldgame.com/api/rest-api/announce/) — in-game broadcast

This operator:

- Uses REST announce + admin basic auth (`notifyPlayers`)
- Does **not** issue RCON commands
- Currently still stops pods with **SIGTERM** + `spec.terminationGracePeriodSeconds` (default 60; raise to 60–120s if needed)
- Does **not** yet call REST `/save` or `/shutdown` before Recreate (optional later)

`spec.rcon` stays default-on with a ClusterIP Service as a legacy listener until Pocketpair removes RCON. It is not required for stop/save.

Prefer a careful update policy in prod (unexpected image/Steam updates mid-session).

## Updating the game server (Steam / patches)

Pocketpair’s official image and community SteamCMD images behave differently. This operator defaults to the official image.

### Official image (`ghcr.io/pocketpairjp/palserver`) — **pin / bump the image tag**

Official Pocketpair guidance ([Updating the Dedicated Server](https://github.com/pocketpairjp/palworld-dedicated-server-docker#updating-the-dedicated-server)):

1. **Back up** the Saved PVC (or snapshot) before updating.
2. **Stop** the server (scale / roll the Deployment — the operator uses `Recreate`).
3. **Change the image tag** in `spec.serverImage` to the game version (e.g. `ghcr.io/pocketpairjp/palserver:v1.0.1.100619`), or move `latest` only after confirming the published tag matches the client (and that the node actually pulls a new digest — prefer `imagePullPolicy: Always` for one roll, or pin an explicit tag).
4. **Start** again and confirm REST `/v1/api/info` `version` and that `worldguid` / days still match the previous world.

| Mechanism | Official Pocketpair image | Community (e.g. thijsvanloef) |
|-----------|---------------------------|--------------------------------|
| How game bits update | **New container image tag** published by Pocketpair | **SteamCMD** inside the container on start |
| `UPDATE_ON_BOOT` / `spec.updateOnBoot` | **Not used** (operator only injects that env for community images) | Controls SteamCMD update/install on boot |
| Pinning a version | Tag or digest on `spec.serverImage` | `TARGET_MANIFEST_ID` / skip-update flags on community images |
| Rebuild required? | No — pull Pocketpair’s published image | No — SteamCMD downloads app **2394010** into the volume |

**Operator practice:** prefer an explicit version tag (or digest) in production; treat `latest` as convenience only. After a game patch, bump `spec.serverImage` when Pocketpair publishes a matching `palserver` tag ([GHCR package](https://github.com/pocketpairjp/palworld-dedicated-server-docker/pkgs/container/palserver)). Do **not** expect an in-container SteamCMD update from the official image.

`spec.updateOnBoot` remains for optional community images; it does not make the official image self-update.

### Opt-in auto-update (`spec.update`)

Set `spec.update.autoUpdateImage: true` to have the operator:

1. **Discover latest** — anonymous GHCR OCI tag list for `spec.update.imageRepository` (default `ghcr.io/pocketpairjp/palserver`); parse `vX.Y.Z.W`; ignore `latest` for comparison (or treat a floating pin as behind any newer version tag). Lookups are cached (~30m).
2. **Compare** — pinned image tag, else REST `/v1/api/info` `version`, vs newest tag. Status: `desiredImage`, `runningVersion`, `latestAvailableVersion`, `updateAvailable`, `lastImageCheckTime`.
3. **Apply safely** — patch `spec.serverImage` to `repo:vX.Y.Z.W` (never leave a floating `latest` after an auto bump). Requires a world pin (`spec.dedicatedServerName` or learned REST `worldguid`) once the server has been Ready. Default `onlyWhenEmpty: true` defers while REST `/v1/api/metrics` `currentplayernum > 0`.
4. **Schedules** (optional, 5-field cron, `spec.update.timeZone` default **UTC**):
   - `checkInterval` (default `6h`) when `checkSchedule` unset — how often to poll GHCR
   - `checkSchedule` — cron minutes when polling is allowed (replaces interval)
   - `applySchedule` — cron must match the **current minute** for a roll (maintenance window); omit to apply whenever idle/safe
5. **In-game notice** (optional) — `notifyPlayers: true` plans `status.plannedApplyTime` (now + max schedule, or next `applySchedule`) and sends staged REST [`POST /v1/api/announce`](https://docs.palworldgame.com/api/rest-api/announce/) warnings as reconcile hits each boundary. Default schedule: `60m`, `30m`, `15m`, `5m`, `1m`, `30s`, then a short `10s` countdown immediately before the image patch. Override with `notifySchedule` (Go durations). Legacy `notifyLeadTime` (e.g. `2m`) is a single-stage schedule when `notifySchedule` is empty. Placeholders in `notifyMessage`: `{version}`, `{image}`, `{remaining}`. Status tracks `announcedNotifyStages` so stages are not re-sent. `onlyWhenEmpty` still gates the **roll**, not the warnings (players online get the notices). Reconcile never blocks for the full lead — it requeues until the next stage. Pocketpair documents **RCON as deprecated**; this operator uses REST announce + admin basic auth and does not issue RCON commands. REST `/save` and `/shutdown` are documented but not called by the operator yet (rolls still use SIGTERM + `terminationGracePeriodSeconds`).

When `autoUpdateImage` is false, the operator never mutates `spec.serverImage` (manual pins win).

Example:

```yaml
spec:
  dedicatedServerName: "YOUR-WORLD-GUID"   # or leave empty to learn from REST
  update:
    autoUpdateImage: true
    checkInterval: 6h
    # checkSchedule: "0 */6 * * *"
    applySchedule: "0 4 * * 1-5"           # 04:00 UTC Mon–Fri
    timeZone: UTC
    onlyWhenEmpty: true
    notifyPlayers: true
    # notifySchedule: ["60m","30m","15m","5m","1m","30s","10s"]  # default when unset + no notifyLeadTime
    # notifyMessage: "[Server] Update {version} — restart in {remaining}"
    # notifyLeadTime: 2m                   # deprecated single-stage fallback
```

### World selection across restarts

Palworld loads the world named in `Pal/Saved/Config/LinuxServer/GameUserSettings.ini` under `[/Script/Pal.PalGameLocalSettings]` → `DedicatedServerName=<SaveGames/0 folder name>`. That folder name is the world GUID (also reported by REST `worldguid`).

The operator seeds `GameUserSettings.ini` from `spec.dedicatedServerName` or learned `status.dedicatedServerName` (REST `worldguid`) via the same `seed-settings` init that writes `PalWorldSettings.ini`. A missing/wrong pin creates a **new** empty world and leaves the old folder on the PVC. Prefer setting `spec.dedicatedServerName` in GitOps after the first successful boot. After any reschedule, confirm REST `worldguid` still matches the intended `SaveGames/0/<guid>/` directory.

## Crossplay

Dedicated servers support Steam / Xbox / PS5 / Mac via `CrossplayPlatforms` (INI) or community image `CROSSPLAY_PLATFORMS` (`spec.crossplayPlatforms`).

Mods and crossplay interact:

- **Consoles cannot load PC client mods.** `spec.optionSettings.bAllowClientMod` is a PC join policy. If PC players join with client mods, console players still cannot use those mods — and a mismatch can look like a failed join.
- **Server content mods can lock consoles out.** A Linux `.pak` (or Windows Workshop package) that changes world content may make the dedicated server unjoinable from Xbox / PS5 even when those platforms are in `crossplayPlatforms`. Prefer a vanilla world if you host consoles.

Console clients usually still need community listing (`spec.community.enabled`) even when crossplay is on. See [CONNECT.md](CONNECT.md).

## Connecting from the game client

Player-facing join flow (Join Multiplayer Game, `connectionAddress:connectionPort`, join vs admin password, community browser): see [CONNECT.md](CONNECT.md).

Common failures (“incapable version”, empty world after restart, password prompt): [FAQ.md](FAQ.md).
