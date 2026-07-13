# Implementation tasks

Ordered work plan for code, build, and test. Mirrored as [GitHub Issues](https://github.com/DataKnifeAI/palworld-operator/issues); keep this file updated as tasks complete.

| ID | Issue |
|----|-------|
| C1–C6 | [#1](https://github.com/DataKnifeAI/palworld-operator/issues/1)–[#6](https://github.com/DataKnifeAI/palworld-operator/issues/6) |
| B1–B3 | [#7](https://github.com/DataKnifeAI/palworld-operator/issues/7)–[#9](https://github.com/DataKnifeAI/palworld-operator/issues/9) |
| T1–T4 | [#10](https://github.com/DataKnifeAI/palworld-operator/issues/10)–[#13](https://github.com/DataKnifeAI/palworld-operator/issues/13) |

Legend: `[ ]` pending · `[~]` in progress · `[x]` done

---

## Phase 0 — Bootstrap (this commit)

- [x] Create `DataKnifeAI/palworld-operator` repository
- [x] Apache-2.0 LICENSE, README, architecture docs
- [x] Directory layout matching windrose-operator / kubebuilder
- [x] Sample CR YAML + Palworld research notes
- [x] Prefer official Pocketpair image (`ghcr.io/pocketpairjp/palserver`); no separate DataKnifeAI server-image repo needed
- [x] Task breakdown (this file + GitHub Issues)

---

## Phase 1 — Code: API & project skeleton

### C1. Initialize Go module and kubebuilder layout
- Add `go.mod` (`github.com/DataKnifeAI/palworld-operator`, Go 1.25+)
- Dependencies aligned with windrose-operator: `controller-runtime`, `gateway-api`, `envoyproxy/gateway`, `k8s.io/*`
- `PROJECT` file, `hack/boilerplate.go.txt`, `cmd/main.go` manager entrypoint
- **Done when:** `go mod tidy` succeeds; empty manager binary builds

### C2. Define `PalworldServer` CRD (`api/v1alpha1`)
- Group: `palworld.dataknife.ai`, kind: `PalworldServer`, shortName: `ps`
- Spec fields (v1alpha1 minimum):
  - `serverImage`, `imagePullPolicy`, `imagePullSecrets`, `nodeSelector`
  - `gateway` (reuse Windrose `GatewayConfig` shape: address, className, gatewayName, envoyProxyName, externalTrafficPolicy)
  - `serverName`, `serverDescription`, `maxPlayers` (1–32)
  - `gamePort` (default 8211), `queryPort` (default 27015)
  - `rcon.enabled`, `rcon.port` (default 25575)
  - `restAPI.enabled`, `restAPI.port` (default 8212) — ClusterIP-only, not Gateway by default
  - `multithreading`, `community`, `publicIP`, `publicPort`, `timezone`, `updateOnBoot`
  - `storageSize` (default `50Gi`), `storageClassName`
  - `resources` (optional override); auto-select from `maxPlayers` when unset
  - Password references: `adminPasswordSecretRef`, `serverPasswordSecretRef` (Secret key refs)
- Status: `phase` (`Pending`/`Running`/`Failed`), `ready`, `connectionAddress`, `connectionPort`, `message`
- **Done when:** Types compile; `make generate` + `make manifests` produce CRD YAML

### C3. Reconciler: owned resources (core)
Port patterns from `windrose-operator/internal/controller`:
- Finalizer + deletion cleanup
- PVC for `/pal/Package/Pal/Saved` (official Pocketpair image mount)
- Deployment using **default** `ghcr.io/pocketpairjp/palserver:latest` (override via `spec.serverImage`)
- ConfigMap for `PalWorldSettings.ini` (+ CLI args for port/threading), matching Windrose ConfigMap pattern
- ClusterIP Service for game ports; Envoy backend Service
- Status updates from Deployment readiness
- Document community image (`thijsvanloef/...`) as optional alternate mount `/palworld` + env mapping
- **Done when:** Creating a CR without Gateway deps still creates Deployment/PVC/Services (Gateway can be feature-gated in C4)

### C4. Reconciler: Envoy Gateway exposure
- Mirror `envoy_gateway.go` from windrose-operator
- UDPRoute for `gamePort` + `queryPort`
- TCPRoute for RCON when enabled (optional; consider ClusterIP-only for RCON/REST)
- Naming: strip trailing `-server` for gateway/envoy names
- **Done when:** Sample CR on a cluster with Envoy Gateway gets external connectivity on 8211/UDP

### C5. Config & secrets UX
- Document required Secrets for admin/server passwords
- **Official image (default):** map CR fields → `PalWorldSettings.ini` ConfigMap + container command args (`-port=`, multithreading flags)
- **Community image (optional):** map CR fields → Docker env (`SERVER_NAME`, `PLAYERS`, `PORT`, `RCON_*`, etc.)
- Graceful shutdown: ensure RCON enabled + suitable `terminationGracePeriodSeconds`
- **Done when:** README documents secret creation; sample uses Secret refs; official INI path documented

### C6. Resource auto-selection
- Derive CPU/memory requests/limits from `maxPlayers` (Palworld is RAM-heavy; start conservative tiers e.g. 8/16/32 Gi)
- `spec.resources` fully overrides auto-selection (Windrose behavior)
- **Done when:** Unit tests cover tier tables

---

## Phase 2 — Build & packaging

### B1. Makefile parity with windrose-operator
- Targets: `generate`, `manifests`, `build`, `test`, `lint`, `ci`, `run`, `docker-build`, `docker-push`, `install`, `deploy`, `undeploy`
- Default `IMG=harbor.dataknife.net/library/palworld-operator:latest`
- **Done when:** `make ci` runs locally

### B2. Dockerfile & manager manifests
- Multi-stage Dockerfile (same pattern as Windrose)
- `config/manager`, `config/rbac`, `config/default` kustomize
- RBAC for Deployments, PVCs, Services, Secrets, Gateway API, EnvoyProxy
- **Done when:** `kubectl apply -k config/default` installs operator (CRD + Deployment)

### B3. CI pipelines
- GitHub Actions: lint + test on PR/push (copy windrose workflows)
- Optional follow-up: GitLab mirror + Harbor publish (see windrose `docs/GITLAB_MIRROR.md`)
- **Done when:** CI green on main

---

## Phase 3 — Test

### T1. Unit tests (controller)
- Fake client: creates expected owned objects
- Finalizer add/remove
- Validation failures (missing gateway.address, bad ports)
- Resource auto-selection tiers
- Env var mapping from spec
- **Done when:** `make test` with race detector passes; meaningful coverage on reconciler helpers

### T2. Envtest / integration (optional stretch)
- controller-runtime envtest for CRD install + reconcile loop
- **Done when:** Documented `make test-integration` or folded into `make test`

### T3. Manual cluster smoke test
- Deploy to `game-servers` (or a sandbox ns) on prd-apps / nprd-apps
- Connect a Palworld client to `.status.connectionAddress`:`gamePort`
- Verify PVC retains save across pod restart
- Verify RCON graceful stop does not corrupt saves
- **Done when:** Checklist in README “Quick start” verified once

### T4. Negative / ops tests
- Wrong StorageClass → Failed phase + clear message
- Missing password Secret → Pending with message
- Port conflict / Gateway address reuse documentation
- **Done when:** Status messages covered by unit tests or runbook notes

---

## Suggested issue labels

| Label | Use |
|-------|-----|
| `phase/code` | C1–C6 |
| `phase/build` | B1–B3 |
| `phase/test` | T1–T4 |
| `priority/P0` | Blocks runnable MVP (C1–C4, B1–B2, T1, T3) |
| `priority/P1` | Hardening (C5–C6, B3, T2, T4) |

## MVP definition

Ship when: CRD + operator image installable via kustomize, `PalworldServer` creates Deployment/PVC/Services + Envoy Gateway UDP exposure for 8211, unit tests green, one successful client join documented.
