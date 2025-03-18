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

# $podman load -i /tmp/alpine.tar
# Loaded image: alpine:latest
#
# we're interested in "alpine:latest", that's field=3
src_image=$(podman load -i "${image_tar_path}" | cut -d' ' -f3)

podman tag "${src_image}" "${target_image}"
pushinfo=$(podman push "${target_image}")

# filtering by image-tag directly does not work currently see: https://github.com/moby/moby/issues/29901
digest=$(echo "$pushinfo" | tail -n 1 | cut -d " " -f 3)
echo "digest=${digest}">> "$GITHUB_OUTPUT"
