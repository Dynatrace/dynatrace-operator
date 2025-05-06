#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 <image_name>"
  exit 1
fi

image=$1

supported_architectures=("amd64" "arm64")
images=()
echo "Creating image-index manifest for ${supported_architectures[*]}"

for architecture in "${supported_architectures[@]}"; do
  docker pull "${image}-${architecture}"
  images+=("${image}-${architecture}")
done
docker manifest create "${image}" "${images[@]}"

sha256=$(docker manifest push "${image}")
echo "digest=${sha256}" >>$GITHUB_OUTPUT
