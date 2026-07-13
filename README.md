# Palworld Operator

Kubernetes operator for [Palworld](https://www.palworldgame.com/) dedicated servers.
Declare a `PalworldServer` CR and run a world on Kubernetes with Deployment, PVC, and Envoy Gateway exposure.

Congrats to Pocketpair on [Palworld 1.0](https://store.steampowered.com/news/app/1623730/view/686383649529010623) — see their [1.0 announcement](https://www.pocketpair.jp/en/game-news/palworld-1-0-july-10-cinematic-trailer-revealed/) and the [Steam store page](https://store.steampowered.com/app/1623730/Palworld/).

**Landing page:** [dataknifeai.github.io/palworld-operator](https://dataknifeai.github.io/palworld-operator/)

**Default game image:** [`ghcr.io/pocketpairjp/palserver`](https://github.com/pocketpairjp/palworld-dedicated-server-docker) (official Pocketpair).
**Operator image:** `harbor.dataknife.net/library/palworld-operator`

## Local / minimal PC (no Kubernetes)

For a gaming PC or laptop — Docker Compose only, same official image the operator runs:

```shell
cp compose/.env.example compose/.env   # set SERVER_PASSWORD / ADMIN_PASSWORD
make compose-up
# Join Multiplayer Game → 127.0.0.1:8211
make compose-down
```

Full guide (RAM, ports, LAN, passwords): [docs/LOCAL.md](docs/LOCAL.md).

## Quick start (Kubernetes)

Prerequisites: Kubernetes 1.28+, [Envoy Gateway](https://gateway.envoyproxy.io/) (`GatewayClass` `envoy`), a StorageClass for saves, and one dedicated external IP per server (`spec.gateway.address`).

```shell
# Optional: bring-your-own credentials (skip if using spec.generateSecrets: true)
kubectl -n game-servers create secret generic palworld-server-secrets \
  --from-literal=admin-password='CHANGE_ME_ADMIN' \
  --from-literal=server-password='CHANGE_ME_JOIN'

kubectl apply -k config/default
kubectl apply -f config/samples/palworld_v1alpha1_palworldserver.yaml
kubectl get palworldserver -n game-servers
```

Connect using `.status.connectionAddress` and `.status.connectionPort` (default `8211` UDP).
**Known limitation:** the operator does not manage `DedicatedServerName` in `GameUserSettings.ini` — without a pin, a restart can load a new empty world (see [docs/PALWORLD_SERVER.md](docs/PALWORLD_SERVER.md#world-selection-across-restarts)).
Read join/admin passwords from the credentials Secret (see [docs/CONNECT.md](docs/CONNECT.md)):

```shell
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.server-password}' | base64 -d; echo
```

In-game join steps (Join Multiplayer Game, passwords, community vs direct): [docs/CONNECT.md](docs/CONNECT.md).
Adjust the [sample CR](config/samples/palworld_v1alpha1_palworldserver.yaml) for your VIP, StorageClass, and resources — including **bring-your-own** Secret refs or **`generateSecrets: true`**.

```shell
kubectl delete palworldserver palworld-server -n game-servers
```

## Docs

| Doc | Contents |
|-----|----------|
| [docs/LOCAL.md](docs/LOCAL.md) | **Docker Compose** local / minimal PC — no Kubernetes |
| [docs/CONNECT.md](docs/CONNECT.md) | Join from inside Palworld (direct connect, passwords, community, crossplay) |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Reconciled resources, Gateway layout, Palworld vs Windrose deltas |
| [docs/PALWORLD_SERVER.md](docs/PALWORLD_SERVER.md) | Ports, mounts, INI/env config, resources, **game updates / Steam** |
| [docs/GITLAB_MIRROR.md](docs/GITLAB_MIRROR.md) | GitHub quality gates + GitLab Harbor publish |
| [CHANGELOG.md](CHANGELOG.md) | Release notes / known gaps |

## Development

Go 1.25+ and [golangci-lint](https://golangci-lint.run/). Common targets: `make generate manifests`, `make test`, `make lint`, `make ci`, `make build`.
CI and Harbor publish details: [docs/GITLAB_MIRROR.md](docs/GITLAB_MIRROR.md). Remaining work: [TASKS.md](TASKS.md).

## License

Apache License 2.0 — see [LICENSE](LICENSE).
Maintained by [DataKnifeAI](https://github.com/DataKnifeAI).
