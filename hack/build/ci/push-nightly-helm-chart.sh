#!/bin/bash

readonly DEFAULT_CHART_CONFIG_FILE="config/helm/chart/default/Chart.yaml"

# `helm push` sets these annotations automagically, oras doesn't => copy the values from our Chart.yaml file
authors=$(yq '[.maintainers[] | .name + " (" + .email + ")"] | join(", ")' "${DEFAULT_CHART_CONFIG_FILE}")
description=$(yq .description "${DEFAULT_CHART_CONFIG_FILE}")
source=$(yq .sources[0] "${DEFAULT_CHART_CONFIG_FILE}")
title=$(yq .name "${DEFAULT_CHART_CONFIG_FILE}")
url=$(yq .home "${DEFAULT_CHART_CONFIG_FILE}")

yq ".appVersion=env(IMAGE_VERSION), .version=env(IMAGE_VERSION)" \
  "${DEFAULT_CHART_CONFIG_FILE}" -o json > "${HELM_CONFIG_FILE}"
output=$(oras push "${REGISTRY_URL}/${REPOSITORY_NAME}:0.0.0-${IMAGE_VERSION}" \
  "${PATH_TO_HELM_CHART}:application/vnd.cncf.helm.chart.content.v1.tar+gzip" \
  --annotation "org.opencontainers.image.authors=${authors}" \
  --annotation "org.opencontainers.image.description=${description}" \
  --annotation "org.opencontainers.image.source=${source}" \
  --annotation "org.opencontainers.image.title=${title}" \
  --annotation "org.opencontainers.image.url=${url}" \
  --annotation "org.opencontainers.image.version=${IMAGE_VERSION}" \
  --annotation "vcs-ref=${GITHUB_SHA}" \
  --config "${HELM_CONFIG_FILE}:application/vnd.cncf.helm.config.v1+json" 2>&1)
exit_status=$?

if [ $exit_status -eq 0 ]; then
  digest=$(echo "$output" | awk '/Digest:/ {print $2}')
  echo "digest=$digest" >> "${GITHUB_OUTPUT}"
else
  echo "Command failed with exit status $exit_status. Error: $output"
  exit $exit_status
fi
