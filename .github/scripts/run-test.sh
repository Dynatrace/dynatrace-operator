#!/usr/bin/env bash

ENVIRONMENT_KUBECONFIG="$RUNNER_TEMP/environment-kubeconfig"
ENVIRONMENT_SECRET_NAME="$ENVIRONMENT-kubeconfig"

echo "Loading kubeconfig from secret for environment '$ENVIRONMENT' from namespace '$NAMESPACE'"
kubectl get secret --namespace $NAMESPACE
kubectl get secret --namespace $NAMESPACE "$ENVIRONMENT_SECRET_NAME" -o jsonpath='{.data.kubeconfig}' | base64 --decode > "$ENVIRONMENT_KUBECONFIG"

echo "Switching to test cluster for environment '$ENVIRONMENT'!"
export KUBECONFIG_SAVED="$KUBECONFIG"
export KUBECONFIG="$ENVIRONMENT_KUBECONFIG"

kubectl version
kubectl config view

echo "Running tests for environment '$ENVIRONMENT' ..."
kubectl get all -A

echo "Test run completed!"
export KUBECONFIG="$KUBECONFIG_SAVED"
