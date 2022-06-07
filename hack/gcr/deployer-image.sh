#!/usr/bin/env bash

export REGISTRY=gcr.io/dynatrace-marketplace-dev
export APP_NAME=dynatrace-operator

if docker build --tag "$REGISTRY/$APP_NAME/deployer" -f config/helm/Dockerfile config/helm; then
  docker push "$REGISTRY/$APP_NAME/deployer"
fi
