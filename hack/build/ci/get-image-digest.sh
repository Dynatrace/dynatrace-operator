#!/bin/bash

digest=$(docker image list --digests ${IMAGE} --format json | jq -r '.[].Digest')
echo "digest=${digest}">> "$GITHUB_OUTPUT"
