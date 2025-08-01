## Runs golangci-lint
go/golangci:
	golangci-lint run --build-tags "$(shell ./hack/build/create_go_build_tags.sh true)" --timeout 300s

## Runs betteralign
go/betteralign:
	betteralign -apply ./...

## Runs all the linting tools
go/lint: prerequisites/go-linting go/betteralign go/golangci go/deadcode

## Runs all go unit and integration tests and writes the coverprofile to coverage.txt
go/test: prerequisites/setup-envtest
	go test ./... -coverprofile=coverage.txt -covermode=atomic -coverpkg=./... -tags "$(shell ./hack/build/create_go_build_tags.sh false)"

## Runs all go unit tests and opens coverage report in a browser
go/coverage: go/test
	go tool cover -html=./coverage.txt

## Runs go integration test
go/integration_test:
	go test -ldflags="-X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Commit=$(shell git rev-parse HEAD)' -X 'github.com/Dynatrace/dynatrace-operator/pkg/version.Version=$(shell git branch --show-current)'" ./test/integration/*

## creates mocks from .mockery.yaml
go/gen_mocks: prerequisites/mockery
	mockery

## Runs deadcode https://go.dev/blog/deadcode
go/deadcode: prerequisites/go-deadcode
	# we add `tee` in the end to make it fail if it finds dead code, by default deadcode always return exit code 0
	deadcode -test -tags="$(shell ./hack/build/create_go_build_tags.sh true)" ./... | tee deadcode.out && [ ! -s deadcode.out ]

## Runs go-test-coverage tool
go/check-coverage: prerequisites/go-test-coverage
	go-test-coverage --config=./.testcoverage.yml
