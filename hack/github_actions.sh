#!/bin/bash

installCGODependencies() {
  sudo apt-get update
  sudo apt-get install -y libdevmapper-dev libbtrfs-dev libgpgme-dev
}

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

createGoBuildArgs() {
  version=$1
  commit=$2

  build_date="$(date -u +"%Y-%m-%dT%H:%M:%S+00:00")"
  go_build_args=(
    "-ldflags=-X 'github.com/Dynatrace/dynatrace-operator/src/version.Version=${version}'"
    "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Commit=${commit}'"
    "-X 'github.com/Dynatrace/dynatrace-operator/src/version.BuildDate=${build_date}'"
    "-s -w"
  )
  echo "${go_build_args[*]}"
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
  go_build_args=$(createGoBuildArgs "${docker_image_tag}" "${GITHUB_SHA}")
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

pushDockerImage() {
  TAG=$1

  if [[ -f "/tmp/operator-arm64.tar" ]]; then
    echo "we build for arm too => combine images"
    docker load --input /tmp/operator-arm64.tar
    docker tag operator-arm64:${TAG} ${IMAGE_QUAY}:${TAG}-arm64
    docker tag operator-amd64:${TAG} ${IMAGE_QUAY}:${TAG}-amd64
    docker push ${IMAGE_QUAY}:${TAG}-arm64
    docker push ${IMAGE_QUAY}:${TAG}-amd64

    docker manifest create ${IMAGE_QUAY}:${TAG} ${IMAGE_QUAY}:${TAG}-arm64 ${IMAGE_QUAY}:${TAG}-amd64
    docker manifest push ${IMAGE_QUAY}:${TAG}
  else
    docker tag operator-amd64:${TAG} ${IMAGE_QUAY}:${TAG}
    docker push ${IMAGE_QUAY}:${TAG}
  fi
}

# call function provided in arguments
"$@"
