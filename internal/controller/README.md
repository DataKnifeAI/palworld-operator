# Controller package

Reconciles `PalworldServer` into Deployment, PVC, ConfigMap, Secrets, Services, and Envoy Gateway resources.

| File | Role |
|------|------|
| `constants.go` | Labels, finalizer, default ports/paths/image |
| `helpers.go` | Naming, resource tiers, INI/CLI (+ community env) mapping |
| `envoy_gateway.go` | Gateway, EnvoyProxy, UDPRoute, TCPRoute |
| `palworldserver_controller.go` | Reconcile loop |
| `update.go` / `registry.go` / `version.go` | Opt-in GHCR tag discovery and image pin |
| `notify.go` / `schedule.go` | Staged REST announce before apply |
| `rest.go` | Palworld REST client (`info`, `announce`, `worldguid`) |
| `*_test.go` | Unit tests (helpers, secrets, version, schedule, notify); envtest / fake-client loop still backlog (#10–#11) |

Fallback game image: `ghcr.io/pocketpairjp/palserver:latest` (samples pin a version tag).
Saved mount (official): `/pal/Package/Pal/Saved`. Optional mods PVC (`spec.mods.enabled`): `/pal/Package/Mods` plus `Paks/~WorkshopMods` and `Paks/LogicMods` subpath overlays.
