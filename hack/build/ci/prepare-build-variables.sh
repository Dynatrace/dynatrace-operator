#!/bin/bash

create_docker_image_tag() {
  if [[ "${GITHUB_EVENT_NAME}" == "pull_request" ]]; then
    head_ref=$(hack/build/ci/sanitize-branch-name.sh "${GITHUB_HEAD_REF}")
    echo "snapshot-${head_ref}"; return
  elif [[ "${GITHUB_EVENT_NAME}" == "schedule" ]]; then # nightly builds
    echo "nightly-$(date --iso-8601)"; return
  fi

  if [[ "${GITHUB_REF_TYPE}" == "tag" ]]; then
    echo "${GITHUB_REF_NAME}"; return
  fi

  if [[ "${GITHUB_REF_NAME}" == "main" ]]; then
    echo "snapshot"; return
  fi

  ref_name=$(hack/build/ci/sanitize-branch-name.sh "${GITHUB_REF_NAME}")
  echo "snapshot-${ref_name}"
}

create_docker_image_labels() {
  if [[ "${GITHUB_REF_TYPE}" != "tag" ]] && [[ ! "${GITHUB_REF_NAME}" =~ ^release-* ]] && [[ "${GITHUB_REF_NAME}" != "main" ]]; then
    echo "quay.expires-after=10d"
  fi

  echo "build-date=$(date --iso-8601)"
  echo "vcs-ref=${GITHUB_SHA}"
}

print_build_variables() {
  local docker_image_tag docker_image_labels go_linker_args
  docker_image_tag=$(create_docker_image_tag)
  docker_image_labels=$(create_docker_image_labels)
  go_linker_args=$(hack/build/create_go_linker_args.sh "${docker_image_tag}" "${GITHUB_SHA}")

  echo "go_linker_args=${go_linker_args}"
  echo "docker_image_labels=${docker_image_labels}"
  echo "docker_image_tag=${docker_image_tag}"
  echo "docker_image_tag_without_prefix=${docker_image_tag#v}"
}

# prepare variables
print_build_variables >> "$GITHUB_OUTPUT"
