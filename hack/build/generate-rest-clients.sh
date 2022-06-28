#!/usr/bin/env bash

readonly SWAGGER_VERSION="3.0.34"
readonly DESTINATION="src/dtclient2/generated"
readonly SCHEMAS_DIR="third_party/dynatrace_api/"

swagger_gen() {
  local USER="$(id -u)"
  local GROUP="$(id -g)"
  local SPEC_DIR="${1}"
  local CLIENT_DIR="${SPEC_DIR}"
  docker run --rm -u "${USER}:${GROUP}" -v "$(pwd)/${SCHEMAS_DIR}:/input:ro" -v "$(pwd)/${DESTINATION}:/output" swaggerapi/swagger-codegen-cli-v3:"${SWAGGER_VERSION}" generate -i "/input/${SPEC_DIR}/spec3.json" -l go -o "/output/${CLIENT_DIR}" 1> /dev/null
}

rm -rf "${DESTINATION}"
mkdir -p "${DESTINATION}"

for api in tenantApi tenantApiV2 ; do
  swagger_gen "${api}"
done

