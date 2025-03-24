#!/bin/bash

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

platforms=($(echo "${raw_platforms}" | tr "," "\n"))

echo "Creating manifest for ${platforms[*]}"

images=()

images=()

for platfrom in "${platforms[@]}"
do
   podman pull "${image}-${platfrom}"
   images+=("${image}-${platfrom}")
done

podman manifest create "${image}"

if [ -z "${annotation}" ]
then
  podman manifest add "${image}" "${images[@]}"
else
  podman manifest add --annotation "${annotation}" "${image}" "${images[@]}"
fi

podman manifest inspect "${image}"

podman manifest push --format oci --digestfile=digestfile.sha256 "${image}"

sha256=$(cat digestfile.sha256)

echo "digest=${sha256}">> $GITHUB_OUTPUT
