#!/bin/bash

set -eu

build_date="$(date -u --rfc-3339=seconds)"
go_build_args=(
  "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/version.Version=${TAG}' -X 'github.com/Dynatrace/dynatrace-operator/version.Commit=${COMMIT}' -X 'github.com/Dynatrace/dynatrace-operator/version.BuildDate=${build_date}'"
  "-tags" "containers_image_storage_stub"
)

go build "${go_build_args[@]}" -o ./build/_output/bin/dynatrace-operator ./cmd/operator/
go build "${go_build_args[@]}" -o ./build/_output/bin/csi-driver ./cmd/csidriver

if [[ "${GCR:-}" == "true" ]]; then
  echo "$GCLOUD_SERVICE_KEY" | base64 -d | docker login -u _json_key --password-stdin https://gcr.io
  gcloud --quiet config set project "$GCP_PROJECT"
fi

go get github.com/google/go-licenses
go-licenses save ./... --save_path third_party_licenses

base_image="dynatrace-operator"

if [[ -z "${LABEL:-}" ]]; then
  docker build . -f ./Dockerfile -t "$base_image"
else
  docker build . -f ./Dockerfile -t "$base_image" --label "$LABEL"
fi

failed=false

read -ra images <<<"$IMAGES"
for image in ${images[@]}; do
  out_image="$image:$TAG"
  if [[ "$image" != "$OAO_IMAGE_RHCC_SCAN" ]]; then
    out_image="$out_image-$TRAVIS_CPU_ARCH"
  fi

  echo "Building docker image: $out_image"
  docker tag "$base_image" "$out_image"

  echo "Pushing docker image: $out_image"
  if ! docker push "$out_image"; then
    echo "Failed to push docker image: $out_image"
    failed=true
  fi
done

if [[ "$failed" == "true" ]]; then
  exit 1
fi
