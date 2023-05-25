#renovate depName=sigs.k8s.io/kustomize/kustomize/v5
KUSTOMIZE_VERSION=v5.0.0
#renovate depName=sigs.k8s.io/kustomize/kustomize
CONTROLLER_GEN_VERSION=v0.12.0
# renovate depName=github.com/golangci/golangci-lint
KUSTOMIZE_VERSION=v5.0.0
# renovate depName=github.com/daixiang0/gci
GCI_VERSION=v0.10.0
# renovate depName=golang.org/x/tools
GOLANG_TOOLS_VERSION=v0.9.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	hack/build/command.sh kustomize "sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)"
KUSTOMIZE=$(shell hack/build/command.sh kustomize)

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	hack/build/command.sh controller-gen "sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)"
CONTROLLER_GEN=$(shell hack/build/command.sh controller-gen)

prerequisites/setup-pre-commit:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(KUSTOMIZE_VERSION)
	go install github.com/daixiang0/gci@v$(GCI_VERSION)
	go install golang.org/x/tools/cmd/goimports@$(GOLANG_TOOLS_VERSION)
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

prerequisites/helm:
	hack/helm/install-unittest-plugin.sh
