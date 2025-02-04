#!/usr/bin/env bash

set -x

DEFAULT_TIMEOUT="20m"

echo "Destroying environment '$FLC_ENVIRONMENT' in namespace '$FLC_NAMESPACE'"

kubectl get flcenvironments --namespace "$FLC_NAMESPACE"

echo "Patching environment '$FLC_ENVIRONMENT' to 'not-deployed'"
kubectl patch --namespace "$FLC_NAMESPACE" --type merge --patch '{"spec": {"desiredState": "environment-not-deployed"}}' flcenvironment "$FLC_ENVIRONMENT"

kubectl wait --namespace "$FLC_NAMESPACE" --timeout="$DEFAULT_TIMEOUT" --for jsonpath='{.status.currentState}'=environment-not-deployed flcenvironment "$FLC_ENVIRONMENT"
