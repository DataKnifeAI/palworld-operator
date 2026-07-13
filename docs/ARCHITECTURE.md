# Architecture

Kubebuilder-style Go operator that reconciles a `PalworldServer` custom resource into a dedicated Palworld world: Deployment, PVC, ConfigMap/Secret, ClusterIP services, and Envoy Gateway (UDPRoute/TCPRoute).

Patterns (Gateway naming, `{name}` / `{name}-envoy` backend split, PVC + config mounts, player-based resource tiers, status fields) align with [windrose-operator](https://github.com/DataKnifeAI/windrose-operator). Palworld-specific ports, image, and config paths differ as noted below.

## Data path

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

## Owned resources

Each `PalworldServer` reconciles:

| Kind | Purpose |
|------|---------|
| Deployment | Game server pod (official image by default) |
| PersistentVolumeClaim | World saves under `/pal/Package/Pal/Saved` |
| ConfigMap | `PalWorldSettings.ini` (official path) |
| Secret | Admin / server passwords injected into INI or env |
| Service (ClusterIP) | Backend for game / query / RCON / REST ports |
| Service (Envoy backend) | `{name}-envoy` ClusterIP |
| Gateway + EnvoyProxy | External VIP binding |
| UDPRoute | Game (`8211`) and Steam query (`27015`) |
| TCPRoute | RCON (`25575`) and optional REST API (`8212`) |

REST should default to **not** exposed via Gateway. Override gateway/proxy names with `spec.gateway.gatewayName` / `spec.gateway.envoyProxyName` when needed.

## Images

| Choice | Detail |
|--------|--------|
| **Default game image** | `ghcr.io/pocketpairjp/palserver:latest` — [official Pocketpair package](https://github.com/orgs/pocketpairjp/packages/container/package/palserver) |
| Operator image (CI) | `harbor.dataknife.net/library/palworld-operator` — see [GITLAB_MIRROR.md](GITLAB_MIRROR.md) |
| Harbor game mirror | Optional retag to `harbor.dataknife.net/library/palserver:...` |
| Community alternative | `thijsvanloef/palworld-server-docker` via `spec.serverImage` |

No DataKnifeAI custom game-image repo is required while Pocketpair publishes the official container. See [PALWORLD_SERVER.md](PALWORLD_SERVER.md) for mounts and config.

## Palworld vs Windrose (deltas)

| | palworld-operator | windrose-operator |
|--|-------------------|-------------------|
| Image | `ghcr.io/pocketpairjp/palserver` | `windroseserver/windroseserver` |
| CRD | `PalworldServer` | `WindroseServer` |
| Primary game port | `8211/UDP` | `7777/TCP+UDP` |
| Extra ports | Query `27015/UDP`, RCON `25575/TCP`, REST `8212/TCP` | None beyond game port |
| Config | ConfigMap → `PalWorldSettings.ini` (+ CLI args) | ConfigMap → `ServerDescription.json` |
| Save mount | `/pal/Package/Pal/Saved` (official) | `/home/ue_user/app/R5/Saved` |
| External access | Envoy Gateway | Envoy Gateway |

## Prerequisites

- Kubernetes 1.28+
- [Envoy Gateway](https://gateway.envoyproxy.io/) with GatewayClass `envoy`
- StorageClass suitable for game saves (prefer ReadWriteOnce)
- One dedicated external IP per server (`spec.gateway.address`)
- Cluster can pull from GHCR (or Harbor mirror + `imagePullSecrets`)

## Official references

- https://docs.palworldgame.com/getting-started/deploy-dedicated-server
- https://docs.palworldgame.com/settings-and-operation/configuration/
- https://github.com/pocketpairjp/palworld-dedicated-server-docker
- https://github.com/orgs/pocketpairjp/packages/container/package/palserver
