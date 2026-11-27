#!/usr/bin/env bash

set -eu

export REGISTRY=gcr.io/dynatrace-marketplace-dev
export APP_NAME=dynatrace-operator
TAG="${1:-""}"
INSTALL_SOURCE="${2:-"google-marketplace"}"

if ! kubectl create namespace test-ns; then
  kubectl delete namespace test-ns
  kubectl create namespace test-ns
fi

mpdev install \
  --deployer="$REGISTRY/$APP_NAME/deployer${TAG}" \
  --parameters="{\"name\": \"test-deployment\", \"namespace\": \"test-ns\", \"installSource\": \"${INSTALL_SOURCE}\"}"
