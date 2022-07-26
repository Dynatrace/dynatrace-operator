#!/usr/bin/env bash

HELM_PLUGINS="$(helm env HELM_PLUGINS)"

architecture="$(uname -m)"
os="$(uname -s)"

if [[ "${architecture}" == "x86_64" ]]; then
  ARCH="amd64"
elif [[ "${architecture}" == "aarch64" ]]; then
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
  -L "https://github.com/quintush/helm-unittest/releases/download/v0.2.8/helm-unittest-${PLATFORM}-${ARCH}-0.2.8.tgz" \
  -o helm-unittest.tgz
mkdir -p "${HELM_PLUGINS}/unittest"
tar xzvf helm-unittest.tgz -C "${HELM_PLUGINS}/unittest"
rm helm-unittest.tgz
