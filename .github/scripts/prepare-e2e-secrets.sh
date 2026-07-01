#!/usr/bin/env bash

set -eu -o pipefail

for dir in test/testdata/secrets test/e2e/testdata/secrets ; do
  mkdir -p ${dir}

  pushd ${dir}

  cat << EOF > single-tenant.yaml
tenantUid: $TENANT1_NAME
apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
apiToken: $TENANT1_APITOKEN
apiTokenNoSettings: $TENANT1_APITOKEN_NOSETTINGS
dataIngestToken: $TENANT1_DATAINGESTTOKEN
platformToken: $TENANT1_PLATFORM_TOKEN
platformTokenNoSettings: $TENANT1_PLATFORM_TOKEN_NOSETTINGS
dataIngestPlatformToken: $TENANT1_DATAINGEST_PLATFORM_TOKEN
EOF


  cat << EOF > phase3-tenant.yaml
tenantUid: $TENANT3_PHASE3_NAME
apiUrl: https://$TENANT3_PHASE3_NAME.dev.dynatracelabs.com/api
apiToken: $TENANT3_PHASE3_APITOKEN
dataIngestToken: $TENANT3_PHASE3_DATAINGESTTOKEN
platformToken: $TENANT3_PHASE3_PLATFORM_TOKEN
platformTokenNoSettings: $TENANT3_PHASE3_PLATFORM_TOKEN_NOSETTINGS
EOF

  cat << EOF > multi-tenant.yaml
tenants:
  - tenantUid: $TENANT1_NAME
    apiUrl: https://$TENANT1_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT1_APITOKEN
    dataIngestToken: $TENANT1_DATAINGESTTOKEN
    platformToken: $TENANT1_PLATFORM_TOKEN
  - tenantUid: $TENANT2_NAME
    apiUrl: https://$TENANT2_NAME.dev.dynatracelabs.com/api
    apiToken: $TENANT2_APITOKEN
    dataIngestToken: $TENANT2_DATAINGESTTOKEN
    platformToken: $TENANT2_PLATFORM_TOKEN
EOF

  cat << EOF > edgeconnect-tenant.yaml
name: e2e-test
tenantUid: $TENANT1_NAME
apiServer: $TENANT1_NAME.dev.apps.dynatracelabs.com
oAuthClientId: $TENANT1_OAUTH_CLIENT_ID
oAuthClientSecret: $TENANT1_OAUTH_SECRET
resource: $TENANT1_OAUTH_URN
EOF

  cat << EOF > edgeconnect-phase3-tenant.yaml
name: e2e-test-phase3
tenantUid: $TENANT3_PHASE3_NAME
apiServer: $TENANT3_PHASE3_NAME.dev.apps.dynatracelabs.com
oAuthClientId: $TENANT3_PHASE3_OAUTH_CLIENT_ID
oAuthClientSecret: $TENANT3_PHASE3_OAUTH_SECRET
resource: $TENANT3_PHASE3_OAUTH_URN
EOF

  popd
done
