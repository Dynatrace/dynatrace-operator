#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
kustomize_version=v5.0.3
#renovate depName=sigs.k8s.io/controller-tools/cmd
controller_gen_version=v0.12.0
# renovate depName=github.com/golangci/golangci-lint
golang_ci_cmd_version=v1.53.2
# renovate depName=github.com/daixiang0/gci
gci_version=v0.10.1
# renovate depName=golang.org/x/tools
golang_tools_version=v0.9.3

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	hack/build/command.sh kustomize "sigs.k8s.io/kustomize/kustomize/v5@$(kustomize_version)"
KUSTOMIZE=$(shell hack/build/command.sh kustomize)

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	hack/build/command.sh controller-gen "sigs.k8s.io/controller-tools/cmd/controller-gen@$(controller_gen_version)"
CONTROLLER_GEN=$(shell hack/build/command.sh controller-gen)

prerequisites/setup-pre-commit:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golang_ci_cmd_version)
	go install github.com/daixiang0/gci@$(gci_version)
	go install golang.org/x/tools/cmd/goimports@$(golang_tools_version)
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

prerequisites/helm:
	hack/helm/install-unittest-plugin.sh
