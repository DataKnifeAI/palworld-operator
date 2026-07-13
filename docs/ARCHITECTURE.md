# Architecture (Windrose-aligned)

This operator intentionally mirrors [DataKnifeAI/windrose-operator](https://github.com/DataKnifeAI/windrose-operator).

## Patterns to reuse

1. **Single CR per server** — `PalworldServer` owns all child resources; one external IP per CR.
2. **Envoy Gateway exposure** — No LoadBalancer on the game pod. Clients hit Gateway address → UDPRoute/TCPRoute → `{name}-envoy` ClusterIP → `{name}` ClusterIP → Deployment.
3. **Naming** — Strip trailing `-server` from CR name for Gateway/EnvoyProxy defaults (`palworld-server` → `palworld-gateway`, `game-palworld-kubevip`).
4. **PVC for world data** — Palworld: mount `/palworld`. Windrose: `/home/ue_user/app/R5/Saved`.
5. **Status surface** — `phase`, `ready`, `connectionAddress`, `connectionPort`, `message` for in-game connect UX.
6. **Resource auto-selection** — Derive requests/limits from player cap; explicit `spec.resources` wins.
7. **Kubebuilder layout** — `api/`, `cmd/`, `internal/controller/`, `config/{crd,rbac,manager,default,samples}`, `Makefile`, Harbor image under `harbor.dataknife.net/library/`.
8. **Controller implementation split** — helpers, constants, gateway builders, reconciler + fake-client tests (see windrose `internal/controller/`).

## Palworld-specific deltas

| Topic | Windrose | Palworld |
|-------|----------|----------|
| Config delivery | ConfigMap file mount (`ServerDescription.json`) | Container env vars (and optional INI later) |
| Secrets | Password in CR today | Prefer Secret refs from day one |
| Ports | Single 7777 TCP+UDP | Multi-port UDP/TCP matrix |
| Image trust | Official publisher image | Community image + pin digests |
| REST/RCON | N/A | First-class optional ports; REST not on Gateway by default |

## Managed resource graph

```
PalworldServer
├── Secret (password refs / optional generated)
├── PersistentVolumeClaim  ({name}-files)
├── Deployment             ({name})
├── Service                ({name})           # ClusterIP — pod selector
├── Service                ({name}-envoy)     # ClusterIP — Envoy backend
├── Gateway                ({base}-gateway)
├── EnvoyProxy             (game-{base}-kubevip)
├── UDPRoute               (game + query)
└── TCPRoute               (RCON optional)
```

## Out of scope for MVP

- Cross-namespace Gateway references
- Multi-world / multi-instance per CR
- Xbox-only dedicated mode as default
- Building a first-party SteamCMD image (use community image first)
- Full `PalWorldSettings.ini` gameplay balance CR schema (rates, PvP flags) — defer to follow-up API fields or raw INI ConfigMap
