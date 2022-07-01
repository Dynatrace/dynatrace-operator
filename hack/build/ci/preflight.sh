#!/usr/bin/env bash

set -xe

readonly PREFLIGHT_VERSION="${1}"
readonly IMAGE_TAG="${2}"

readonly PREFLIGHT_EXECUTABLE="preflight-linux-amd64"
readonly PREFLIGHT_REPORT_JSON="report.json"
readonly PREFLIGHT_LOG="preflight.log"

download_preflight() {
  curl -LO "https://github.com/redhat-openshift-ecosystem/openshift-preflight/releases/download/${PREFLIGHT_VERSION}/${PREFLIGHT_EXECUTABLE}"
  sudo chmod +x "${PREFLIGHT_EXECUTABLE}"
}

check_image() {
  ./"${PREFLIGHT_EXECUTABLE}" check container "${IMAGE_TAG}" 1> "${PREFLIGHT_REPORT_JSON}" 2> "${PREFLIGHT_LOG}"
  echo "${PREFLIGHT_EXECUTABLE} returned ${?}"
  cat "${PREFLIGHT_REPORT_JSON}"
  cat "${PREFLIGHT_LOG}"
  grep "Preflight result: PASSED" "${PREFLIGHT_LOG}" || exit 1
}


download_preflight
check_image
