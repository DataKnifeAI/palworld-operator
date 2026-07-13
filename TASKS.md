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
- [x] `go.mod` + deps aligned with windrose-operator; manager entrypoint builds

### C2. Define `PalworldServer` CRD (`api/v1alpha1`)
- [x] Group `palworld.dataknife.ai`, kind `PalworldServer`, shortName `ps`
- [x] `make generate` + `make manifests` produce CRD YAML

### C3. Reconciler: owned resources (core)
- [x] Finalizer, PVC, Deployment, ConfigMap INI, ClusterIP + envoy Services, status

### C4. Reconciler: Envoy Gateway exposure
- [x] EnvoyProxy + Gateway + UDPRoutes for game/query; REST optional TCPRoute

### C5. Config & secrets UX
- [x] Secret refs for admin/server passwords; official INI + community env paths

### C6. Resource auto-selection
- [x] Conservative tiers for ~8Gi nodes; `spec.resources` override; unit tests

---

## Phase 2 — Build & packaging

### B1. Makefile parity with windrose-operator
- [x] Targets match windrose; default Harbor IMG

### B2. Dockerfile & manager manifests
- [x] Multi-stage Dockerfile + kustomize install to `palworld-operator-system`

### B3. CI pipelines
- [x] GitLab Harbor publish updated; GitHub Actions present from bootstrap

---

## Phase 3 — Test

### T1. Unit tests (controller)
- [x] Helper tests (INI, resources, naming, community detection) with race

### T2. Envtest / integration (optional stretch)
- [ ] controller-runtime envtest for CRD install + reconcile loop

### T3. Manual cluster smoke test
- [~] Deploy on prd-apps `game-servers`; verify owned resources / status

### T4. Negative / ops tests
- [ ] Wrong StorageClass / missing Secret / VIP reuse runbook coverage

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
