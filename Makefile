SHELL := /usr/bin/env bash
.SHELLFLAGS := -o pipefail -ec
.DEFAULT_GOAL := help

PROJECT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
LOCALBIN ?= $(PROJECT_DIR)/bin

IMG ?= ghcr.io/winrarr/infisical-entity-operator:dev
CONTAINER_TOOL ?= docker
KUBECTL ?= kubectl
KIND ?= $(LOCALBIN)/kind
HELM ?= helm
PROJECT_NAME ?= infisical-entity-operator
KIND_CLUSTER ?= infisical-entity-operator
KIND_NODE_IMAGE ?= kindest/node:v1.37.0
OPERATOR_NAMESPACE ?= infisical-entity-operator-system
INFISICAL_NAMESPACE ?= infisical
INFISICAL_RELEASE ?= infisical
INFISICAL_CHART_VERSION ?= 1.10.0
INFISICAL_IMAGE_TAG ?= v0.165.8
CILIUM_VERSION ?= 1.20.1
GO_TOOLCHAIN ?= go1.27.1
KIND_VERSION ?= v0.33.0

KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.22.0
GOLANGCI_LINT_VERSION ?= v2.13.2

GO := GOTOOLCHAIN=$(GO_TOOLCHAIN) go
GOFMT := $(shell GOTOOLCHAIN=$(GO_TOOLCHAIN) go env GOROOT)/bin/gofmt

.PHONY: all
all: check build ## Run the default verification and build workflow.

##@ Help

.PHONY: help
help: ## Display available targets.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: format
format: ## Format Go sources.
	$(GO) fmt ./...

.PHONY: format-check
format-check: ## Fail when Go sources are not gofmt-clean.
	@files="$$($(GOFMT) -l $$(find api cmd internal test -name '*.go' -type f))"; \
	if [ -n "$$files" ]; then echo "Go sources need formatting:" >&2; echo "$$files" >&2; exit 1; fi

.PHONY: generate
generate: controller-gen ## Generate Go deepcopy code.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen ## Generate CRDs and RBAC from API/controller markers.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(MAKE) sync-chart-generated

