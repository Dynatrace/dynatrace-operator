#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
kustomize_version=v5.3.0
#renovate depName=sigs.k8s.io/controller-tools/cmd
controller_gen_version=v0.14.0
# renovate depName=github.com/golangci/golangci-lint
golang_ci_cmd_version=v1.56.2
# renovate depName=github.com/daixiang0/gci
gci_version=v0.13.0
# renovate depName=golang.org/x/tools
golang_tools_version=v0.18.0
# renovate depName=github.com/vektra/mockery
mockery_version=v2.42.0
# renovate depName=github.com/igorshubovych/markdownlint-cli
markdownlint_cli_version=v0.39.0
# renovate depName=github.com/helm-unittest/helm-unittest
helmunittest_version=v0.4.2
# renovate depName=github.com/princjef/gomarkdoc
gomarkdoc_version=v1.1.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Install all prerequisites
prerequisites: prerequisites/setup-go-dev-dependencies prerequisites/helm-unittest prerequisites/markdownlint prerequisites/gomarkdoc

## Setup go development dependencies
prerequisites/setup-go-dev-dependencies: prerequisites/kustomize prerequisites/controller-gen prerequisites/go-linting prerequisites/mockery

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	go install "sigs.k8s.io/controller-tools/cmd/controller-gen@$(controller_gen_version)"
CONTROLLER_GEN=$(shell hack/build/command.sh controller-gen)

## Install go linters
prerequisites/go-linting:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golang_ci_cmd_version)
	go install github.com/daixiang0/gci@$(gci_version)
	go install golang.org/x/tools/cmd/goimports@$(golang_tools_version)
	go install github.com/bombsimon/wsl/v4/cmd...@master
	go install golang.org/x/tools/cmd/deadcode@$(golang_tools_version)
	go install github.com/dkorunic/betteralign/cmd/betteralign@latest

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
	go install github.com/vektra/mockery/v2@$(mockery_version)

## Install 'pre-commit' if it is missing
prerequisites/setup-pre-commit:
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

## Install 'gomarkdoc' if it is missing
prerequisites/gomarkdoc:
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(gomarkdoc_version)
