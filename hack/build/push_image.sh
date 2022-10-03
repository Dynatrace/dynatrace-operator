#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

DOCKERFILE="${DOCKERFILE:-"./Dockerfile"}"

commit=$(git rev-parse HEAD)
go_linker_args=$(hack/build/create_go_linker_args.sh "${TAG}" "${commit}")
base_image="dynatrace-operator"
out_image="${IMG:-quay.io/dynatrace/dynatrace-operator}:${TAG}"

# directory required by docker copy command
mkdir -p third_party_licenses
docker build . -f "${DOCKERFILE}" -t "${base_image}" \
  --build-arg "GO_LINKER_ARGS=${go_linker_args}" \
  --label "quay.expires-after=14d" \
  --no-cache
rm -rf third_party_licenses

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
