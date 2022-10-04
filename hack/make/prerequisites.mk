# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

## Installs 'kustomize' if it is missing
prerequisites/kustomize:
	hack/build/command.sh kustomize "sigs.k8s.io/kustomize/kustomize/v4@v4.5.7"
KUSTOMIZE=$(shell hack/build/command.sh kustomize)

## Install 'controller-gen' if it is missing
prerequisites/controller-gen:
	hack/build/command.sh controller-gen "sigs.k8s.io/controller-tools/cmd/controller-gen@v0.10.0"
CONTROLLER_GEN=$(shell hack/build/command.sh controller-gen)

prerequisites/setup-pre-commit:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0
	go install github.com/daixiang0/gci@v0.8.0
	go install golang.org/x/tools/cmd/goimports@v0.1.12
	cp ./.github/pre-commit ./.git/hooks/pre-commit
	chmod +x ./.git/hooks/pre-commit

prerequisites/helm:
	hack/helm/install-unittest-plugin.sh
