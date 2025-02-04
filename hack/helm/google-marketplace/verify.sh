#!/usr/bin/env bash

set -eu

gcloud config set project dynatrace-marketplace-dev
# Set the registry to your project GCR repo.
export REGISTRY=gcr.io/$(gcloud config get-value project | tr ':' '/')
export APP_NAME=dynatrace-operator
export VERSION=test

docker build --tag $REGISTRY/$APP_NAME/deployer:$VERSION ./.. --no-cache
docker push $REGISTRY/$APP_NAME/deployer:$VERSION

kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/dynatrace.com_dynakubes.yaml
mpdev verify --deployer=$REGISTRY/$APP_NAME/deployer:$VERSION