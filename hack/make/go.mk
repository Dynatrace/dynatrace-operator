LINT_TARGET ?= ./...

ifeq ($(LINT_TARGET), ./...)
	GCI_TARGET ?= .
else
	GCI_TARGET ?= $(LINT_TARGET)
endif

## Runs go fmt
go/fmt:
	go fmt $(LINT_TARGET)

## Runs gci
go/gci:
	gci write --skip-generated $(GCI_TARGET)

## Runs linters that format/change the code in place
go/format: go/fmt go/gci

## Runs go vet
go/vet:
	go vet -copylocks=false $(LINT_TARGET)

## Runs go wsl linter
go/wsl:
	wsl -fix -allow-trailing-comment ./pkg/...

## Runs golangci-lint
go/golangci:
	$(GOLANGCI_LINT) run --build-tags "$(shell ./hack/build/create_go_build_tags.sh true)" --timeout 300s

go/betteralign:
	$(BETTERALIGN) -apply ./...

## Runs all the linting tools
go/lint: prerequisites/go-linting go/format go/vet go/wsl go/betteralign go/golangci go/deadcode

## Runs all go unit and integration tests and writes the coverprofile to coverage.txt
go/test: prerequisites/setup-envtest
	go test ./... -coverprofile=coverage.txt -covermode=atomic -coverpkg=./... -tags "$(shell ./hack/build/create_go_build_tags.sh false)"

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
	$(DEADCODE) -test -tags="$(shell ./hack/build/create_go_build_tags.sh true)" $(LINT_TARGET) | tee deadcode.out && [ ! -s deadcode.out ]

## Runs go-test-coverage tool
go/check-coverage: prerequisites/go-test-coverage
	$(GO_TEST_COVERAGE) --config=./.testcoverage.yml
