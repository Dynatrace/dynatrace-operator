#!/usr/bin/env bash

readonly BRANCH_NAME=${1}

if [ -z "${BRANCH_NAME}" ]; then
  echo ""
  exit 0
fi

SANITIZED_BRANCH_NAME="${BRANCH_NAME//[^a-zA-Z0-9_.-]/-}"
echo "${SANITIZED_BRANCH_NAME:0:63}"
