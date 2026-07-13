# Controller package

Implement during TASKS.md C3–C6, porting patterns from:

`github.com/DataKnifeAI/windrose-operator/internal/controller`

Suggested files:

- `constants.go` — labels, finalizer, default ports/paths
  - `defaultServerImage = "ghcr.io/pocketpairjp/palserver:latest"`
  - `savedMountPath = "/pal/Package/Pal/Saved"` (official)
- `helpers.go` — naming, resource tiers, INI/CLI (and optional community env) mapping
- `envoy_gateway.go` — Gateway, EnvoyProxy, UDPRoute, TCPRoute
- `palworldserver_controller.go` — reconcile loop
- `*_test.go` — fake-client unit tests (T1)
