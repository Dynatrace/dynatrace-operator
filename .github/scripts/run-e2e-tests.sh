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

source ../ref/.github/scripts/prepare-e2e-secrets.sh

echo "Running tests for environment '$FLC_ENVIRONMENT'..."

if [[ $FLC_ENVIRONMENT =~ "olm" ]]; then
  echo "run no csi tests suite using OLM"
  make test/e2e/no-csi/publish/olm
elif [[ $FLC_ENVIRONMENT =~ "fips" ]]; then
  echo "run fips e2e test suites"
  make BRANCH="$TARGET_BRANCH" FIPS=true test/e2e-publish
else
  echo "fall back to default branch target"
  make BRANCH="$TARGET_BRANCH" test/e2e-publish
fi

# Permissions scenario: helm-chart RBAC validation. Run once on k8s-latest and ocp-latest.
# Excluded from FIPS (same RBAC, only the image differs) and OLM (non-helm install).
if [[ $FLC_ENVIRONMENT == "dto-k8s-latest-flc" || $FLC_ENVIRONMENT == "dto-ocp-latest-flc" ]]; then
  echo "run permissions e2e suite"
  make BRANCH="$TARGET_BRANCH" test/e2e/permissions/publish
fi

echo "Success!"
