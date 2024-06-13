#!/bin/bash

set -x

readonly platform="${1}"
targetImage="${2}"
readonly skip_platform_suffix="${3}"

readonly imageTarPath="/tmp/operator-${platform}.tar"

if [ -z "${skip_platform_suffix}" ]
then
  targetImage=${targetImage}-${platform}
fi

docker load -i "${imageTarPath}"

# $docker load -i /tmp/alpine.tar
# Loaded image: alpine:latest
#
# we're interested in "alpine:latest", that's field=3
srcImage=$(docker load -i "${imageTarPath}" | cut -d' ' -f3)

docker load --input "${imageTarPath}"
docker tag "${srcImage}" "${targetImage}"
pushinfo=$(docker push "${targetImage}")

# filtering by image-tag directly does not work currently see: https://github.com/moby/moby/issues/29901
digest=$(echo "$pushinfo" | tail -n 1 | cut -d " " -f 3)
echo "digest=${digest}">> "$GITHUB_OUTPUT"
