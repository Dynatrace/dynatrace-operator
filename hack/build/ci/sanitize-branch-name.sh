#!/usr/bin/env bash

readonly BRANCH_NAME=${1}

if [ -z "${BRANCH_NAME}" ]; then
  echo "Usage: $0 <branch name>"
  exit 1
fi

echo "${BRANCH_NAME//[^a-zA-Z0-9_-]/-}"
