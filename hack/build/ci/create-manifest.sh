#!/bin/bash

if [ -z "$3" ]
then
  echo "Usage: $0 <image_name> <image_tag> <platforms>"
  exit 1
fi

image_name=$1
image_tag=$2
raw_platforms=$3

image="${image_name}:${image_tag}"

platforms=($(echo "${raw_platforms}" | tr "," "\n"))

echo "Creating manifest for ${platforms[*]}"

images=()

for platfrom in "${platforms[@]}"
do
   docker pull "${image}-${platfrom}"
   images+=("${image}-${platfrom}")
done

docker manifest create "${image}" "${images[@]}"
docker manifest inspect "${image}"

sha256=$(docker manifest push "${image}")
echo "digest=${sha256}">> $GITHUB_OUTPUT
