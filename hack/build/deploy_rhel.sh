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
  export CGO_ENABLED=1
  export GOOS=linux
  export GOARCH=${GOARCH:-amd64}

  go build -ldflags "$args" -tags exclude_graphdriver_btrfs -o ./build/_output/bin/dynatrace-operator ./src/cmd/operator/

  if [[ "$?" != 0 ]]; then
	  echo "ERROR: go build exited abnormally. Aborting..."
	  exit 10
  fi

  go get github.com/google/go-licenses
  go-licenses save ./... --save_path third_party_licenses --force

  docker build . -f ./Dockerfile-localbuild -t "${base_image}" --label "quay.expires-after=14d" --no-cache

  rm -rf ./third_party_licenses
else
  # directory required by docker copy command
  mkdir -p third_party_licenses
  docker build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_BUILD_ARGS=$args" --label "quay.expires-after=14d" --no-cache
  rm -rf third_party_licenses
fi

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
