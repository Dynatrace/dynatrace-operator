#!/usr/bin/env bash

set -eu -o pipefail

echo "Switching to target branch directory..."
cd target

FLC_ENVIRONMENT_KUBECONFIG="$RUNNER_TEMP/environment-kubeconfig"
FLC_ENVIRONMENT_SECRET_NAME="$FLC_ENVIRONMENT-kubeconfig"

echo "Wait 30s for secret is created '$FLC_ENVIRONMENT_SECRET_NAME'"
kubectl wait --timeout=30s --for=create secret --namespace "$FLC_NAMESPACE" "$FLC_ENVIRONMENT_SECRET_NAME"

echo "Loading kubeconfig from secret for environment '$FLC_ENVIRONMENT' from namespace '$FLC_NAMESPACE'"
kubectl get secret --namespace "$FLC_NAMESPACE" "$FLC_ENVIRONMENT_SECRET_NAME" -o jsonpath='{.data.kubeconfig}' | base64 --decode > "$FLC_ENVIRONMENT_KUBECONFIG"

echo "Switching to test cluster for environment '$FLC_ENVIRONMENT'!"
export KUBECONFIG="$FLC_ENVIRONMENT_KUBECONFIG"

echo "Exporting env var containing helm chart used for installation (if provided)"
export HELM_CHART

echo "Preparing test tenant secrets..."

pwd

bash ./.github/scripts/prepare-e2e-secrets.sh

echo "Running tests for environment '$FLC_ENVIRONMENT'..."

if [[ -z "${TARGET_IMAGE}" ]]; then
  make IMG="$TARGET_IMAGE" test/e2e-publish
else
  echo "fall back to default branch target"
  make BRANCH="$TARGET_BRANCH" test/e2e-publish
fi

echo "Success!"
