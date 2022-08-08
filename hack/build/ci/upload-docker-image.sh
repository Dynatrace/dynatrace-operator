#!/bin/bash

set -x

if [ -z "$4" ]
then
  echo "Usage: $0 <platform> <registry> <repository> <version>"
  exit 1
fi

readonly platform=${1}
readonly registry=${2}
readonly repository=${3}
readonly version=${4}
readonly imageTarPath="/tmp/operator-${platform}.tar"

targetImageTag="${registry}/${repository}:${version}"
if [ "${registry}" != "scan.connect.redhat.com" ]
then
  targetImageTag=${targetImageTag}-${platform}
fi

docker load -i "${imageTarPath}"

# $docker load -i /tmp/alpine.tar
# Loaded image: alpine:latest
#
# we're interested in "alpine:latest", that's field=3
srcImageTag=$(docker load -i "${imageTarPath}" | cut -d' ' -f3)

docker load --input "${imageTarPath}"
docker tag "${srcImageTag}" "${targetImageTag}"
docker push "${targetImageTag}"
