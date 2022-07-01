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
srcImageTag=$(docker load -i "${imageTarPath}" | cut -d' ' -f3)

docker load --input "${imageTarPath}"
docker tag "${srcImageTag}" "${targetImageTag}"
docker push "${targetImageTag}"
