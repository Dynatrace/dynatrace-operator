#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "usage: 'make deploy-local TAG=\"<your-image-tag>\"'"
  exit 5
fi

export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64

commit=$(git rev-parse HEAD)
build_date="$(date -u --rfc-3339=seconds)"
go_build_args=(
  "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/version.Version=${TAG}' -X 'github.com/Dynatrace/dynatrace-operator/version.Commit=${commit}' -X 'github.com/Dynatrace/dynatrace-operator/version.BuildDate=${build_date}'"
  "-tags" "containers_image_storage_stub"
)
base_image="dynatrace-operator"
out_image="quay.io/dynatrace/dynatrace-operator:${TAG}"

go build "${go_build_args[@]}" -o ./build/_output/bin/dynatrace-operator ./cmd/operator/

go get github.com/google/go-licenses
go-licenses save ./... --save_path third_party_licenses --force

docker build . -f ./Dockerfile -t "${base_image}" --label "quay.expires-after=14d" --no-cache
docker tag "${base_image}" "${out_image}"
docker push "${out_image}"

rm -rf ./third_party_licenses
