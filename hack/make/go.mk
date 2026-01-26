LOCAL_GO_VERSION := $(shell go version | awk '{print $$3}')
DOCKER_GO_VERSION := $(shell cat Dockerfile | grep golang | cut -d "@" -f1 | cut -d ":" -f2)

## Check go version
go/check-version:
	@if [ "$(LOCAL_GO_VERSION)" != "go$(DOCKER_GO_VERSION)" ]; then \
		printf "\033[0;31mGo version mismatch: Local version is $(LOCAL_GO_VERSION) but Dockerfile requires go$(DOCKER_GO_VERSION)\033[0m\n"; \
	fi

## Runs golangci-lint
go/golangci: go/check-version
	$(GOLANGCI_LINT) run --fix

## Runs all the linting tools
go/lint: prerequisites/go-linting go/golangci go/deadcode

## Runs all go unit and integration tests and writes the coverprofile to coverage.txt
go/test: prerequisites/setup-envtest go/check-version
	go test ./... -coverprofile=coverage.txt -covermode=atomic -coverpkg=./...

## Runs all go unit tests and opens coverage report in a browser
go/coverage: go/test
	go tool cover -html=./coverage.txt

## Runs go integration test
go/integration_test: prerequisites/setup-envtest
	go test -ldflags="-X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Commit=$(shell git rev-parse HEAD)' -X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Version=$(shell git branch --show-current)'" ./test/integration/*

## creates mocks from .mockery.yaml
go/gen_mocks: prerequisites/mockery
	$(MOCKERY)

## Runs deadcode https://go.dev/blog/deadcode
go/deadcode: prerequisites/go-deadcode
	# we add `tee` in the end to make it fail if it finds dead code, by default deadcode always return exit code 0
	$(DEADCODE) -test -tags=e2e ./... | tee deadcode.out && [ ! -s deadcode.out ]

## Runs go-test-coverage tool
go/check-coverage: prerequisites/go-test-coverage
	$(GO_TEST_COVERAGE) --config=./.testcoverage.yml
