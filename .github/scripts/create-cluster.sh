#!/usr/bin/env bash

set -x

DEFAULT_TIMEOUT="80m" # K8s takes <10min, Openshift >40min
DESIRED_STATE="environment-deployed"

kubectl version

echo "Creating environment '$FLC_ENVIRONMENT' in namespace '$FLC_NAMESPACE'"

kubectl get flcenvironments --namespace "$FLC_NAMESPACE"

echo "Patching environment '$FLC_ENVIRONMENT' to 'deployed'"
kubectl patch --namespace "$FLC_NAMESPACE" --type merge --patch '{"spec": {"desiredState": "environment-deployed"}}' flcenvironment "$FLC_ENVIRONMENT"

echo "Waiting up to '$DEFAULT_TIMEOUT' for successful deployment of environment '$FLC_ENVIRONMENT'"
kubectl wait --namespace "$FLC_NAMESPACE" --timeout="$DEFAULT_TIMEOUT" --for jsonpath='{.status.currentState}'="$DESIRED_STATE" flcenvironment "$FLC_ENVIRONMENT"
