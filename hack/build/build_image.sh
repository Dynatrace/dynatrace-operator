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
dockerfile=${4:-Dockerfile}
platform=${5:-linux/amd64}

commit=$(git rev-parse HEAD)
go_linker_args=$(hack/build/create_go_linker_args.sh "${tag}" "${commit}" "${debug}")
go_build_tags=""

out_image="${image}:${tag}"

# directory required by docker copy command
mkdir -p third_party_licenses

if ! command -v docker 2>/dev/null; then
  CONTAINER_CMD=podman
else
  CONTAINER_CMD=docker
fi

${CONTAINER_CMD} build "--platform=${platform}" . -f "${dockerfile}" -t "${out_image}" \
  --build-arg "GO_LINKER_ARGS=${go_linker_args}" \
  --build-arg "GO_BUILD_TAGS=${go_build_tags}" \
  --build-arg "DEBUG_TOOLS=${debug}" \
  --label "quay.expires-after=14d"

rm -rf third_party_licenses
