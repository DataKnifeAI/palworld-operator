# Image URL to use all building/pushing image targets
IMG ?= harbor.dataknife.net/library/palworld-operator:latest

VERSION ?= 0.1.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
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

.PHONY: help
help:
	@echo "Targets match windrose-operator."
