ENVTEST_ASSETS_DIR = $(shell pwd)/testbin

## Runs go fmt
go/fmt:
	go fmt ./...

## Runs go vet
go/vet:
	go vet -copylocks=false ./...

## Lints the go code
go/lint: go/fmt go/vet
	gci write .
	golangci-lint run --build-tags containers_image_storage_stub,e2e --timeout 300s

## Runs go unit tests (without containers_image_storage_stub) and writes the coverprofile to cover.out
go/test: go/fmt
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

## Runs all go unit tests and writes the coverprofile to cover.out
go/test-all:
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out -tags containers_image_storage_stub

