#!/usr/bin/env bash

set -eu

export REGISTRY=quay.io/dynatrace
export APP_NAME=dynatrace-operator
TAG="${1:-""}"

if docker build --tag "$REGISTRY/$APP_NAME/deployer${TAG}"  --platform="linux/amd64" -f config/helm/Dockerfile config/helm; then
  docker push "$REGISTRY/$APP_NAME/deployer${TAG}"
fi
