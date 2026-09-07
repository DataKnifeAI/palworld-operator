# Local / minimal PC (Docker Compose)

Run a Palworld dedicated server on a **gaming PC or laptop** with **no Kubernetes**.
This is the simple path for smoke-testing the official image, hosting a tiny private
world for friends on the LAN, or developing against the same container the operator uses
plus a local **Server Manager** sidecar.

| Path | When to use |
|------|-------------|
| **Docker Compose (this guide)** | Local / PC / “just run the game server” — **no cluster** |
| **Kubernetes operator** | Production / shared cluster — Envoy Gateway, PVC, CRDs — see [README](../README.md) |

Troubleshooting (version mismatch, passwords, world pin): [FAQ.md](FAQ.md).

Official image: [`ghcr.io/pocketpairjp/palserver`](https://github.com/pocketpairjp/palworld-dedicated-server-docker).
Upstream sample lives under their `compose/` directory; this repo’s `compose/` is a
**minimal-PC** variant (resource caps, localhost-bound REST / Server Manager / legacy RCON,
`.env` seed for passwords, community `.pak` overlays, Server Manager sidecar).

Compose does **not** run SteamCMD. Image updates are new Pocketpair tags (see below).

## Prerequisites

- Docker Engine + Compose v2 (`docker compose version`)
- Prefer **Linux** hosts (Pocketpair warns that Docker Desktop on Windows/macOS has slow disk I/O for saves)
- Roughly **8 GiB free RAM** for a comfortable 2–4 player world (compose default `MEM_LIMIT=6g`; raise if OOM-killed)
- UDP **8211** reachable from clients (localhost or LAN)
- Disk for the official image (tens of GiB). If you already pulled `ghcr.io/pocketpairjp/palserver`, Compose will reuse it.

## ~5 commands

From the repo root:

```shell
cp compose/.env.example compose/.env
# Edit compose/.env — at least SERVER_PASSWORD and ADMIN_PASSWORD

make compose-up
make compose-logs          # optional: watch startup
# In Palworld → Join Multiplayer Game → 127.0.0.1:8211 (or LAN IP)
# Server Manager: http://127.0.0.1:8088  (basic auth admin + ADMIN_PASSWORD)
make compose-down
```

Equivalent without Make:

```shell
cp compose/.env.example compose/.env
./compose/scripts/seed-settings.sh
docker compose -f compose/compose.yaml --project-directory compose up -d --build
docker compose -f compose/compose.yaml --project-directory compose down
```

`make compose-up` copies `.env` from the example if missing, seeds settings + empty mods
dirs, then `docker compose up -d --build` (builds the sidecar from this repo’s Dockerfile
when Harbor is unavailable or you want local `/server-manager`).

First PalServer start can take **several minutes** (image pull + world init).

## What gets created

| Item | Location / detail |
|------|-------------------|
| Game container | `palworld-local` (`palworld-server` service) |
| Server Manager | `palworld-local-manager` — operator image, entrypoint `/server-manager` |
| Saves + INI | `compose/Saved/` → `/pal/Package/Pal/Saved` (sidecar `/saves`) |
| Settings seed | `compose/Saved/Config/LinuxServer/PalWorldSettings.ini` (first start only) |
| Mods root | `compose/Mods/` → `/pal/Package/Mods` (sidecar `/mods`) |
| Pak overlays | `compose/Mods/paks/~WorkshopMods` → `Paks/~WorkshopMods` |
| | `compose/Mods/paks/LogicMods` → `Paks/LogicMods` |
| Game port | Host `${GAME_PORT:-8211}` → container `8211/udp` |
| Query port | `${QUERY_PORT:-27015}/udp` (community browser; optional) |
| REST / RCON / UI | Bound to **127.0.0.1** only (`8212` REST, `25575` legacy RCON, `8088` Server Manager) |

The seed script **never overwrites** an existing `PalWorldSettings.ini` — edit that file
(or delete it after `compose-down`) to change name/passwords/max players. It **does**
create empty mods overlay dirs so Docker does not invent root-owned mount points.

Overlays are **subfolders only**. Do not bind-mount the whole `Paks/` directory — that
would hide `Pal-LinuxServer.pak` and the server would not start.

## Server Manager (local)

Open [http://127.0.0.1:8088/](http://127.0.0.1:8088/) and sign in with basic auth:
username `admin` (or `SERVER_MANAGER_USER`) and `ADMIN_PASSWORD` from `.env`.

The UI ships the same tabs as the cluster sidecar (Overview, Controls, Updates, Saves,
Mods, Settings). **REST-first:** Overview / announce / save / kick / ban talk to Palworld
REST (`SERVER_MANAGER_REST_BASE`, default `http://palworld-server:8212`). Do **not** treat
legacy RCON as the admin path (Pocketpair deprecated it).

| Works locally | Needs Kubernetes (limited / 4xx or 503 here) |
|---------------|----------------------------------------------|
| UI `/` (200), Mods list/upload/delete, Saves zip | **Updates** tab (image pin / Check now / Force update) |
| REST proxy when PalServer is up | **Settings** apply (`spec.optionSettings` / profile) |
| Drop `.pak` files on disk or via Mods tab | **Credential rotate** (Secret keys) |
| | **Recreate restart** (`DeploymentRestarter`) |

Local restart/settings-apply/rotate return **4xx/503** (`restart is not configured`,
`update API is not configured`, `settings API is not configured`) instead of crashing.
To restart the game container: `docker compose … restart palworld-server` (or
`make compose-down` / `make compose-up`). To change settings: edit
`PalWorldSettings.ini` (or delete it and re-seed from `.env`).

Sidecar image default is `palworld-operator:compose` (built from this repo’s Dockerfile,
`/server-manager`). Set `SERVER_MANAGER_IMAGE=harbor.dataknife.net/library/palworld-operator:latest`
to use Harbor instead. Memory default is **128Mi** — the cluster sidecar’s 1Gi cap is not
required on a laptop.

## Mods `.pak` drop path

**Palworld Server does load community pak files.** Only `.pak` is supported. Official
[Pocketpair mods](https://docs.palworldgame.com/settings-and-operation/mod/) (Workshop /
`PalModSettings.ini` / `-workshopdir`) plus UE4SS, Lua, and Win64 DLLs are not supported.

Both the **client and the server** must have the mod installed. Mods typically align to
PC players; consoles cannot load PC client mods.

Copy files here (or upload via Server Manager → Mods):

```text
compose/Mods/paks/~WorkshopMods/YourMod.pak
compose/Mods/paks/LogicMods/YourMod.pak
```

Default UI path is `paks/~WorkshopMods`. Official OptionSettings reference:
[https://docs.palworldgame.com/settings-and-operation/configuration/](https://docs.palworldgame.com/settings-and-operation/configuration/).

## Passwords

Set in `compose/.env` before first start:

| Variable | Purpose | Share? |
|----------|---------|--------|
| `SERVER_PASSWORD` | In-game join / `ServerPassword` | Trusted players only |
| `ADMIN_PASSWORD` | Admin / REST / Server Manager basic auth (legacy RCON same password) | **No** |

Read back after seed:

```shell
grep -E 'ServerPassword|AdminPassword' compose/Saved/Config/LinuxServer/PalWorldSettings.ini
```

Do not commit `compose/.env` or live `Saved/` / uploaded `.pak` data (gitignored).

## World selection across restarts

Same as the operator path: Palworld picks the world from `GameUserSettings.ini` →
`DedicatedServerName`. Compose does not manage that file — after a recreate, confirm the
save under `compose/Saved/SaveGames/0/` still matches REST `worldguid`. Details:
[PALWORLD_SERVER.md](PALWORLD_SERVER.md#world-selection-across-restarts).

## Connect from the game client

Same flow as the operator path — see [CONNECT.md](CONNECT.md):

1. Launch **Palworld** → **Join Multiplayer Game**
2. Direct-connect:
   - Same machine: `127.0.0.1:8211` (or your `GAME_PORT`)
   - Another PC on LAN: `<host-LAN-IP>:8211`
3. Enable **Enter password** and use `SERVER_PASSWORD`

For LAN friends, allow UDP **8211** through the host firewall. Do **not** port-forward
REST (`8212`), Server Manager (`8088`), or legacy RCON (`25575`) — compose already binds
them to loopback.

## Resource expectations

| Knob | Default | Notes |
|------|---------|-------|
| `MAX_PLAYERS` | `4` | Keep low on a shared gaming PC |
| `MEM_LIMIT` | `6g` | Raise to `8g`–`12g` if the game container is killed |
| `CPU_LIMIT` | `4.0` | Multithreading CLI flags are enabled |
| `SIDECAR_MEM_LIMIT` | `128m` | Server Manager only; do not copy the cluster 1Gi request |
| Disk | grows under `compose/Saved/` | Worlds grow with bases/Pals; tens of GiB over time |

This is **not** a 16 GiB “must have” floor for a tiny private world, but UE5 will use
several GiB even idle. If the host is also running the Palworld **client**, leave headroom.

## Updating the game image

1. Back up `compose/Saved/`
2. `make compose-down`
3. Bump `PALSERVER_IMAGE` in `.env` to a Pocketpair version tag (or pull `latest`)
4. `make compose-up`

Official image updates are **new image tags**, not in-container SteamCMD — see
[PALWORLD_SERVER.md](PALWORLD_SERVER.md#updating-the-game-server-steam--patches).
The Compose **Updates** tab cannot pin/roll the image (that writes a PalworldServer CR).

## Optional: Kubernetes local clusters

Compose is the supported local path. If you need to exercise the **operator** itself
(Updates, Settings CR apply, credential rotate, Recreate), use kind/k3d/minikube
separately (`make install` / `make deploy` + a sample CR). That path still expects
Gateway plumbing for production-style exposure and is out of scope for this minimal PC
guide.

## Related

- [CONNECT.md](CONNECT.md) — in-game join steps
- [PALWORLD_SERVER.md](PALWORLD_SERVER.md) — ports, INI, official vs community images, mods
- [ARCHITECTURE.md](ARCHITECTURE.md) — operator / Envoy layout (cluster)
- Official settings: [configuration](https://docs.palworldgame.com/settings-and-operation/configuration/)
- Upstream compose: [pocketpairjp/palworld-dedicated-server-docker](https://github.com/pocketpairjp/palworld-dedicated-server-docker)
