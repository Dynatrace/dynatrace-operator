#!/bin/bash

# This script cleans up a maximum of 500 Kubernetes settings found on the tenant 
# by filtering by schemaId cloud.kubernetes and deleting them via Dynatrace API.

if [[ -z "$TENANT_NAME" ]]; then
    echo "TENANT_NAME env var is undefined" 1>&2
    exit 1
fi
if [[ -z "$TENANT_APITOKEN" ]]; then
    echo "TENANT_APITOKEN env var is undefined" 1>&2
    exit 1
fi

response_file="response.json"

echo "Getting Kubernetes settings from $TENANT_NAME tenant..."

curl -s -X 'GET' \
  "https://$TENANT_NAME.dev.dynatracelabs.com/api/v2/settings/objects?schemaIds=builtin%3Acloud.kubernetes&fields=objectId&pageSize=500&adminAccess=false" \
  -H 'accept: application/json; charset=utf-8' \
  -H "Authorization: Api-Token $TENANT_APITOKEN" > "$response_file"

echo "Total objects found: $(jq -r '.totalCount' "$response_file")"

object_ids=$(jq -r '.items[].objectId' "$response_file")

for object_id in $object_ids; do
  echo "Deleting objectId $object_id"
  curl -s -X 'DELETE' \
  "https://$TENANT_NAME.dev.dynatracelabs.com/api/v2/settings/objects/$object_id?adminAccess=false" \
  -H 'accept: */*' \
  -H "Authorization: Api-Token $TENANT_APITOKEN"
done

curl -s -X 'GET' \
  "https://$TENANT_NAME.dev.dynatracelabs.com/api/v2/settings/objects?schemaIds=builtin%3Acloud.kubernetes&fields=objectId&adminAccess=false" \
  -H 'accept: application/json; charset=utf-8' \
  -H "Authorization: Api-Token $TENANT_APITOKEN" > "$response_file"

echo "Total objects left after cleanup: $(jq -r '.totalCount' "$response_file")"

rm -f "$response_file"
echo "Response file cleaned up."
