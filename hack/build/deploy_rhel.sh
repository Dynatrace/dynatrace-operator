#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

commit=$(git rev-parse HEAD)
build_date="$(date -u +"%Y-%m-%d %H:%M:%S+00:00")"
go_build_args=(
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Version=${TAG}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.BuildDate=${build_date}'"
)
base_image="dynatrace-operator"
out_image="${IMG:-quay.io/dynatrace/dynatrace-operator}:${TAG}"

args="${go_build_args[@]}"
if [[ "${LOCALBUILD}" ]]; then
  export GOOS=linux
  export GOARCH=${GOARCH:-amd64}

  # directory required by docker copy command
  mkdir -p third_party_licenses
  docker build . -f ./Dockerfile -t "${base_image}" \
    --build-arg "GO_LINKER_ARGS=${args}" \
    --build-arg "TAGS_ARG=exclude_graphdriver_btrfs" \
    --label "quay.expires-after=14d" \
    --no-cache
  rm -rf third_party_licenses
else
  # directory required by docker copy command
  mkdir -p third_party_licenses
  docker build . -f ./Dockerfile -t "${base_image}" \
    --build-arg "GO_LINKER_ARGS=${args}" \
    --label "quay.expires-after=14d" \
    --no-cache
  rm -rf third_party_licenses
fi

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
