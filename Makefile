# Image URL to use all building/pushing image targets
IMG ?= harbor.dataknife.net/library/palworld-operator:latest

VERSION ?= 0.1.0

# Resolve GOBIN lazily so compose-* targets work without a Go toolchain.
GOBIN ?= $(shell go env GOBIN 2>/dev/null)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH 2>/dev/null)/bin
endif
CONTROLLER_GEN ?= $(GOBIN)/controller-gen

.PHONY: all
all: generate manifests build

.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: build
build: generate
	go build -o bin/manager cmd/main.go

.PHONY: test
test: generate
	go test ./... -race -coverprofile cover.out

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix ./...

.PHONY: ci
ci: generate manifests vet lint test

.PHONY: run
run: manifests generate
	go run ./cmd/main.go

.PHONY: docker-build
docker-build:
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: install
install: manifests
	kubectl apply -f config/crd/bases

.PHONY: deploy
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kubectl apply -k config/default

.PHONY: undeploy
undeploy:
	kubectl delete -k config/default --ignore-not-found

.PHONY: controller-gen
controller-gen:
	test -s $(CONTROLLER_GEN) || GOBIN=$(GOBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.2

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

# Local / minimal PC — Docker Compose (no Kubernetes). See docs/LOCAL.md
COMPOSE_DIR ?= compose
COMPOSE = docker compose -f $(COMPOSE_DIR)/compose.yaml --project-directory $(COMPOSE_DIR)

.PHONY: compose-up
compose-up:
	@test -f $(COMPOSE_DIR)/.env || cp $(COMPOSE_DIR)/.env.example $(COMPOSE_DIR)/.env
	@$(COMPOSE_DIR)/scripts/seed-settings.sh
	$(COMPOSE) up -d
	@echo "Game: UDP $${GAME_PORT:-8211} on this host (see docs/LOCAL.md)"

.PHONY: compose-down
compose-down:
	$(COMPOSE) down

.PHONY: compose-logs
compose-logs:
	$(COMPOSE) logs -f

.PHONY: compose-ps
compose-ps:
	$(COMPOSE) ps

.PHONY: help
help:
	@echo "Operator: generate manifests build test lint ci install deploy"
	@echo "Local PC (Compose, no K8s): compose-up compose-down compose-logs compose-ps"
	@echo "See docs/LOCAL.md"
