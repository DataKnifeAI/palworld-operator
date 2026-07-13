# Local / minimal PC (Docker Compose)

Run a Palworld dedicated server on a **gaming PC or laptop** with **no Kubernetes**.
This is the simple path for smoke-testing the official image, hosting a tiny private
world for friends on the LAN, or developing against the same container the operator uses.

| Path | When to use |
|------|-------------|
| **Docker Compose (this guide)** | Local / PC / “just run the game server” — **no cluster** |
| **Kubernetes operator** | Production / shared cluster — Envoy Gateway, PVC, CRDs — see [README](../README.md) |

Official image: [`ghcr.io/pocketpairjp/palserver`](https://github.com/pocketpairjp/palworld-dedicated-server-docker).
Upstream sample lives under their `compose/` directory; this repo’s `compose/` is a
**minimal-PC** variant (resource caps, localhost-bound REST/RCON, `.env` seed for passwords).

## Prerequisites

- Docker Engine + Compose v2 (`docker compose version`)
- Prefer **Linux** hosts (Pocketpair warns that Docker Desktop on Windows/macOS has slow disk I/O for saves)
- Roughly **8 GiB free RAM** for a comfortable 2–4 player world (compose default `MEM_LIMIT=6g`; raise if OOM-killed)
- UDP **8211** reachable from clients (localhost or LAN)

## ~5 commands

From the repo root:

```shell
cp compose/.env.example compose/.env
# Edit compose/.env — at least SERVER_PASSWORD and ADMIN_PASSWORD

make compose-up
make compose-logs          # optional: watch startup
# In Palworld → Join Multiplayer Game → 127.0.0.1:8211 (or LAN IP)
make compose-down
```

Equivalent without Make:

```shell
cp compose/.env.example compose/.env
./compose/scripts/seed-settings.sh
docker compose -f compose/compose.yaml --project-directory compose up -d
docker compose -f compose/compose.yaml --project-directory compose down
```

## What gets created

| Item | Location / detail |
|------|-------------------|
| Container | `palworld-local` |
| Saves + INI | `compose/Saved/` → `/pal/Package/Pal/Saved` |
| Settings seed | `compose/Saved/Config/LinuxServer/PalWorldSettings.ini` (first start only) |
| Game port | Host `${GAME_PORT:-8211}` → container `8211/udp` |
| Query port | `${QUERY_PORT:-27015}/udp` (community browser; optional) |
| REST / RCON | Bound to **127.0.0.1** only (`8212`, `25575`) |

`make compose-up` copies `.env` from the example if missing, runs the seed script, then
`docker compose up -d`. The seed script **never overwrites** an existing
`PalWorldSettings.ini` — edit that file (or delete it after `compose-down`) to change
name/passwords/max players.

## Passwords

Set in `compose/.env` before first start:

| Variable | Purpose | Share? |
|----------|---------|--------|
| `SERVER_PASSWORD` | In-game join / `ServerPassword` | Trusted players only |
| `ADMIN_PASSWORD` | Admin / RCON | **No** |

Read back after seed:

```shell
grep -E 'ServerPassword|AdminPassword' compose/Saved/Config/LinuxServer/PalWorldSettings.ini
```

Do not commit `compose/.env` or live `Saved/` data (both are gitignored).

## Connect from the game client

Same flow as the operator path — see [CONNECT.md](CONNECT.md):

1. Launch **Palworld** → **Join Multiplayer Game**
2. Direct-connect:
   - Same machine: `127.0.0.1:8211` (or your `GAME_PORT`)
   - Another PC on LAN: `<host-LAN-IP>:8211`
3. Enable **Enter password** and use `SERVER_PASSWORD`

For LAN friends, allow UDP **8211** through the host firewall. Do **not** port-forward
REST (`8212`) or RCON (`25575`) — compose already binds them to loopback.

## Resource expectations

| Knob | Default | Notes |
|------|---------|-------|
| `MAX_PLAYERS` | `4` | Keep low on a shared gaming PC |
| `MEM_LIMIT` | `6g` | Raise to `8g`–`12g` if the container is killed |
| `CPU_LIMIT` | `4.0` | Multithreading CLI flags are enabled |
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

## Optional: Kubernetes local clusters

Compose is the supported local path. If you need to exercise the **operator** itself,
use kind/k3d/minikube separately (`make install` / `make deploy` + a sample CR). That
path still expects Gateway plumbing for production-style exposure and is out of scope
for this minimal PC guide.

## Related

- [CONNECT.md](CONNECT.md) — in-game join steps
- [PALWORLD_SERVER.md](PALWORLD_SERVER.md) — ports, INI, official vs community images
- [ARCHITECTURE.md](ARCHITECTURE.md) — operator / Envoy layout (cluster)
- Upstream compose: [pocketpairjp/palworld-dedicated-server-docker](https://github.com/pocketpairjp/palworld-dedicated-server-docker)
