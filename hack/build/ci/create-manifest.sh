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

platforms=($(echo "${raw_platforms}" | tr "," "\n"))

echo "Creating manifest for ${platforms[*]}"

images=()

for platfrom in "${platforms[@]}"
do
   docker pull "${image}-${platfrom}"
   images+=("${image}-${platfrom}")
done

docker manifest create "${image}"

if [ -z "${annotation}" ]
then
  docker manifest add "${image}" "${images[@]}"
else
  docker manifest add --annotation "${annotation}" "${image}" "${images[@]}"
fi

docker manifest inspect "${image}"

docker manifest push --format oci --digestfile=digestfile.sha256 "${image}"

sha256=$(cat digestfile.sha256)

echo "digest=${sha256}">> $GITHUB_OUTPUT
