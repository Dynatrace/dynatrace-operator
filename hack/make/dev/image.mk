## Builds operator-builder image (with OS packages an go modules cached)
dev/builder-image:
	docker build . -f ./hack/build/dev/Dockerfile-builder -t operator-builder

GO_LINKER_ARGS := $(shell ./hack/build/create_go_linker_args.sh latestgreatest `git rev-parse HEAD`)
## Builds dynatrace-operator binary using operator-builder image
dev/build-go:
	docker run --env GO_LINKER_ARGS="${GO_LINKER_ARGS}" -v"`pwd`:/in" -v"`pwd`:/out" operator-builder

IMAGE = quay.io/dynatrace/dynatrace-operator:snapshot-$(shell git branch --show-current | sed "s/[^a-zA-Z0-9_-]/-/g")
## Builds dynatrace-operator image based on dynatrace-operator:snapshot with dynatrace-operator binary replaced with local one
dev/build-operator-image: dev/build-go
	docker build . -f ./hack/build/dev/Dockerfile-replace-binaries -t ${IMAGE}

## Builds dynatrace-operator, replaces one in dynatrace-operator:snapshot with it and pushes to quay
dev/build-and-push: dev/build-operator-image
	docker push ${IMAGE}
