#!/bin/bash

if [[ ! "${1}" ]]; then
  echo "first param is not set, should be the image without the tag"
  exit 1
fi
if [[ ! "${2}" ]]; then
  echo "second param is not set, should be the tag of the image"
  exit 1
fi
image=${1}
tag=${2}

out_image="${image}:${tag}"

if ! command -v docker
  CONTAINER_CMD=docker
then
  CONTAINER_CMD=podman
fi

${CONTAINER_CMD} push "${out_image}"
