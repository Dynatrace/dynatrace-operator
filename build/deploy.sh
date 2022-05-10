#!/bin/bash

set -eu

go_build_args=$(hack/github_actions.sh createGoBuildArgs "${TAG}" "${commit}")

if [[ "${GCR:-}" == "true" ]]; then
  echo "$GCLOUD_SERVICE_KEY" | base64 -d | docker login -u _json_key --password-stdin https://gcr.io
  gcloud --quiet config set project "$GCP_PROJECT"
fi

base_image="dynatrace-operator"

if [[ -z "${LABEL:-}" ]]; then
  docker build . -f ./Dockerfile -t "$base_image" --build-arg "GO_BUILD_ARGS=$go_build_args"
else
  docker build . -f ./Dockerfile -t "$base_image" --build-arg "GO_BUILD_ARGS=$go_build_args" --label "$LABEL"
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
