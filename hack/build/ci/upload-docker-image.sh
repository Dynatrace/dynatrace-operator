#!/bin/bash

if [ -z "$3" ]
then
  echo "Usage: $0 <platform> <source_image> <target_image>"
  exit 1
fi

platform=$1
source_image=$2
target_image=$3

docker load --input "/tmp/operator-${platform}.tar"
docker tag "${source_image}" "${target_image}"
docker push "${target_image}"
