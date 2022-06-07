#!/usr/bin/env bash

export REGISTRY=gcr.io/dynatrace-marketplace-dev
export APP_NAME=dynatrace-operator

if ! kubectl create namespace test-ns; then
  kubectl delete namespace test-ns
  kubectl create namespace test-ns
fi

mpdev install \
  --deployer=$REGISTRY/$APP_NAME/deployer \
  --parameters='{"name": "test-deployment", "namespace": "test-ns"}'