-include manifests/*.mk

ENVTEST_ASSETS_DIR = $(shell pwd)/testbin

## Runs go fmt
go/fmt:
	go fmt ./...

## Runs go vet
go/vet:
	go vet ./...

## Lints the go code
go/lint: go/fmt go/vet
	gci -w .
	golangci-lint run --build-tags integration,containers_image_storage_stub --timeout 300s

## Runs go unit tests and writes the coverprofile to cover.out
go/test: manifests/kubernetes manifests/openshift go/fmt
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

## Runs the Operator using the configured Kubernetes cluster in ~/.kube/config
go/run: export RUN_LOCAL=true
go/run: export POD_NAMESPACE=dynatrace
go/run: manifests/kubernetes manifests/openshift go/fmt go/vet
	go run ./src/cmd/operator/

## Builds the Operators binary and writes it to bin/manager
go/build/manager: manifests/generate-crd go/fmt go/vet
	go build -o bin/manager ./src/cmd/operator/

## Builds the Operators binary specifically for AMD64 and writes it to bin/manager
go/build/manager/amd64: export GOOS=linux
go/build/manager/amd64: export GOARCH=amd64
go/build/manager/amd64: manifests/generate-crd go/fmt go/vet
	go build -o bin/manager-amd64 ./src/cmd/operator/