#!/usr/bin/env bash

set -e # instructs bash to immediately exit if any command has a non-zero exit status.
set -u # causes the bash shell to treat unset variables as an error and exit immediately.
set -o pipefail # causes a pipeline to return the exit status of the last command in the pipe that returned a non-zero return value.
set -x # instructs bash to print each command and its arguments to standard output as they are executed.

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
mkdir -p test/testdata/secrets/

pushd test/testdata/secrets/

cat << EOF > single-tenant.yaml
tenantUid: $TENANT1_NAME
apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
apiToken: $TENANT1_APITOKEN
apiTokenNoSettings: $TENANT1_APITOKEN_NOSETTINGS
dataIngestToken: $TENANT1_DATAINGESTTOKEN
EOF

cat << EOF > multi-tenant.yaml
tenants:
  - tenantUid: $TENANT1_NAME
    apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT1_APITOKEN
    dataIngestToken: $TENANT1_DATAINGESTTOKEN
  - tenantUid: $TENANT2_NAME
    apiUrl: https://$TENANT2_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT2_APITOKEN
    dataIngestToken: $TENANT2_DATAINGESTTOKEN
EOF

cat << EOF > edgeconnect-tenant.yaml
name: e2e-test
tenantUid: $TENANT1_NAME
apiServer: $TENANT1_NAME.dev.apps.dynatracelabs.com
oAuthClientId: $TENANT1_OAUTH_CLIENT_ID
oAuthClientSecret: $TENANT1_OAUTH_SECRET
resource: $TENANT1_OAUTH_URN
EOF

popd

echo "Running tests for environment '$FLC_ENVIRONMENT'..."

which gotestsum
which make

if [[ -z "${TARGET_IMAGE}" ]]; then
  make IMG="$TARGET_IMAGE" test/e2e-publish
else
  echo "fall back to default branch target"
  make BRANCH="$TARGET_BRANCH" test/e2e-publish
fi

echo "Success!"
