#!/bin/bash
# Usage: check-image-available.sh <image> <tag> [timeout] [interval]

set -eu

IMAGE="${1:?Image name required (e.g. dynatrace/dynatrace-operator)}"
TAG="${2:?Tag required}"
TIMEOUT="${3:-600}"
INTERVAL="${4:-30}"
ELAPSED=0

echo "Checking if image ${IMAGE}:${TAG} is available on ghcr.io"

token=$(curl -s https://ghcr.io/token\?scope\="repository:${IMAGE}:pull" | jq -r .token)
lookup_image() {
    curl -s --fail \
        -H "Authorization: Bearer ${token}" \
        -H 'Accept: application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
        "https://ghcr.io/v2/${IMAGE}/manifests/${TAG}" &>/dev/null
}

while ! lookup_image; do
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo "Timeout reached. Image does not exist."
    exit 1
  fi
  echo "Image not available yet. Waiting... ($ELAPSED/$TIMEOUT s)"
  sleep "$INTERVAL"
  ELAPSED=$((ELAPSED+INTERVAL))
done
echo "Image exists"
