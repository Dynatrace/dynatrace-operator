## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOBIN ?= $(LOCALBIN)

# Location to install npm binaries to
LOCALBIN_NPM ?= $(shell pwd)/node_modules/.bin

#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
KUSTOMIZE_VERSION ?= v5.6.0
#renovate depName=sigs.k8s.io/controller-tools/cmd
CONTROLLER_GEN_VERSION ?= v0.18.0
# renovate depName=github.com/golangci/golangci-lint/v2
GOLANGCI_LINT_VERSION ?= v2.1.6
# renovate depName=github.com/daixiang0/gci
GCI_VERSION ?= v0.13.7
# renovate depName=golang.org/x/tools
GOLANG_TOOLS_VERSION ?= v0.34.0
# renovate depName=github.com/vektra/mockery
MOCKERY_VERSION ?= v2.53.5
# renovate depName=github.com/igorshubovych/markdownlint-cli
MARKDOWNLINT_CLI_VERSION ?= v0.45.0
# renovate depName=github.com/helm-unittest/helm-unittest
HELMUNITTEST_VERSION ?= v0.8.2
# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
CYCLONEDX_GOMOD_VERSION ?= v1.9.0

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GCI ?= $(LOCALBIN)/gci
DEADCODE ?= $(LOCALBIN)/deadcode
GOIMPORTS ?= $(LOCALBIN)/goimports
MOCKERY ?= $(LOCALBIN)/mockery
CYCLONEDX_GOMOD ?= $(LOCALBIN)/cyclonedx-gomod
BETTERALIGN ?= $(LOCALBIN)/betteralign
GO_TEST_COVERAGE ?= $(LOCALBIN)/go-test-coverage
SETUP_ENVTEST ?= $(LOCALBIN)/setup-envtest
WSL ?= $(LOCALBIN)/wsl
PYTHON ?= $(LOCALBIN)/.venv/bin/python3
MARKDOWNLINT ?= $(LOCALBIN_NPM)/markdownlint

ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
# ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
ENVTEST_K8S_VERSION = 1.32 # TODO: change it when env test list add it and cotroller-runtime lib updated

## Install all prerequisites
prerequisites: prerequisites/setup-go-dev-dependencies prerequisites/helm-unittest prerequisites/markdownlint

## Setup go development dependencies
prerequisites/setup-go-dev-dependencies: prerequisites/kustomize prerequisites/controller-gen prerequisites/go-linting prerequisites/mockery

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_GEN_VERSION))

## Install go linters
prerequisites/go-linting: prerequisites/go-deadcode
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	$(call go-install-tool,$(GCI),github.com/daixiang0/gci,$(GCI_VERSION))
	$(call go-install-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports,$(GOLANG_TOOLS_VERSION))
	$(call go-install-tool,$(WSL),github.com/bombsimon/wsl/v4/cmd...,master)
	$(call go-install-tool,$(BETTERALIGN),github.com/dkorunic/betteralign/cmd/betteralign,latest)

## Install go deadcode
prerequisites/go-deadcode:
	$(call go-install-tool,$(DEADCODE),golang.org/x/tools/cmd/deadcode,$(GOLANG_TOOLS_VERSION))

## Install go test coverage
prerequisites/go-test-coverage:
	$(call go-install-tool,$(GO_TEST_COVERAGE),github.com/vladopajic/go-test-coverage/v2,latest)

## Install setup-envtest locally
prerequisites/envtest:
	$(call go-install-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,latest)

## Setup envtest binaries for the specified Kubernetes version
prerequisites/setup-envtest: prerequisites/envtest
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}
	@echo
	@$(SETUP_ENVTEST) cleanup "<$(ENVTEST_K8S_VERSION)" --bin-dir $(LOCALBIN)
	@echo "Setup of envtest binaries completed."

## Install 'helm' if it is missing
## TODO: Have version accessible by renovate?
prerequisites/helm-unittest:
	hack/helm/install-unittest-plugin.sh $(HELMUNITTEST_VERSION)

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

## Install 'markdownlint' if it is missing
prerequisites/markdownlint:
	npm install --force markdownlint-cli@$(MARKDOWNLINT_CLI_VERSION)

## Install verktra/mockery
prerequisites/mockery:
	$(call go-install-tool,$(MOCKERY),github.com/vektra/mockery/v2,$(MOCKERY_VERSION))

## Install 'pre-commit' if it is missing
prerequisites/setup-pre-commit:
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

## Install python dependencies
prerequisites/python:
	python3 -m venv $(LOCALBIN)/.venv && source $(LOCALBIN)/.venv/bin/activate && pip3 install -r hack/requirements.txt

## Install 'cyclonedx-gomod' if it is missing
prerequisites/cyclonedx-gomod:
	$(call go-install-tool,$(CYCLONEDX_GOMOD),github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod,$(CYCLONEDX_GOMOD_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $$(realpath $(1)-$(3)) $(1)
endef
