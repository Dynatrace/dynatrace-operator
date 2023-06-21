#!/bin/bash

digest=$(skopeo inspect docker://"${IMAGE}" --format "{{.Digest}}")
echo "digest=${digest}">> "$GITHUB_OUTPUT"
