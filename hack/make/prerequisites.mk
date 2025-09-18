#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
kustomize_version=v5.7.1
#renovate depName=sigs.k8s.io/controller-tools/cmd
controller_gen_version=v0.19.0
# renovate depName=github.com/golangci/golangci-lint/v2
golang_ci_cmd_version=v2.4.0
# renovate depName=github.com/daixiang0/gci
gci_version=v0.13.7
# renovate depName=golang.org/x/tools
golang_tools_version=v0.37.0
# renovate depName=github.com/vektra/mockery
mockery_version=v3.5.5
# renovate depName=github.com/igorshubovych/markdownlint-cli
markdownlint_cli_version=v0.45.0
# renovate depName=github.com/helm-unittest/helm-unittest
helmunittest_version=v1.0.1
# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
cyclonedx_gomod_version=v1.9.0

ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
# 1.34 does not work yet, see [DAQ-13500]
ENVTEST_K8S_VERSION = 1.33

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Install all prerequisites
prerequisites: prerequisites/setup-go-dev-dependencies prerequisites/helm-unittest prerequisites/markdownlint

## Setup go development dependencies
prerequisites/setup-go-dev-dependencies: prerequisites/kustomize prerequisites/controller-gen prerequisites/go-linting prerequisites/mockery

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$(controller_gen_version)"
CONTROLLER_GEN=$(shell hack/build/command.sh controller-gen)

## Install go linters
prerequisites/go-linting: prerequisites/go-deadcode
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(golang_ci_cmd_version)
	go install golang.org/x/tools/cmd/goimports@$(golang_tools_version)
	go install github.com/dkorunic/betteralign/cmd/betteralign@latest

## Install go deadcode
prerequisites/go-deadcode:
	go install golang.org/x/tools/cmd/deadcode@$(golang_tools_version)

## Install go test coverage
prerequisites/go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

## Install setup-envtest locally
prerequisites/envtest:
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

prerequisites/setup-envtest: prerequisites/envtest
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(GOBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

## Install 'helm' if it is missing
## TODO: Have version accessible by renovate?
prerequisites/helm-unittest:
	hack/helm/install-unittest-plugin.sh $(helmunittest_version)

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	go install "sigs.k8s.io/kustomize/kustomize/v5@$(kustomize_version)"
KUSTOMIZE=$(shell hack/build/command.sh kustomize)

## Install 'markdownlint' if it is missing
prerequisites/markdownlint:
	npm install -g --force markdownlint-cli@$(markdownlint_cli_version)

## Install verktra/mockery
prerequisites/mockery:
	go install github.com/vektra/mockery/v3@$(mockery_version)

## Install 'pre-commit' if it is missing
prerequisites/setup-pre-commit:
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

## Install python dependencies
prerequisites/python:
	python3 -m venv local/.venv && source local/.venv/bin/activate && pip3 install -r hack/requirements.txt

## Install 'cyclonedx-gomod' if it is missing
prerequisites/cyclonedx-gomod:
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(cyclonedx_gomod_version)
