#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  exit 5
fi

commit=$(git rev-parse HEAD)
build_date="$(date -u +"%Y-%m-%d %H:%M:%S+00:00")"
go_build_args=(
  "-linkmode external -extldflags '-static' -s -w"
  "-X 'github.com/Dynatrace/dynatrace-operator/version.Version=${TAG}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/version.BuildDate=${build_date}'"
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

  docker build . -f ./Dockerfile-localbuild -t "${base_image}" --label "quay.expires-after=14d" --no-cache

  rm -rf ./third_party_licenses
else
  echo Saving licenses
  mkdir -p third_party_licenses
  if ! command -v go-licenses &> /dev/null
  then
    go get github.com/google/go-licenses
  fi
  go-licenses save ./... --save_path third_party_licenses --force
  docker build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_BUILD_ARGS=$args" --label "quay.expires-after=14d" --no-cache
  rm -rf third_party_licenses
fi

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
