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
	gci write $(GCI_TARGET)

## Runs linters that format/change the code in place
go/format: go/fmt go/gci

## Runs go vet
go/vet:
	go vet -copylocks=false $(LINT_TARGET)

## Runs golangci-lint
go/golangci:
	golangci-lint run --build-tags "containers_image_openpgp,osusergo,netgo,sqlite_omit_load_extension,containers_image_storage_stub,containers_image_docker_daemon_stub,e2e" --timeout 300s

## Runs all the linting tools
go/lint: go/format go/vet go/golangci

## Runs all go unit tests and writes the coverprofile to cover.out
go/test:
	go test ./... -coverprofile cover.out -tags "containers_image_openpgp,osusergo,netgo,sqlite_omit_load_extension,containers_image_storage_stub,containers_image_docker_daemon_stub"

