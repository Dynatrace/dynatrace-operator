#!/usr/bin/env bash

set -eu -o pipefail

echo "Switching to target branch directory..."
cd target

FLC_ENVIRONMENT_KUBECONFIG="$RUNNER_TEMP/environment-kubeconfig"
FLC_ENVIRONMENT_SECRET_NAME="$FLC_ENVIRONMENT-kubeconfig"

echo "Loading kubeconfig from secret for environment '$FLC_ENVIRONMENT' from namespace '$FLC_NAMESPACE'"
kubectl get secret --namespace "$FLC_NAMESPACE" "$FLC_ENVIRONMENT_SECRET_NAME" -o jsonpath='{.data.kubeconfig}' | base64 --decode > "$FLC_ENVIRONMENT_KUBECONFIG"

echo "Switching to test cluster for environment '$FLC_ENVIRONMENT'!"
export KUBECONFIG="$FLC_ENVIRONMENT_KUBECONFIG"

echo "Preparing test tenant secrets..."
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
apiServer: $TENANT1_NAME.dev.apps.dynatracelabs.com
oAuthClientId: $TENANT1_OAUTH_CLIENT_ID
oAuthClientSecret: $TENANT1_OAUTH_SECRET
EOF

cat << EOF > otel-tenant.yaml
endpoint: $TENANT1_NAME.dev.dynatracelabs.com
apiToken: $TENANT1_OTELTOKEN
EOF

popd

echo "Running tests for environment '$FLC_ENVIRONMENT'..."
make BRANCH="$TARGET_BRANCH" test/e2e

echo "Success!"
