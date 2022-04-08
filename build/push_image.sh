#!/bin/bash

if [ ${CONTAINER_CLI} == "" ]
then
    CONTAINER_CLI=docker
fi


if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

commit=$(git rev-parse HEAD)
build_date="$(date -u +"%Y-%m-%dT%H:%M:%S+00:00")"
go_build_args=(
  "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/src/version.Version=${TAG}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.BuildDate=${build_date}'"
  "-linkmode external -extldflags '-static' -s -w"
)
base_image="dynatrace-operator"
out_image="quay.io/dynatrace/dynatrace-operator:${TAG}"


args="${go_build_args[@]}"
if [[ "${LOCALBUILD}" ]]; then
  export CGO_ENABLED=1
  export GOOS=linux
  export GOARCH=amd64

  go build "$args" -o ./build/_output/bin/dynatrace-operator ./src/cmd/operator/

  go get github.com/google/go-licenses
  go-licenses save ./... --save_path third_party_licenses --force

  $CONTAINER_CLI build . -f ./Dockerfile-localbuild -t "${base_image}" --label "quay.expires-after=14d" --no-cache

  rm -rf ./third_party_licenses
else
  # directory required by $CONTAINER_CLI copy command
  mkdir -p third_party_licenses
  $CONTAINER_CLI build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_BUILD_ARGS=$args" --label "quay.expires-after=14d" --no-cache
  rm -rf third_party_licenses
fi

$CONTAINER_CLI tag "${base_image}" "${out_image}"
$CONTAINER_CLI push "${out_image}"
