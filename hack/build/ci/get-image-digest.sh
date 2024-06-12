#!/bin/bash

digest=$(docker image list --format "{{.Digest}}" ${IMAGE})
echo "digest=${digest}">> "$GITHUB_OUTPUT"
