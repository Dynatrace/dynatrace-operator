#!/bin/bash

# This script cleans up all Kubernetes settings found on the tenant by filtering by schemaId cloud.kubernetes
# and deleting them via Dynatrace API.

response_file="response.json"

echo "Getting Kubernetes settings from $TENANT_NAME tenant."

curl -X 'GET' \
  "https://$TENANT_NAME.dev.dynatracelabs.com/api/v2/settings/objects?schemaIds=builtin%3Acloud.kubernetes&fields=objectId&pageSize=500&adminAccess=false" \
  -H 'accept: application/json; charset=utf-8' \
  -H "Authorization: Api-Token $TENANT_APITOKEN" > "$response_file"

object_ids=$(jq -r '.items[].objectId' "$response_file")

for object_id in $object_ids; do
  echo "Deleting objectId: $object_id"
  curl -X 'DELETE' \
  "https://$TENANT_NAME.dev.dynatracelabs.com/api/v2/settings/objects/$object_id?adminAccess=false" \
  -H 'accept: */*' \
  -H "Authorization: Api-Token $TENANT_APITOKEN"
done

echo "All Kubernetes settings have been cleaned up."

rm -f "$response_file"
echo "Response file cleaned up."
