#!/usr/bin/env bash

set -eu -o pipefail

vault kv get -field kubeconfig "secret/pipelines/${ENVIRONMENT_NAME}/kubeconfig" > "${ENVIRONMENT_NAME}.kubeconfig"

KUBECONFIG="$(pwd)/${ENVIRONMENT_NAME}.kubeconfig"
export KUBECONFIG

pushd dynatrace-operator-repo

mkdir -p test/testdata/secrets/

pushd test/testdata/secrets/

cat << EOF > single-tenant.yaml
tenantUid: $DYNATRACE_TENANT_1
apiUrl: $DYNATRACE_API_URL_1
apiToken: $DYNATRACE_API_TOKEN_1
EOF

cat << EOF > multi-tenant.yaml
tenants:
  - tenantUid: $DYNATRACE_TENANT_1
    apiUrl: $DYNATRACE_API_URL_1
    apiToken: $DYNATRACE_API_TOKEN_1
  - tenantUid: $DYNATRACE_TENANT_2
    apiUrl: $DYNATRACE_API_URL_2
    apiToken: $DYNATRACE_API_TOKEN_2
EOF

cat << EOF > edgeconnect-tenant.yaml
name: $DYNATRACE_EDGECONNECT_NAME
tenantUid: $DYNATRACE_EDGECONNECT_TENANT
apiServer: $DYNATRACE_EDGECONNECT_API_SERVER
oAuthClientId: $DYNATRACE_EDGECONNECT_OAUTH_CLIENT_ID
oAuthClientSecret: $DYNATRACE_EDGECONNECT_OAUTH_SECRET
EOF

cat << EOF > otel-tenant.yaml
endpoint: $DYNATRACE_OTEL_ENDPOINT
apiToken: $DYNATRACE_OTEL_API_TOKEN
EOF

popd

make BRANCH="${BRANCH}" test/e2e/cloudnative/proxy
# make BRANCH="${BRANCH}" test/e2e/standard
# make BRANCH="${BRANCH}" test/e2e/istio
# make BRANCH="${BRANCH}" test/e2e/release
