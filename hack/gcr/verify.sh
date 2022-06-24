#!/usr/bin/env bash

set -eu

export REGISTRY=gcr.io/dynatrace-marketplace-dev
export APP_NAME=dynatrace-operator
export VERSION=0.7.0

make images/push TAG="$VERSION" IMG="$REGISTRY/$APP_NAME"

kubectl apply -f "https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml"

if docker build --tag "$REGISTRY/$APP_NAME/deployer" -f config/helm/Dockerfile config/helm; then
  docker push "$REGISTRY/$APP_NAME/deployer"
fi

mpdev verify \
  --deployer=$REGISTRY/$APP_NAME/deployer \
  --parameters='{"name": "test-deployment", "namespace": "test-ns"}'
