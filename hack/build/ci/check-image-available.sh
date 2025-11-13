#!/bin/bash
# Usage: check-image-available.sh <image> <tag> [timeout] [interval]

set -eu

IMAGE="${1:?Image name required (e.g. dynatrace/dynatrace-operator)}"
TAG="${2:?Tag required}"
TIMEOUT="${3:-600}"
INTERVAL="${4:-10}"
ELAPSED=0

echo "Checking if image ${IMAGE}:${TAG} is available on quay.io"

while true; do
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" "https://quay.io/v2/${IMAGE}/manifests/${TAG}")
  if [ "$STATUS" -eq 200 ]; then
    echo "Image exists"
    exit 0
  fi
  if [ "$ELAPSED" -ge "$TIMEOUT" ]; then
    echo "Timeout reached. Image does not exist."
    exit 1
  fi
  echo "Image not available yet. Waiting... ($ELAPSED/$TIMEOUT s)"
  sleep "$INTERVAL"
  ELAPSED=$((ELAPSED+INTERVAL))
done
