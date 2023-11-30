#!/bin/bash

readonly online="${1}"

if [ "$#" -ne 1 ];
then
  echo "Usage: $0 <fetch_from_registry>"
  exit 1
fi

if [ "$online" == "true" ];
then
  readonly PROVIDER="docker://"
else
  readonly PROVIDER="docker-daemon:"
fi

digest=$(skopeo inspect ${PROVIDER}"${IMAGE}" --format "{{.Digest}}")
echo "digest=${digest}">> "$GITHUB_OUTPUT"
