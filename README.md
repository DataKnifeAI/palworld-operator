# Palworld Operator

Kubernetes operator for [Palworld](https://www.palworldgame.com/) dedicated game servers.

This project follows the same architectural model as
[windrose-operator](https://github.com/DataKnifeAI/windrose-operator): a
kubebuilder-style Go operator that manages game servers declaratively via a
custom resource, exposes them with **Envoy Gateway** (TCPRoute/UDPRoute), and
persists world saves on a PVC.

> **Status:** Project bootstrap. The CRD API surface, architecture, and
> implementation plan are documented here and in [TASKS.md](TASKS.md). Full
> reconciler implementation is tracked as ordered GitHub Issues.

## Goals

- Manage Palworld dedicated servers via a `PalworldServer` custom resource
- Prefer the community Linux Docker image [`thijsvanloef/palworld-server-docker`](https://hub.docker.com/r/thijsvanloef/palworld-server-docker) (SteamCMD App ID `2394010` under the hood)
- Match DataKnife `prd-apps` / `game-servers` Envoy Gateway exposure patterns used by Windrose
- Keep secrets (admin/server passwords) out of plain CR fields where practical (Secrets + envFrom)

## Comparison with Windrose Operator

| | palworld-operator | windrose-operator |
|--|-------------------|-------------------|
| Image | `thijsvanloef/palworld-server-docker` (community; SteamCMD-based) | `windroseserver/windroseserver` (official) |
| CRD | `PalworldServer` | `WindroseServer` |
| Primary game port | `8211/UDP` | `7777/TCP+UDP` |
| Extra ports | Query `27015/UDP`, RCON `25575/TCP`, REST `8212/TCP` | None beyond game port |
| Config | Env vars → `PalWorldSettings.ini` | ConfigMap → `ServerDescription.json` |
| Save mount | `/palworld` (image convention) | `/home/ue_user/app/R5/Saved` |
| External access | Envoy Gateway (planned) | Envoy Gateway |

## Architecture (planned)

```
Clients → spec.gateway.address (Kube-VIP / MetalLB)
              ↓
      {base}-gateway  (GatewayClass: envoy)
              ↓
   UDPRoute (8211, 27015) + TCPRoute (25575, 8212 optional)
              ↓
      {name}-envoy  (ClusterIP)  →  {name} (ClusterIP)
              ↓
      Deployment  (palworld-server-docker)
              ↓
      PVC (/palworld)  +  Secret (passwords)  +  optional ConfigMap
```

Each `PalworldServer` will reconcile:

| Kind | Purpose |
|------|---------|
| Deployment | Game server pod |
| PersistentVolumeClaim | World saves + server files under `/palworld` |
| Secret / env | `ADMIN_PASSWORD`, `SERVER_PASSWORD`, optional Discord webhook |
| Service (ClusterIP) | Backend for game / query / RCON / REST ports |
| Service (Envoy backend) | `{name}-envoy` ClusterIP matching Windrose naming |
| Gateway + EnvoyProxy | External VIP binding |
| UDPRoute | Game (`8211`) and Steam query (`27015`) |
| TCPRoute | RCON (`25575`) and optional REST API (`8212`) |

## Palworld server requirements (operator-relevant)

Sources: [official deploy guide](https://docs.palworldgame.com/getting-started/deploy-dedicated-server),
[configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/),
[palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker).

### Ports

| Port | Protocol | Role |
|------|----------|------|
| 8211 | UDP | Primary game traffic (required) |
| 27015 | UDP | Steam query / community browser |
| 25575 | TCP | RCON (required for graceful stop/save in Docker image) |
| 8212 | TCP | REST API (default on in Docker image; do not publicly expose carelessly) |

### Persistence

- Save data lives under `Pal/Saved/SaveGames/` (Docker image mounts `/palworld`)
- Config: `Pal/Saved/Config/LinuxServer/PalWorldSettings.ini` (copied from `DefaultPalWorldSettings.ini`)
- Stop the server before mutating settings files; shutdown overwrites in-memory settings
- Native backups via `bIsUseBackupSaveData` / image `BACKUP_*` env vars; PVC should be sized generously (start at **50–100Gi**; worlds grow with bases/Pals)

### Resources (guidance)

| Players | Suggested memory | Notes |
|---------|------------------|-------|
| 1–8 | 8–16 Gi | Light private world |
| 8–16 | 16–24 Gi | Typical dedicated |
| 16–32 | 24–32+ Gi | Public / large bases; UE5 scales with structures |

CPU: prefer multi-core; Docker image `MULTITHREADING=true` helps up to ~4 threads.

### Key configuration knobs (CR / env mapping)

| Concern | Docker env / INI | CR field (planned) |
|---------|------------------|--------------------|
| Display name | `SERVER_NAME` | `spec.serverName` |
| Max players | `PLAYERS` / `ServerPlayerMaxNum` | `spec.maxPlayers` |
| Game port | `PORT` | `spec.gamePort` (default 8211) |
| Query port | `QUERY_PORT` | `spec.queryPort` (default 27015) |
| RCON | `RCON_ENABLED` / `RCON_PORT` | `spec.rcon` |
| REST API | `REST_API_ENABLED` / `REST_API_PORT` | `spec.restAPI` |
| Passwords | `SERVER_PASSWORD`, `ADMIN_PASSWORD` | `spec.serverPasswordSecretRef` / `adminPasswordSecretRef` |
| Community list | `COMMUNITY`, `PUBLIC_IP`, `PUBLIC_PORT` | `spec.community` + gateway address |
| Crossplay | `CROSSPLAY_PLATFORMS` / `CrossplayPlatforms` | `spec.crossplayPlatforms` |
| Update on boot | `UPDATE_ON_BOOT` | `spec.updateOnBoot` |

## Prerequisites (runtime)

- Kubernetes 1.28+
- [Envoy Gateway](https://gateway.envoyproxy.io/) with GatewayClass `envoy`
- StorageClass suitable for game saves (NFS/CSI OK; prefer ReadWriteOnce)
- One dedicated external IP per server (`spec.gateway.address`)

## Quick start (after implementation)

```shell
kubectl apply -k config/default
kubectl apply -f config/samples/palworld_v1alpha1_palworldserver.yaml
kubectl get palworldserver -n game-servers
```

Connect in-game using `.status.connectionAddress` and `.status.connectionPort` (default `8211`).

## Example (planned CR)

```yaml
apiVersion: palworld.dataknife.ai/v1alpha1
kind: PalworldServer
metadata:
  name: palworld-server
  namespace: game-servers
spec:
  serverImage: thijsvanloef/palworld-server-docker:latest
  gateway:
    address: 192.168.14.200
    className: envoy
  serverName: DataKnife Palworld
  maxPlayers: 16
  gamePort: 8211
  queryPort: 27015
  rcon:
    enabled: true
    port: 25575
  restAPI:
    enabled: true
    port: 8212
  multithreading: true
  updateOnBoot: true
  storageSize: 100Gi
  storageClassName: truenas-csi-nfs
  nodeSelector:
    kubernetes.io/os: linux
    kubernetes.io/arch: amd64
```

## Development

Requires Go 1.25+ (aligned with windrose-operator) and [golangci-lint](https://golangci-lint.run/).

```shell
make generate manifests   # CRD, RBAC, deepcopy (once API types land)
make test
make lint
make ci
make build
make docker-build IMG=harbor.dataknife.net/library/palworld-operator:latest
```

See [TASKS.md](TASKS.md) for the ordered code / build / test plan.

## Related projects

- [DataKnifeAI/windrose-operator](https://github.com/DataKnifeAI/windrose-operator) — architectural reference
- [Official Palworld dedicated server docs](https://docs.palworldgame.com/getting-started/deploy-dedicated-server)
- [Palworld configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/)
- [thijsvanloef/palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
