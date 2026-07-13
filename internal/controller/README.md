# Controller package

Reconciles `PalworldServer` into Deployment, PVC, ConfigMap, Secrets, Services, and Envoy Gateway resources.

| File | Role |
|------|------|
| `constants.go` | Labels, finalizer, default ports/paths/image |
| `helpers.go` | Naming, resource tiers, INI/CLI (+ community env) mapping |
| `envoy_gateway.go` | Gateway, EnvoyProxy, UDPRoute, TCPRoute |
| `palworldserver_controller.go` | Reconcile loop |
| `*_test.go` | Unit tests (helpers, secrets); envtest / fake-client loop still backlog (#10–#11) |

Default game image: `ghcr.io/pocketpairjp/palserver:latest`.
Saved mount (official): `/pal/Package/Pal/Saved`.
