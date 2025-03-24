#!/bin/bash

set -x # # activate debugging

if [ -z "$3" ]
then
  echo "Usage: $0 <image_name> <raw_platforms> <annotation>"
  echo "<raw_platforms> single platform or comma separated platforms in buildah format"
  echo "<annotation> should be key=value string"
  exit 1
fi

image=$1
raw_platforms=$2
annotation=$3

buildah manifest inspect ${image}

# Used for testing
# raw_platforms="linux/amd64"
# raw_platforms="linux/amd64, linux/arm64"
# raw_platforms="linux/amd64,linux/arm64"

# we want same format as buildah doing internally during building multi-arch image
# which is `platformarch` without slashes and spaces
platforms=($(echo ${raw_platforms//\//} | tr "," "\n"))
# debug array
echo "${platforms[@]}"

for platform in "${platforms[@]}"
do
  echo "$platform"
  buildah manifest annotate --annotation $annotation "${image}" "${image}-${platform}"
done

buildah manifest inspect ${image}
