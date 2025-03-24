#!/bin/bash

set -euxo pipefail

if [ -z "$3" ]
then
  echo "Usage: $0 <image_name> <image_tag> <platforms> <annotation>"
  exit 1
fi

image_name=$1
image_tag=$2
raw_platforms=$3
annotation=$4

image="${image_name}:${image_tag}"

echo "This script is based on podman version 4.9.3"
echo "current version of podman is $(podman --version)"

platforms=($(echo ${raw_platforms} | tr "," "\n"))

echo "Creating manifest for ${platforms[@]}"

images=()

for platfrom in "${platforms[@]}"
do
   echo "$platform"
   podman pull "${image}-${platfrom}"
   images+=("${image}-${platfrom}")
done

podman manifest create "${image}"

podman manifest add --annotation "andrii=test" "${image}" "${images[@]}"

podman manifest inspect "${image}"

podman manifest push --format oci "${image}"

podman manifest inspect "${image}"
