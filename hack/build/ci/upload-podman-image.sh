#!/bin/bash

set -x

readonly platform="${1}"
target_image="${2}"
readonly skip_platform_suffix="${3}"

readonly image_tar_path="/tmp/operator-${platform}.tar"

if [ -z "${skip_platform_suffix}" ]
then
  target_image=${target_image}-${platform}
fi

src_image=$(podman load -i "${image_tar_path}" | cut -d' ' -f3)

# we need retag it because we load it as localhost
podman tag "${src_image}" "${target_image}"

# --format is manifest type (oci, v2s2, or v2s1) to use when pushing an image.
podman push --format oci "${target_image}"
