# Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOBIN ?= $(LOCALBIN)

# Location to install npm binaries to
LOCALBIN_NPM ?= $(shell pwd)/node_modules/.bin

#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
KUSTOMIZE_VERSION ?= v5.8.1
#renovate depName=sigs.k8s.io/controller-tools/cmd
CONTROLLER_GEN_VERSION ?= v0.20.0
# renovate depName=github.com/golangci/golangci-lint/v2
GOLANGCI_LINT_VERSION ?= v2.9.0
# renovate depName=golang.org/x/tools
GOLANG_TOOLS_VERSION ?= v0.42.0
# renovate depName=github.com/vektra/mockery
MOCKERY_VERSION ?= v3.6.4
# renovate depName=github.com/igorshubovych/markdownlint-cli
MARKDOWNLINT_CLI_VERSION ?= v0.47.0
# renovate depName=github.com/tcort/markdown-link-check
MARKDOWN_LINK_CHECK_VERSION ?= v3.14.2
# renovate depName=github.com/helm-unittest/helm-unittest
HELMUNITTEST_VERSION ?= v1.0.3
# renovate depName=github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod
CYCLONEDX_GOMOD_VERSION ?= v1.10.0
# renovate depName=github.com/mikefarah/yq/v4
YQ_VERSION ?= v4.52.2
# renovate depName=github.com/vladopajic/go-test-coverage/v2
GO_TEST_COVERAGE_VERSION ?= v2.18.3

# Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
DEADCODE ?= $(LOCALBIN)/deadcode
MOCKERY ?= $(LOCALBIN)/mockery
CYCLONEDX_GOMOD ?= $(LOCALBIN)/cyclonedx-gomod
YQ ?= $(LOCALBIN)/yq
GO_TEST_COVERAGE ?= $(LOCALBIN)/go-test-coverage
SETUP_ENVTEST ?= $(LOCALBIN)/setup-envtest
PYTHON ?= $(LOCALBIN)/.venv/bin/python3
MARKDOWNLINT ?= $(LOCALBIN_NPM)/markdownlint
MARKDOWN_LINK_CHECK ?= $(LOCALBIN_NPM)/markdown-link-check

#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell v='$(call gomodver,sigs.k8s.io/controller-runtime)'; \
	[ -n "$$v" ] || { echo "Set ENVTEST_VERSION manually (controller-runtime replace has no tag)" >&2; exit 1; }; \
	printf '%s\n' "$$v" | sed -E 's/^v?([0-9]+)\.([0-9]+).*/release-\1.\2/')

#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell v='$(call gomodver,k8s.io/api)'; \
	[ -n "$$v" ] || { echo "Set ENVTEST_K8S_VERSION manually (k8s.io/api replace has no tag)" >&2; exit 1; }; \
	printf '%s\n' "$$v" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')

## Install all prerequisites
prerequisites: prerequisites/setup-go-dev-dependencies prerequisites/helm-unittest prerequisites/markdownlint prerequisites/git-ignore-revs-file

## Setup go development dependencies
prerequisites/setup-go-dev-dependencies: prerequisites/kustomize prerequisites/controller-gen prerequisites/go-linting prerequisites/mockery

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_GEN_VERSION))

## Install go linters
prerequisites/go-linting: prerequisites/go-deadcode
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

## Install go deadcode
prerequisites/go-deadcode:
	$(call go-install-tool,$(DEADCODE),golang.org/x/tools/cmd/deadcode,$(GOLANG_TOOLS_VERSION))

## Install go test coverage
prerequisites/go-test-coverage:
	$(call go-install-tool,$(GO_TEST_COVERAGE),github.com/vladopajic/go-test-coverage/v2,$(GO_TEST_COVERAGE_VERSION))

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

## Install verktra/mockery
prerequisites/mockery:
	$(call go-install-tool,$(MOCKERY),github.com/vektra/mockery/v3,$(MOCKERY_VERSION))

## Install 'cyclonedx-gomod' if it is missing
prerequisites/cyclonedx-gomod:
	$(call go-install-tool,$(CYCLONEDX_GOMOD),github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod,$(CYCLONEDX_GOMOD_VERSION))

## Install 'yq' if it is missing
prerequisites/yq:
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,$(YQ_VERSION))

## Install setup-envtest locally
prerequisites/envtest:
	$(call go-install-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

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
prerequisites/helm-unittest:
## TODO: Have version accessible by renovate?
	hack/helm/install-unittest-plugin.sh $(HELMUNITTEST_VERSION)

## Install 'markdownlint' if it is missing
prerequisites/markdownlint:
	npm install markdownlint-cli@$(MARKDOWNLINT_CLI_VERSION)

## Install 'markdown-link-check' if it is missing
prerequisites/markdown-link-check:
	npm install markdown-link-check@$(MARKDOWN_LINK_CHECK_VERSION)

## Install python dependencies
prerequisites/python:
	python3 -m venv $(LOCALBIN)/.venv && source $(LOCALBIN)/.venv/bin/activate && pip3 install -r hack/requirements.txt

## Install 'pre-commit' if it is missing
prerequisites/setup-pre-commit:
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

## Configure git to use .git-blame-ignore-revs file
prerequisites/git-ignore-revs-file:
	git config blame.ignoreRevsFile .git-blame-ignore-revs

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

define gomodver
$(shell go list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' $(1) 2>/dev/null)
endef
