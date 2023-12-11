#!/bin/bash

digest=$(skopeo inspect docker-daemon:"${IMAGE}" --format "{{.Digest}}")
echo "digest=${digest}">> "$GITHUB_OUTPUT"
