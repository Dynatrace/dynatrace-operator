#!/usr/bin/env bash

set -eu -o pipefail

ENVIRONMENT_KUBECONFIG="$RUNNER_TEMP/environment-kubeconfig"
ENVIRONMENT_SECRET_NAME="$ENVIRONMENT-kubeconfig"

echo "Loading kubeconfig from secret for environment '$ENVIRONMENT' from namespace '$NAMESPACE'"
kubectl get secret --namespace "$NAMESPACE" "$ENVIRONMENT_SECRET_NAME" -o jsonpath='{.data.kubeconfig}' | base64 --decode > "$ENVIRONMENT_KUBECONFIG"

echo "Switching to test cluster for environment '$ENVIRONMENT'!"
export KUBECONFIG="$ENVIRONMENT_KUBECONFIG"

kubectl version
kubectl config view

mkdir -p test/testdata/secrets/

pushd test/testdata/secrets/

cat << EOF > single-tenant.yaml
tenantUid: $TENANT1_NAME
apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
apiToken: $TENANT1_APITOKEN
EOF

cat << EOF > multi-tenant.yaml
tenants:
  - tenantUid: $TENANT1_NAME
    apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT1_APITOKEN
  - tenantUid: $TENANT2_NAME
    apiUrl: https://$TENANT2_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT2_APITOKEN
EOF

cat << EOF > edgeconnect-tenant.yaml
name: e2e-test
tenantUid: $TENANT1_NAME
apiServer: $TENANT1_NAME.dev.apps.dynatracelabs.com/api
oAuthClientId: $TENANT1_OAUTH_CLIENT_ID
oAuthClientSecret: $TENANT1_OAUTH_SECRET
EOF

cat << EOF > otel-tenant.yaml
endpoint: $TENANT1_NAME.dev.dynatracelabs.com
apiToken: $TENANT1_OTELTOKEN
EOF

popd

echo "Running tests for environment '$ENVIRONMENT' ..."
make BRANCH="${GITHUB_REF##*/}" test/e2e # use current branch image to run tests

echo "Test run completed!"
