#!/bin/bash

digest=$(skopeo inspect docker-daemon:"${IMAGE}" --format "{{.Digest}}")
echo "${DIGEST_KEY}=${digest}">> "$GITHUB_OUTPUT"

