#!/bin/bash

if [[ ! "${1}" ]]; then
  echo "first param is not set, should be the image without the tag"
  exit 1
fi
if [[ ! "${2}" ]]; then
  echo "second param is not set, should be the tag of the image"
  exit 1
fi

image=${1}
tag=${2}
debug=${3:-false}

commit=$(git rev-parse HEAD)
go_linker_args=$(hack/build/create_go_linker_args.sh "${tag}" "${commit}" "${debug}")
go_build_tags=$(hack/build/create_go_build_tags.sh false)

out_image="${image}:${tag}"

# directory required by docker copy command
mkdir -p third_party_licenses

if ! command -v docker 2>/dev/null; then
  CONTAINER_CMD=podman
else
  CONTAINER_CMD=docker
fi

OPERATOR_BUILD_PLATFORM="--platform=linux/amd64"
if [ -n "${OPERATOR_DEV_BUILD_PLATFORM}" ]; then
  echo "overriding platform to ${OPERATOR_DEV_BUILD_PLATFORM}"
  OPERATOR_BUILD_PLATFORM="--platform=${OPERATOR_DEV_BUILD_PLATFORM}"
fi

${CONTAINER_CMD} build "${OPERATOR_BUILD_PLATFORM}" . -f ./Dockerfile -t "${out_image}" \
  --build-arg "GO_LINKER_ARGS=${go_linker_args}" \
  --build-arg "GO_BUILD_TAGS=${go_build_tags}" \
  --build-arg "DEBUG_TOOLS=${debug}" \
  --label "quay.expires-after=14d"

rm -rf third_party_licenses
