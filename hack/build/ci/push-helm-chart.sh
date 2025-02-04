#!/bin/bash

readonly PATH_TO_HELM_CHART="${1}"
readonly REGISTRY_URL="${2}"

output=$(helm push "${PATH_TO_HELM_CHART}" "${REGISTRY_URL}" 2>&1)
exit_status=$?

if [ $exit_status -eq 0 ]; then
  digest=$(echo "$output" | awk '/Digest:/ {print $2}')
  echo "digest=$digest" >> $GITHUB_OUTPUT
else
  echo "Command failed with exit status $exit_status. Error: $output"
  exit $exit_status
fi
