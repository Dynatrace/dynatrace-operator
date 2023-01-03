#!/bin/bash

set -x

readonly platform="${1}"
readonly targetImage="${2}"
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
docker push "${targetImage}"
