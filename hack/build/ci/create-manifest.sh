#!/bin/bash

if [ -z "$2" ]
then
  echo "Usage: $0 <image_name> <image_tag> <enable_multiplatform>"
  exit 1
fi

image_name=$1
image_tag=$2
multiplatform=$3

if [ "$multiplatform" == "true" ]
then
  echo "Creating manifest for AMD and ARM images"
  docker manifest create "${image_name}:${image_tag}" "${image_name}:${image_tag}-arm64" "${image_name}:${image_tag}-amd64"
else
  echo "Creating manifest for the AMD image "
  docker manifest create "${image_name}:${image_tag}" "${image_name}:${image_tag}-amd64"
fi

docker manifest push "${image_name}:${image_tag}"
