#!/bin/bash

if [[ ! "${1}" ]]; then
  echo "1 param is not set, should be the image without the tag"
  exit 5
fi
if [[ ! "${2}" ]]; then
  echo "2 param is not set, should be the tag of the image"
  exit 5
fi
IMAGE=${1}
TAG=${2}

out_image="${IMAGE}${TAG}"
is_image_present=$(docker images | grep "${out_image}")
if [ "${is_image_present}" != "" ]; then
   echo "image is not present, please build it first"
   exit 5
fi
docker push "${out_image}"
