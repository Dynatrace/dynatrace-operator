#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

commit=$(git rev-parse HEAD)
build_date="$(date -u +"%Y-%m-%d %H:%M:%S+00:00")"
go_build_args=(
  "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/version.Version=${TAG}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/version.BuildDate=${build_date}'"
  "-linkmode external -extldflags '-static' -s -w"
)
base_image="dynatrace-operator"
out_image="quay.io/dynatrace/dynatrace-operator:${TAG}"

args=${go_build_args[@]}
docker build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_BUILD_ARGS=$args" --label "quay.expires-after=14d" --no-cache
docker tag "${base_image}" "${out_image}"
docker push "${out_image}"

rm -rf ./third_party_licenses
