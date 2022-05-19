#!/bin/bash

pushDockerImage() {
  TAG=$1

  if [[ -f "/tmp/operator-arm64.tar" ]]; then
    echo "we build for arm too => combine images"
    docker load --input /tmp/operator-arm64.tar
    docker tag operator-arm64:${TAG} ${IMAGE_QUAY}:${TAG}-arm64
    docker tag operator-amd64:${TAG} ${IMAGE_QUAY}:${TAG}-amd64
    docker push ${IMAGE_QUAY}:${TAG}-arm64
    docker push ${IMAGE_QUAY}:${TAG}-amd64

    docker manifest create ${IMAGE_QUAY}:${TAG} ${IMAGE_QUAY}:${TAG}-arm64 ${IMAGE_QUAY}:${TAG}-amd64
    docker manifest push ${IMAGE_QUAY}:${TAG}
  else
    docker tag operator-amd64:${TAG} ${IMAGE_QUAY}:${TAG}
    docker push ${IMAGE_QUAY}:${TAG}
  fi
}

pushDockerImage $1
