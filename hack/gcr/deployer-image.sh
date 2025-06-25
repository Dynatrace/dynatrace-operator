#!/usr/bin/env bash

set -eu

export REGISTRY=quay.io/dynatrace
export APP_NAME=dynatrace-operator
TAG="${1:-""}"

if podman build --tag "$REGISTRY/$APP_NAME/deployer${TAG}" -f config/helm/Dockerfile config/helm; then
  podman push "$REGISTRY/$APP_NAME/deployer${TAG}"
  echo "Deployer image built successfully: $REGISTRY/$APP_NAME/deployer${TAG}"
fi
