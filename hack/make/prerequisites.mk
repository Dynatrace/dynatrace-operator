#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
kustomize_version=v5.5.0
#renovate depName=sigs.k8s.io/controller-tools/cmd
controller_gen_version=v0.16.4
# renovate depName=github.com/golangci/golangci-lint
golang_ci_cmd_version=v1.61.0
# renovate depName=github.com/daixiang0/gci
gci_version=v0.13.5
# renovate depName=golang.org/x/tools
golang_tools_version=v0.26.0
# renovate depName=github.com/vektra/mockery
mockery_version=v2.46.3
# renovate depName=github.com/igorshubovych/markdownlint-cli
markdownlint_cli_version=v0.42.0
# renovate depName=github.com/helm-unittest/helm-unittest
helmunittest_version=v0.6.3
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
prerequisites/go-linting: prerequisites/go-deadcode
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golang_ci_cmd_version)
	go install github.com/daixiang0/gci@$(gci_version)
	go install golang.org/x/tools/cmd/goimports@$(golang_tools_version)
	go install github.com/bombsimon/wsl/v4/cmd...@master
	go install github.com/dkorunic/betteralign/cmd/betteralign@latest

## Install go deadcode
prerequisites/go-deadcode:
	go install golang.org/x/tools/cmd/deadcode@$(golang_tools_version)

## Install go test coverage
prerequisites/go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

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

## Install python dependencies
prerequisites/python:
	python3 -m venv local/.venv && source local/.venv/bin/activate && pip3 install -r hack/requirements.txt
