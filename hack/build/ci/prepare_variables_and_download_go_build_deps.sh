#!/bin/bash

# aka: version
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

# aka: label
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

prepareVariablesAndDependencies() {
  # prepare variables
  docker_image_tag=$(createDockerImageTag)
  docker_image_labels=$(createDockerImageLabels)
  go_build_args=$(hack/build/ci/create_go_build_args.sh "${docker_image_tag}" "${GITHUB_SHA}")
  setBuildRelatedVariables

  # get licenses if no cache exists
  if ! [ -d ./third_party_licenses ]; then
    go get github.com/google/go-licenses && go-licenses save ./... --save_path third_party_licenses --force
  fi

  # fetch dependencies
  go get -d ./...
  ls -la $HOME/go/pkg/mod
  cp -r $HOME/go/pkg/mod .
}

prepareVariablesAndDependencies
