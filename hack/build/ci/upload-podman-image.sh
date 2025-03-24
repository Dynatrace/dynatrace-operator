#!/bin/bash

set -x

target_image="${1}"
readonly image_tar_path="/tmp/operator-all-platforms.tar"

src_image=$(podman load -i "${image_tar_path}" | cut -d' ' -f3)

# we need retag it because we load it as localhost
podman tag "${src_image}" "${target_image}"

# --format, -f=format
# Manifest Type (oci, v2s2, or v2s1) to use when pushing an image.
podman push --format oci "${target_image}"

# TODO: add digest later
# echo "digest=${digest}">> "$GITHUB_OUTPUT"
