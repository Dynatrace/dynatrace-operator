#!/usr/bin/env bash

desired_version="$1"

### reintroduced curl due to github rate-limiting our downloads via api calls!   ####
### -> unit tests failed as plugin could not be installed correctly              ####

HELM_PLUGINS="$(helm env HELM_PLUGINS)"

architecture="$(uname -m)"
os="$(uname -s)"

if [[ "${architecture}" == "x86_64" ]]; then
  ARCH="amd64"
elif [[ "${architecture}" == "aarch64" ]]; then
  ARCH="arm64"
elif [[ "${architecture}" == "arm64" ]]; then
  ARCH="arm64"
else
  echo "Unsupported architecture '${architecture}'"
  exit 1
fi

if [[ "${os}" == "Linux" ]]; then
  PLATFORM="linux"
elif [[ "${os}" == "Darwin" ]]; then
  PLATFORM="macos"
else
  echo "Unsupported operating system '${os}'"
  exit 1
fi


helm plugin uninstall unittest || true
curl \
  -L "https://github.com/helm-unittest/helm-unittest/releases/download/${desired_version}/helm-unittest-${PLATFORM}-${ARCH}-${desired_version//v}.tgz" \
  -o helm-unittest.tgz
mkdir -p "${HELM_PLUGINS}/unittest"
tar xzvf helm-unittest.tgz -C "${HELM_PLUGINS}/unittest"
rm helm-unittest.tgz
