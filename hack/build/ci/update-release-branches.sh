#!/bin/bash

set -o errexit
set -o pipefail
if [[ ${RUNNER_DEBUG-} == "true" ]]; then
    set -o xtrace
fi

NEW_VERSION=${1-$(git branch -r --list 'origin/release-*' | sort --version-sort | tail -n 1 | cut -d/ -f2)}
if grep -q "$NEW_VERSION" .github/renovate.json5; then
    echo "$NEW_VERSION release branch already present"
    exit 0
fi

NUM_VERSIONS=3
RENOVATE_FILE=.github/renovate.json5
WORKFLOW_FILE=.github/workflows/e2e-tests-ondemand.yaml

# Fetch list of versions by looking for the "baseBranches" key and reading the following $NUM_VERSIONS+1 lines.
# Trim the first line to get rid of the "$default".
# Clean up the list by removing all spaces, commas and quotes.
VERSION_LIST=$(grep -A $((NUM_VERSIONS+1)) baseBranches $RENOVATE_FILE | tail -n+$NUM_VERSIONS | tr -d ' ",')
if (( NUM_VERSIONS != "$(wc -l <<< "$VERSION_LIST")" )); then
    printf "unexpected list of versions:\n%s\n" "$VERSION_LIST" >&2
    exit 1
fi

sed=sed
if command -v gsed &>/dev/null; then
    sed=gsed
fi

# Replace versions bottom up. New version replaces the last from the current list
# Then we shorten the list by one and assign the new version to the removed element
# Repeat this until we iterated over all expected versions
while ((NUM_VERSIONS > 0)); do
    $sed -i "s/$(tail -1 <<< "$VERSION_LIST")/$NEW_VERSION/g" $RENOVATE_FILE
    # Repurpose the replacement for the workflow file
    $sed -i "s/$(tail -1 <<< "$VERSION_LIST")/$NEW_VERSION/g" $WORKFLOW_FILE
    NEW_VERSION=$(tail -1 <<< "$VERSION_LIST")
    NUM_VERSIONS=$((NUM_VERSIONS-1))
    VERSION_LIST=$(head -n+$NUM_VERSIONS <<<"$VERSION_LIST")
done
