#!/usr/bin/env bash

set -x

DEFAULT_TIMEOUT="60m" # K8s takes <10min, Openshift >40min

kubectl version

echo "Creating environment '$ENVIRONMENT' in namespace '$NAMESPACE'"

kubectl get flcenvironments --namespace $NAMESPACE

echo "Patching environment '$ENVIRONMENT' to 'deployed'"
kubectl patch --namespace $NAMESPACE --type merge --patch '{"spec": {"desiredState": "environment-deployed"}}' flcenvironment $ENVIRONMENT

echo "Waiting up to '$DEFAULT_TIMEOUT' for successful deployment of environment '$ENVIRONMENT'"
kubectl wait --namespace $NAMESPACE --timeout="$DEFAULT_TIMEOUT" --for=condition=InTransition=false flcenvironment $ENVIRONMENT
