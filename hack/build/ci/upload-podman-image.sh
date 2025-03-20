#!/bin/bash

set -x

target_image="${1}"
readonly image_tar_path="/tmp/operator-all-platforms.tar"

podman load -i "${image_tar_path}"

podman tag "${src_image}" "${target_image}"

podman push "${target_image}"

podman manifest inspect "${target_image}"

# TODO: add digest later
# echo "digest=${digest}">> "$GITHUB_OUTPUT"
