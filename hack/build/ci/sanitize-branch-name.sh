#!/usr/bin/env bash

readonly BRANCH_NAME=${1}

if [ -z "${BRANCH_NAME}" ]; then
  echo ""
  exit 0
fi

echo "${BRANCH_NAME//[^a-zA-Z0-9_.-]/-}"
