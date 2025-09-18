#!/bin/bash

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <image_name> [architectures (comma-separated, default: amd64,arm64)]"
  exit 1
fi

image=$1
arch=${2:-"amd64,arm64"}

IFS=',' read -ra supported_architectures <<< "$arch"
images=()
echo "Creating image-index manifest for ${supported_architectures[*]}"

if ! command -v docker 2>/dev/null; then
  CONTAINER_CMD=podman
else
  CONTAINER_CMD=docker
fi

for architecture in "${supported_architectures[@]}"; do
  ${CONTAINER_CMD} pull "${image}-${architecture}"
  images+=("${image}-${architecture}")
done

${CONTAINER_CMD} manifest rm "${image}" 2>/dev/null || true
${CONTAINER_CMD} manifest create "${image}" "${images[@]}"

sha256=$(${CONTAINER_CMD} manifest push "${image}")
if [ "$GITHUB_OUTPUT" ]; then
  echo "Pushed image index to ${image} with digest ${sha256}"
  echo "digest=${sha256}" >> "$GITHUB_OUTPUT"
else
  echo "Image index created locally with digest ${sha256}"
fi
