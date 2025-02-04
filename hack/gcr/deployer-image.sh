#!/usr/bin/env bash

set -eu

export REGISTRY=gcr.io/dynatrace-marketplace-dev
export APP_NAME=dynatrace-operator
TAG="${1:-""}"

if docker build --tag "$REGISTRY/$APP_NAME/deployer${TAG}" -f config/helm/Dockerfile config/helm; then
  docker push "$REGISTRY/$APP_NAME/deployer${TAG}"
fi
