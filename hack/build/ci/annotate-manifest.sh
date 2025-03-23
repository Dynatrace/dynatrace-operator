#!/bin/bash

set -x # # activate debugging

if [ -z "$2" ]
then
  echo "Usage: $0 <image_name> <annotation>"
  exit 1
fi

image=${1}
annotation=${2}

podman pull $image

podman manifest inspect $image | jq


for digest in ${digest}
do
  echo digest
  # podman manifest annotate --annotation $annotation $image $digest
done

