#!/usr/bin/env bash

set -eu -o pipefail

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
