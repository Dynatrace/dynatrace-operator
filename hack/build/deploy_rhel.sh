#!/bin/bash

if [[ ! "${TAG}" ]]; then
  echo "TAG variable not set"
  echo "Usage: 'make deploy-local TAG=\"<your-image-tag>\"' or 'make deploy-local-easy'"
  echo "See '-h' option for help"
  exit 5
fi

help_opts='^-h$'
for o in "$@"; do
	if [[ "$o" =~ $help_opts ]]; then
		echo "Supported environment variables:"
		echo "  IMG          Operator image pull spec override"
		echo "  TAG          Operator image tag (required)"
		echo "  LOCALBUILD   Do not build in a container (required for multi-arch)"
		echo "  GOARCH       Desired CPU architecture (amd64|arm64|ppcle64|s390x)"
		echo "  QUAY_EXPIRY  Days before deleting image from quay (defaults to never)"
		exit 1
	fi
done

function print_err() {
	ERR_CMD=${ERR_CMD:-"UNKNOWN"}
	if [[ "${ERR_CMD}" != "UNKNOWN" ]]; then
		echo "ERROR: \"${ERR_CMD}\" exited abnormally. Aborting..."
	else
		echo "ERROR: An unknown error occurred. Aborting..."
	fi
}

function check_err() {
  if [[ "$?" != 0 ]]; then
	  ERR_CON=$1
	  case ${ERR_CON} in
		  GO_BUILD)
			  ERR_CMD="go build"
			  print_err
			  exit 10
			  ;;
		  GO_LICENSES)
			  ERR_CMD="go-licenses save"
			  print_err
			  exit 20
			  ;;
		  DOCKER_BUILD)
			  ERR_CMD="docker build"
			  print_err
			  exit 40
			  ;;
		  *)
			  print_err
			  exit 80
			  ;;
	  esac
  fi
}

commit=$(git rev-parse HEAD)
build_date="$(date -u +"%Y-%m-%d %H:%M:%S+00:00")"
go_build_args=(
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Version=${TAG}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.Commit=${commit}'"
  "-X 'github.com/Dynatrace/dynatrace-operator/src/version.BuildDate=${build_date}'"
)
base_image="dynatrace-operator"
out_image="${IMG:-quay.io/dynatrace/dynatrace-operator}:${TAG}"

num_check='^[0-9]+$'
if [[ -n "${QUAY_EXPIRY}" ]] && [[ "${QUAY_EXPIRY}" =~ ${num_check} ]]; then
  expiry_args="--label quay.expires-after=${QUAY_EXPIRY}d"
fi

args="${go_build_args[@]}"
if [[ "${LOCALBUILD}" ]]; then
  export CGO_ENABLED=1
  export GOOS=linux
  export GOARCH=${GOARCH:-amd64}

  go build -ldflags "${args}" -tags exclude_graphdriver_btrfs -o ./build/_output/bin/dynatrace-operator ./src/cmd/
  check_err GO_BUILD
  
  go get github.com/google/go-licenses
  go-licenses save ./... --save_path third_party_licenses --force
  check_err GO_LICENSES

  docker build . -f ./Dockerfile-localbuild -t "${base_image}" ${expiry_args} --no-cache
  check_err DOCKER_BUILD

  rm -rf ./third_party_licenses
else
  # directory required by docker copy command
  mkdir -p third_party_licenses
  docker build . -f ./Dockerfile -t "${base_image}" --build-arg "GO_BUILD_ARGS=${args}" ${expiry_args} --no-cache
  rm -rf third_party_licenses
fi

docker tag "${base_image}" "${out_image}"
docker push "${out_image}"
