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

echo "This script is based on podman version 5.4.1"
echo "current version of podman is $(/usr/local/bin/podman --version)"

platforms=($(echo "${raw_platforms}" | tr "," "\n"))

echo "Creating manifest for ${platforms[*]}"

images=()

for platfrom in "${platforms[@]}"
do
   /usr/local/bin/podman pull "${image}-${platfrom}"
   images+=("${image}-${platfrom}")
done

/usr/local/bin/podman manifest create --annotation "${annotation}" "${image}" "${images[@]}"

if [ -z "${annotation}" ]
then
 /usr/local/bin/podman manifest create "${image}" "${images[@]}"
else
  /usr/local/bin/podman manifest create --annotation "${annotation}" "${image}" "${images[@]}"
fi

/usr/local/bin/podman manifest inspect "${image}"

/usr/local/bin/podman manifest push --format oci --digestfile=digestfile.sha256 "${image}"

sha256=$(cat digestfile.sha256)

echo "digest=${sha256}">> $GITHUB_OUTPUT
