#!/usr/bin/env bash

set -x

readonly PREFLIGHT_VERSION="${1}"
readonly IMAGE_TAG="${2}"
readonly PREFLIGHT_REPORT_NAME="${3}"

readonly PREFLIGHT_EXECUTABLE="preflight-linux-amd64"
readonly PREFLIGHT_LOG="preflight.log"

download_preflight() {
  curl -LO "https://github.com/redhat-openshift-ecosystem/openshift-preflight/releases/download/${PREFLIGHT_VERSION}/${PREFLIGHT_EXECUTABLE}"
  sudo chmod +x "${PREFLIGHT_EXECUTABLE}"
}

check_image() {
  ./"${PREFLIGHT_EXECUTABLE}" check container "${IMAGE_TAG}" 1> "${PREFLIGHT_REPORT_NAME}" 2> "${PREFLIGHT_LOG}"
  echo "${PREFLIGHT_EXECUTABLE} returned ${?}"
  cat "${PREFLIGHT_LOG}"
  grep "Preflight result: PASSED" "${PREFLIGHT_LOG}" || exit 1
}

submit_report() {
  ./"${PREFLIGHT_EXECUTABLE}" check container "${IMAGE_TAG}" \
    --pyxis-api-token="${RHCC_APITOKEN}" --certification-project-id="${RHCC_PROJECT_ID}" \
    --docker-config="${HOME}/.docker/config.json" \
    --submit
}

download_preflight
check_image
readonly passed=$?
if [[ ${passed} -eq 0 ]]; then
    submit_report
fi

exit ${passed}

