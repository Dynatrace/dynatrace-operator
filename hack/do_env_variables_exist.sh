#!/usr/bin/env bash

# check if parameters are set
if [ -z "$1" ]
then
  echo "Usage: $0 <env variables to check>"
  exit 1
fi

# make sure all fields are set
for field in $1; do
  if [ -z "${!field}" ]; then
    echo "Error: $field is not set"
    exit 1
  fi
done
