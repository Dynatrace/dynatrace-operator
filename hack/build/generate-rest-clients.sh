#!/usr/bin/env bash

readonly SWAGGER_VERSION="3.0.34"
readonly DESTINATION="src/dtclient2/generated"
readonly SCHEMAS_DIR="third_party/dynatrace_api/"
readonly SPEC_HASH_FILENAME=".spec-hash"

swagger_gen() {
  local USER="$(id -u)"
  local GROUP="$(id -g)"
  local SPEC_DIR="${1}"
  local CLIENT_DIR="${SPEC_DIR}"

  local SCHEMA_PATH="./input/${SPEC_DIR}/spec3.json"

  docker run --rm -u "${USER}:${GROUP}" \
    -v "$(pwd)/${SCHEMAS_DIR}:/input:ro" \
    -v "$(pwd)/${DESTINATION}:/output" \
    swaggerapi/swagger-codegen-cli-v3:"${SWAGGER_VERSION}" \
    generate -i "${SCHEMA_PATH}" -l go -o "/output/${CLIENT_DIR}" 1> /dev/null
}

spec_hash_gen() {
  local SPEC_PATH="third_party/dynatrace_api/${1}/spec3.json"
  md5sum -z "${SPEC_PATH}" | cut -f 1 -d " " | tr -d ' \n'
}

mkdir -p "${DESTINATION}"

for api in tenantApi tenantApiV2 ; do
  newHash=$(spec_hash_gen "${api}")
  oldHash="<none>"

  hashPath="${DESTINATION}/${api}/${SPEC_HASH_FILENAME}"

  if [ -e "${hashPath}" ]; then
    oldHash=$(cat "${hashPath}")
  fi

  if [ "${oldHash}" = "${newHash}" ]; then
    echo "\"${api}\" swagger client is up to date, skipping generation"
    continue
  fi

  if [ -d "${DESTINATION}/${api}" ]; then
    (cd "${DESTINATION}" && rm -rf "${api}")
  fi

  swagger_gen "${api}"

  echo "${newHash}" > "${hashPath}"
done

