#!/bin/bash

set -x

readonly snyk_org_id=${1}
readonly snyk_int_id=${2}
readonly snyk_api_token=${3}
readonly image=${4}

response=$(curl --include \
     --request POST \
     --header "Content-Type: application/json; charset=utf-8" \
     --header "Authorization: token ${snyk_api_token}" \
     --data-binary "{
  \"target\": {
    \"name\": \"${image}\"
  }
}" \
"https://api.snyk.io/v1/org/${snyk_org_id}/integrations/${snyk_int_id}/import")

echo "$response"

rc=$(echo "$response" | grep HTTP/ | cut -d ' ' -f2)

if [ "$rc" != "201" ]
then
  exit 1
fi
