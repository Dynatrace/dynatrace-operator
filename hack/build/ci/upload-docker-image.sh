#!/bin/bash

set -x

if [ -z "$2" ]
then
  echo "Usage: $0 <platform> <targetImageTag>"
  exit 1
fi

readonly platform=${1}
readonly targetImageTag=${2}
readonly imageTarPath="/tmp/operator-${platform}.tar"

docker load -i "${imageTarPath}"

# $docker load -i /tmp/alpine.tar
# Loaded image: alpine:latest
#
# we're interested in "alpine:latest", that's field=3
srcImageTag=$(docker load -i "${imageTarPath}" | cut -d' ' -f3)

docker load --input "${imageTarPath}"
docker tag "${srcImageTag}" "${targetImageTag}"
docker push "${targetImageTag}"
