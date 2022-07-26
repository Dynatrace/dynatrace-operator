#!/usr/bin/env bash

HELM_PLUGINS="$(helm env HELM_PLUGINS)"

osArch="$(uname -m)"

if [[ "${osArch}" == "x86_64" ]]; then
  ARCH=amd64
elif [[ "${osArch}" == "aarch64" ]]; then
  ARCH=arm64
else
  echo "Unsupported architecture '${osArch}'"
  exit 1
fi

helm plugin uninstall unittest || true
wget "https://github.com/quintush/helm-unittest/releases/download/v0.2.8/helm-unittest-linux-${ARCH}-0.2.8.tgz" -O helm-unittest.tgz
mkdir -p "${HELM_PLUGINS}/unittest"
tar xzvf helm-unittest.tgz -C "${HELM_PLUGINS}/unittest"
rm helm-unittest.tgz

