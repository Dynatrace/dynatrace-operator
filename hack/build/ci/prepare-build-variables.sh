#!/bin/bash

createDockerImageTag() {
  if [[ "${GITHUB_EVENT_NAME}" == "pull_request" ]]; then
    echo "snapshot-$(echo "${GITHUB_HEAD_REF}" | sed 's#[^a-zA-Z0-9_-]#-#g')"
  else
    if [[ "${GITHUB_REF_TYPE}" == "tag" ]]; then
      echo "${GITHUB_REF_NAME}"
    elif [[ "${GITHUB_REF_NAME}" == "master" ]]; then
      echo "snapshot"
    else
      echo "snapshot-$(echo "${GITHUB_REF_NAME}" | sed 's#[^a-zA-Z0-9_-]#-#g')"
    fi
  fi
}

createDockerImageLabels() {
  if [[ "${GITHUB_REF_TYPE}" != "tag" ]] && [[ ! "${GITHUB_REF_NAME}" =~ ^release-* ]]; then
    echo "quay.expires-after=10d"
  fi
}

setBuildRelatedVariables() {
  echo ::set-output name=go_build_args::"${go_build_args}"
  echo ::set-output name=docker_image_labels::"${docker_image_labels}"
  echo ::set-output name=docker_image_tag::"${docker_image_tag}"
}

# prepare variables
docker_image_tag=$(createDockerImageTag)
docker_image_labels=$(createDockerImageLabels)
go_build_args=$(hack/build/create_go_build_args.sh "${docker_image_tag}" "${GITHUB_SHA}")
setBuildRelatedVariables

