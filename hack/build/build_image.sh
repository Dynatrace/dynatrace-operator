#!/bin/bash

if [[ ! "${1}" ]]; then
  echo "1 param is not set, should be the image without the tag"
  exit 5
fi
if [[ ! "${2}" ]]; then
  echo "2 param is not set, should be the tag of the image"
  exit 5
fi
IMAGE=${1}
TAG=${2}

commit=$(git rev-parse HEAD)
go_linker_args=$(hack/build/create_go_linker_args.sh "${TAG}" "${commit}")

base_image="dynatrace-operator"
out_image="${IMAGE}:${TAG}"

# directory required by docker copy command
mkdir -p third_party_licenses
docker build . -f ./Dockerfile -t "${base_image}" \
  --build-arg "GO_LINKER_ARGS=${go_linker_args}" \
  --label "quay.expires-after=14d" \
rm -rf third_party_licenses

docker tag "${base_image}" "${out_image}"

