#!/usr/bin/env bash

set -x

echo "Destroying environment '$ENVIRONMENT' in namespace '$NAMESPACE'"

kubectl get flcenvironments --namespace $NAMESPACE

echo "Patching environment '$ENVIRONMENT' to 'not-deployed'"
kubectl patch --namespace $NAMESPACE --type merge --patch '{"spec": {"desiredState": "environment-not-deployed"}}' flcenvironment $ENVIRONMENT
