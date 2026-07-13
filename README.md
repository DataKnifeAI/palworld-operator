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
- Prefer the **official** Pocketpair Linux image [`ghcr.io/pocketpairjp/palserver`](https://github.com/pocketpairjp/palworld-dedicated-server-docker) (same “publisher image first” stance as Windrose)
- Match DataKnife `prd-apps` / `game-servers` Envoy Gateway exposure patterns used by Windrose
- Keep secrets (admin/server passwords) out of plain CR fields where practical (Secrets + mounts/env)

## Image strategy

| Choice | Detail |
|--------|--------|
| **Default** | `ghcr.io/pocketpairjp/palserver:latest` — [official Pocketpair package](https://github.com/orgs/pocketpairjp/packages/container/package/palserver) |
| Operator image (CI) | `harbor.dataknife.net/library/palworld-operator` — built on GitLab mirror ([docs/GITLAB_MIRROR.md](docs/GITLAB_MIRROR.md)) |
| Harbor game mirror | Optional: retag/copy to `harbor.dataknife.net/library/palserver:...` if the cluster should pull only from Harbor |
| Community alternative | `thijsvanloef/palworld-server-docker` — env-driven config; set via `spec.serverImage` |
| Custom DataKnifeAI game-image repo | **Not required** while Pocketpair publishes the official container (unlike Windrose’s Wine image in `windrose-server-k8s`) |

See [docs/PALWORLD_SERVER.md](docs/PALWORLD_SERVER.md) for mounts, INI vs env config, and ports.

## Comparison with Windrose Operator

| | palworld-operator | windrose-operator |
|--|-------------------|-------------------|
| Image | `ghcr.io/pocketpairjp/palserver` (official) | `windroseserver/windroseserver` (official) |
| CRD | `PalworldServer` | `WindroseServer` |
| Primary game port | `8211/UDP` | `7777/TCP+UDP` |
| Extra ports | Query `27015/UDP`, RCON `25575/TCP`, REST `8212/TCP` | None beyond game port |
| Config | ConfigMap → `PalWorldSettings.ini` (+ CLI args) | ConfigMap → `ServerDescription.json` |
| Save mount | `/pal/Package/Pal/Saved` (official) | `/home/ue_user/app/R5/Saved` |
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
      Deployment  (ghcr.io/pocketpairjp/palserver)
              ↓
      PVC (/pal/Package/Pal/Saved)  +  Secret + ConfigMap (INI)
```

Each `PalworldServer` will reconcile:

| Kind | Purpose |
|------|---------|
| Deployment | Game server pod (official image by default) |
| PersistentVolumeClaim | World saves under `/pal/Package/Pal/Saved` |
| ConfigMap | `PalWorldSettings.ini` (official path) |
| Secret | Admin / server passwords injected into INI or env |
| Service (ClusterIP) | Backend for game / query / RCON / REST ports |
| Service (Envoy backend) | `{name}-envoy` ClusterIP matching Windrose naming |
| Gateway + EnvoyProxy | External VIP binding |
| UDPRoute | Game (`8211`) and Steam query (`27015`) |
| TCPRoute | RCON (`25575`) and optional REST API (`8212`) |

## Palworld server requirements (operator-relevant)

Sources: [official deploy guide](https://docs.palworldgame.com/getting-started/deploy-dedicated-server),
[configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/),
[official Docker image](https://github.com/pocketpairjp/palworld-dedicated-server-docker).

### Ports

| Port | Protocol | Role |
|------|----------|------|
| 8211 | UDP | Primary game traffic (required) |
| 27015 | UDP | Steam query / community browser |
| 25575 | TCP | RCON (enable for graceful stop/save) |
| 8212 | TCP | REST API (do not publicly expose carelessly) |

### Persistence

- Official image: PVC → `/pal/Package/Pal/Saved` (covers `SaveGames/` + `Config/LinuxServer/`)
- Config: `PalWorldSettings.ini` (copied from `DefaultPalWorldSettings.ini` in the image)
- Stop the server before mutating settings files; shutdown overwrites in-memory settings
- PVC should be sized generously (start at **50–100Gi**; worlds grow with bases/Pals)

### Resources (guidance)

| Players | Suggested memory | Notes |
|---------|------------------|-------|
| 1–8 | 8–16 Gi | Light private world |
| 8–16 | 16–24 Gi | Typical dedicated |
| 16–32 | 24–32+ Gi | Public / large bases; UE5 scales with structures |

CPU: prefer multi-core; official CLI flags include `-UseMultithreadForDS` (community image: `MULTITHREADING=true`).

### Key configuration knobs (CR mapping)

| Concern | Official (INI / CLI) | Community env (optional image) | CR field (planned) |
|---------|----------------------|--------------------------------|--------------------|
| Display name | `ServerName` in INI | `SERVER_NAME` | `spec.serverName` |
| Max players | `ServerPlayerMaxNum` | `PLAYERS` | `spec.maxPlayers` |
| Game port | `-port=` CLI | `PORT` | `spec.gamePort` (default 8211) |
| Query port | INI / server args | `QUERY_PORT` | `spec.queryPort` (default 27015) |
| RCON | `RCONEnabled` / `RCONPort` | `RCON_*` | `spec.rcon` |
| REST API | INI | `REST_API_*` | `spec.restAPI` |
| Passwords | INI fields | `SERVER_PASSWORD`, `ADMIN_PASSWORD` | Secret refs |
| Community list | INI + public bind | `COMMUNITY`, `PUBLIC_*` | `spec.community` + gateway |
| Crossplay | `CrossplayPlatforms` | `CROSSPLAY_PLATFORMS` | `spec.crossplayPlatforms` |

## Prerequisites (runtime)

- Kubernetes 1.28+
- [Envoy Gateway](https://gateway.envoyproxy.io/) with GatewayClass `envoy`
- StorageClass suitable for game saves (NFS/CSI OK; prefer ReadWriteOnce)
- One dedicated external IP per server (`spec.gateway.address`)
- Cluster can pull from GHCR (or use a Harbor mirror + `imagePullSecrets` as needed)

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
  serverImage: ghcr.io/pocketpairjp/palserver:latest
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
  storageSize: 100Gi
  storageClassName: truenas-csi-nfs
  nodeSelector:
    kubernetes.io/os: linux
    kubernetes.io/arch: amd64
```

Optional community image:

```yaml
spec:
  serverImage: thijsvanloef/palworld-server-docker:latest
  # Community image uses /palworld + env-driven settings; reconciler must
  # detect or document the alternate mount/config path when overriding.
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
- [GitLab mirror](docs/GITLAB_MIRROR.md) — CI builds `harbor.dataknife.net/library/palworld-operator`
- [Official Palworld dedicated server docs](https://docs.palworldgame.com/getting-started/deploy-dedicated-server)
- [Palworld configuration parameters](https://docs.palworldgame.com/settings-and-operation/configuration/)
- [Official Docker image (Pocketpair)](https://github.com/pocketpairjp/palworld-dedicated-server-docker)
- [Community alternative: thijsvanloef/palworld-server-docker](https://github.com/thijsvanloef/palworld-server-docker)

## License

Apache License 2.0 — see [LICENSE](LICENSE).
