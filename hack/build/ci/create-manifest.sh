#!/bin/bash

set -x # # activate debugging

if [ -z "$2" ]
then
  echo "Usage: $0 <image_name> <image_tag> <enable_multiplatform>"
  exit 1
fi

image_name=$1
image_tag=$2
multiplatform=$3
image="${image_name}:${image_tag}"

echo "This script is based on podman version 4.9.3"
echo "current version of podman is $(podman --version)"

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

  podman manifest create "${image}"

  podman manifest add --annotation "andrii=test" "${image}" "${images[@]}"

#  if [[ "$image" =~ gcr.io ]]
#  then
#    podman manifest create --annotation "com.googleapis.cloudmarketplace.product.service.name=services/dynatrace-operator-dynatrace-marketplace-prod.cloudpartnerservices.goog" "${image}" "${images[@]}"
#  fi

else
  echo "Creating manifest for the AMD image "

  podman pull "${image}-amd64"

  podman manifest create "${image}"

  podman manifest add --annotation "andrii=test" "${image}" "${image}-amd64"

fi

podman manifest inspect "${image}

sha256=$(podman manifest push "${image}")

echo "digest=${sha256}">> $GITHUB_OUTPUT
