#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

commit=$(git rev-parse HEAD)
go_linker_args=$(hack/build/create_go_build_args.sh "${TAG}" "${commit}")
base_image="dynatrace-operator"
out_image="${IMG:-quay.io/dynatrace/dynatrace-operator}:${TAG}"

if [[ "${LOCALBUILD}" ]]; then
  export CGO_ENABLED=1
  export GOOS=linux
  export GOARCH=amd64

  go build -ldflags="$go_linker_args" -o ./build/_output/bin/dynatrace-operator ./src/cmd/operator/

  go get github.com/google/go-licenses
  go-licenses save ./... --save_path third_party_licenses --force

  docker build . -f ./Dockerfile-localbuild -t "${base_image}" --label "quay.expires-after=14d" --no-cache

  rm -rf ./third_party_licenses
else
  # directory required by docker copy command
  mkdir -p third_party_licenses
  docker build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_LINKER_ARGS=$go_linker_args" --label "quay.expires-after=14d" --no-cache
  rm -rf third_party_licenses
fi

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
