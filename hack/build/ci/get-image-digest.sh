#!/bin/bash

digest=$(skopeo inspect docker://"${IMAGE}" --format "{{.Digest}}")
digest_formatted=$(echo ${digest} | tr : -)
echo "digest=${digest}">> "$GITHUB_OUTPUT"
echo "digest_formatted=${digest_formatted}">> "$GITHUB_OUTPUT"

