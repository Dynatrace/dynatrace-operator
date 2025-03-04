#!/bin/bash

createDockerImageTag() {
  if [[ "${GITHUB_EVENT_NAME}" == "pull_request" ]]; then
    echo "snapshot-$(echo "${GITHUB_HEAD_REF}" | sed 's#[^a-zA-Z0-9_-]#-#g')"
  else
    if [[ "${GITHUB_REF_TYPE}" == "tag" ]]; then
      echo "${GITHUB_REF_NAME}"
    elif [[ "${GITHUB_REF_NAME}" == "main" ]]; then
      echo "snapshot"
    else
      echo "snapshot-$(echo "${GITHUB_REF_NAME}" | sed 's#[^a-zA-Z0-9_-]#-#g')"
    fi
  fi
}

createDockerImageLabels() {
  if [[ "${GITHUB_REF_TYPE}" != "tag" ]] && [[ ! "${GITHUB_REF_NAME}" =~ ^release-* ]] && [[ "${GITHUB_REF_NAME}" != "main" ]]; then
    echo "quay.expires-after=10d"
  fi

  echo "build-date=$(date --iso-8601)"
}

printBuildRelatedVariables() {
  echo "go_linker_args=${go_linker_args}"
  echo "go_build_tags=${go_build_tags}"
  echo "docker_image_labels=${docker_image_labels}"
  echo "docker_image_tag=${docker_image_tag}"
  echo "docker_image_tag_without_prefix=${docker_image_tag#v}"
}

# prepare variables
docker_image_tag=$(createDockerImageTag)
docker_image_labels=$(createDockerImageLabels)
go_linker_args=$(hack/build/create_go_linker_args.sh "${docker_image_tag}" "${GITHUB_SHA}")
go_build_tags=$(hack/build/create_go_build_tags.sh false)
printBuildRelatedVariables >> "$GITHUB_OUTPUT"
