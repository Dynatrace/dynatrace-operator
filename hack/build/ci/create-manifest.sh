#!/bin/bash

if [ -z "$2" ]
then
  echo "Usage: $0 <image_name> <image_tag> <enable_multiplatform>"
  exit 1
fi

image_name=$1
image_tag=$2
multiplatform=$3
image="${image_name}:${image_tag}"

if [ "$multiplatform" == "true" ]
then
  supported_architectures=("amd64" "arm64" "ppc64le" "s390x")
  images=()
  echo "Creating manifest for ${supported_architectures[*]}"

  for architecture in "${supported_architectures[@]}"
  do
    podman pull "${image}-${architecture}"
    images+=("${image}-${architecture}")
  done
  podman manifest create "${image}" "${images[@]}"
else
  echo "Creating manifest for the AMD image "
  podman pull "${image}-amd64"
  podman manifest create "${image}" "${image}-amd64"
fi

sha256=$(podman manifest push "${image}")
echo "digest=${sha256}">> $GITHUB_OUTPUT