.PHONY: sync-chart-generated
sync-chart-generated: ## Copy generated CRDs and RBAC into the Helm chart.
	@mkdir -p charts/infisical-entity-operator/crds
	@rm -f charts/infisical-entity-operator/crds/*.yaml
	@cp config/crd/bases/*.yaml charts/infisical-entity-operator/crds/
	@cp config/rbac/role.yaml charts/infisical-entity-operator/templates/clusterrole.yaml

.PHONY: verify-generated
verify-generated: manifests generate ## Verify committed generated artifacts are current.
	@git diff --exit-code -- api/infisical/v1alpha1/zz_generated.deepcopy.go config/crd/bases config/rbac/role.yaml charts/infisical-entity-operator/crds charts/infisical-entity-operator/templates/clusterrole.yaml

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: manifests generate format-check vet ## Run unit and package tests.
	$(GO) test ./... -coverprofile=cover.out

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint with safe automatic fixes.
	"$(GOLANGCI_LINT)" run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Validate the golangci-lint configuration.
	"$(GOLANGCI_LINT)" config verify

.PHONY: helm-lint
helm-lint: ## Lint the operator Helm chart.
	$(HELM) lint charts/infisical-entity-operator

.PHONY: check
check: manifests generate format-check vet test lint-config lint helm-lint ## Run the complete local verification suite.

##@ Build

.PHONY: build
build: manifests generate format-check vet ## Build the controller binary.
	@mkdir -p bin
	$(GO) build -trimpath -ldflags="-s -w" -o bin/manager ./cmd

.PHONY: run
run: manifests generate ## Run the controller against the current kubeconfig context.
	$(GO) run ./cmd

.PHONY: docker-build
docker-build: ## Build the controller image.
	$(CONTAINER_TOOL) build --tag $(IMG) .

.PHONY: docker-push
docker-push: ## Push the controller image.
	$(CONTAINER_TOOL) push $(IMG)

.PHONY: docker-buildx
docker-buildx: ## Build and push a multi-platform controller image.
	$(CONTAINER_TOOL) buildx build --platform=$(PLATFORMS) --tag $(IMG) --push .

PLATFORMS ?= linux/amd64,linux/arm64

.PHONY: build-installer
build-installer: manifests generate kustomize ## Build a standalone Kustomize installation bundle.
	@mkdir -p dist
	"$(KUSTOMIZE)" build config/default > dist/install.yaml

##@ Kubernetes deployment

.PHONY: install
install: manifests kustomize ## Install the CRDs in the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | "$(KUBECTL)" apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Remove the CRDs from the current Kubernetes context.
	"$(KUSTOMIZE)" build config/crd | "$(KUBECTL)" delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests generate helm-lint ## Install or upgrade the operator Helm chart.
	"$(HELM)" upgrade --install $(PROJECT_NAME) charts/infisical-entity-operator \
		--namespace $(OPERATOR_NAMESPACE) --create-namespace \
		--set image.repository=$$(echo $(IMG) | cut -d: -f1) \
		--set image.tag=$$(echo $(IMG) | cut -d: -f2-) \
		--wait --timeout 5m

.PHONY: undeploy
undeploy: ## Uninstall the operator Helm release.
	"$(HELM)" uninstall $(PROJECT_NAME) --namespace $(OPERATOR_NAMESPACE) --ignore-not-found

##@ Local Kind environment

.PHONY: kind-up
kind-up: kind-create kind-install-cilium kind-install-infisical ## Create a local Kind cluster with Cilium and Infisical.

.PHONY: kind-create
kind-create: kind ## Create the isolated Kind cluster if it does not exist.
	@if ! "$(KIND)" get clusters | grep -Fxq "$(KIND_CLUSTER)"; then \
		"$(KIND)" create cluster --name "$(KIND_CLUSTER)" --image "$(KIND_NODE_IMAGE)" --config hack/kind-configuration-cilium.yaml; \
	else \
		echo "Kind cluster $(KIND_CLUSTER) already exists"; \
	fi

.PHONY: kind-install-cilium
kind-install-cilium: ## Install the pinned Cilium release into Kind.
	@"$(HELM)" upgrade --install cilium oci://quay.io/cilium/charts/cilium \
		--version "$(CILIUM_VERSION)" --namespace kube-system \
		--set kubeProxyReplacement=true \
		--set k8sServiceHost=$(KIND_CLUSTER)-control-plane \
		--set k8sServicePort=6443 \
		--set operator.replicas=1 \
		--set hubble.enabled=true \
		--set hubble.relay.enabled=true \
		--set hubble.ui.enabled=false \
		--wait --timeout 10m

.PHONY: kind-install-infisical
kind-install-infisical: ## Install the pinned Infisical release with an ephemeral bootstrap token.
	@"$(KUBECTL)" create namespace "$(INFISICAL_NAMESPACE)" --dry-run=client -o yaml | "$(KUBECTL)" apply -f -
	@if ! "$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" get secret infisical-secrets >/dev/null 2>&1; then \
		auth_secret="$$(openssl rand -base64 32)"; encryption_key="$$(openssl rand -hex 16)"; \
		"$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" create secret generic infisical-secrets \
			--from-literal=AUTH_SECRET="$$auth_secret" \
			--from-literal=ENCRYPTION_KEY="$$encryption_key" \
			--from-literal=SITE_URL=http://localhost \
			--dry-run=client -o yaml | "$(KUBECTL)" apply -f -; \
	fi
	@if ! "$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" get secret infisical-bootstrap-credentials >/dev/null 2>&1; then \
		bootstrap_password="$$(openssl rand -hex 24)"; \
		"$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" create secret generic infisical-bootstrap-credentials \
			--from-literal=INFISICAL_ADMIN_EMAIL=operator-dev@example.invalid \
			--from-literal=INFISICAL_ADMIN_PASSWORD="$$bootstrap_password" \
			--dry-run=client -o yaml | "$(KUBECTL)" apply -f -; \
	fi
	@"$(HELM)" repo add infisical-helm-charts https://dl.cloudsmith.io/public/infisical/helm-charts/helm/charts/ --force-update >/dev/null
	@"$(HELM)" repo update infisical-helm-charts >/dev/null
	@if ! "$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" get secret infisical-bootstrap-token >/dev/null 2>&1; then \
		"$(HELM)" uninstall "$(INFISICAL_RELEASE)" --namespace "$(INFISICAL_NAMESPACE)" --ignore-not-found >/dev/null 2>&1 || true; \
	fi
	@"$(HELM)" upgrade --install "$(INFISICAL_RELEASE)" infisical-helm-charts/infisical-standalone \
		--version "$(INFISICAL_CHART_VERSION)" --namespace "$(INFISICAL_NAMESPACE)" \
		--values hack/infisical-values.yaml \
		--set infisical.image.tag="$(INFISICAL_IMAGE_TAG)" \
		--wait --timeout 15m
	@"$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" wait --for=condition=available \
		deployment/infisical-infisical-standalone-infisical --timeout=10m
	@if ! "$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" get secret infisical-bootstrap-token >/dev/null 2>&1; then \
		"$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" wait --for=condition=complete \
			job/$(INFISICAL_RELEASE)-bootstrap-1 --timeout=10m; \
	fi
	@"$(KUBECTL)" -n "$(INFISICAL_NAMESPACE)" get secret infisical-bootstrap-token >/dev/null

.PHONY: kind-deploy
kind-deploy: kind-up docker-build kind-load-image install deploy ## Build and deploy the operator into Kind.

.PHONY: kind-load-image
kind-load-image: ## Load IMG into the isolated Kind cluster.
	"$(KIND)" load docker-image "$(IMG)" --name "$(KIND_CLUSTER)"

.PHONY: kind-refresh
kind-refresh: docker-build kind-load-image deploy ## Build, load, and restart the operator in Kind.
	"$(KUBECTL)" -n "$(OPERATOR_NAMESPACE)" rollout restart deployment --selector=app.kubernetes.io/instance=$(PROJECT_NAME)
	"$(KUBECTL)" -n "$(OPERATOR_NAMESPACE)" rollout status deployment --selector=app.kubernetes.io/instance=$(PROJECT_NAME) --timeout=5m

.PHONY: kind-e2e
kind-e2e: kind-deploy ## Run the live Infisical reconciliation and network-policy test.
	KIND_CLUSTER="$(KIND_CLUSTER)" OPERATOR_NAMESPACE="$(OPERATOR_NAMESPACE)" INFISICAL_NAMESPACE="$(INFISICAL_NAMESPACE)" ./hack/e2e-kind.sh

.PHONY: kind-down
kind-down: ## Delete only the isolated Kind cluster.
	"$(KIND)" delete cluster --name "$(KIND_CLUSTER)"

##@ Dependencies

$(LOCALBIN):
	@mkdir -p "$@"

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download the pinned Kustomize binary.

$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$@,sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download the pinned controller-gen binary.

$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$@,sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download the pinned golangci-lint binary.

$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$@,github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: kind
kind: $(KIND) ## Download the pinned Kind binary.

KIND_OS ?= linux
KIND_ARCH ?= $(shell $(GO) env GOARCH)

$(KIND): $(LOCALBIN)
	@echo "Downloading kind $(KIND_VERSION)"
	@curl --fail --location --silent --show-error \
		-o "$@" "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(KIND_OS)-$(KIND_ARCH)"
	@chmod +x "$@"

define go-install-tool
@if [ ! -f "$(1)-$(3)" ] || [ -L "$(1)-$(3)" ]; then \
	set -e; \
	echo "Downloading $(2)@$(3)"; \
	rm -f "$(1)" "$(1)-$(3)"; \
	GOBIN="$(LOCALBIN)" $(GO) install $(2)@$(3); \
	mv "$(LOCALBIN)/$$(basename "$(1)")" "$(1)-$(3)"; \
fi; \
ln -sfn "$(1)-$(3)" "$(1)"
endef
