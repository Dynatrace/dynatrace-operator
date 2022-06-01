#!/usr/bin/env bash

if [[ "$OSTYPE" == "darwin"* ]]; then
  arch=`uname -m`
  ostype=darwin
else
  arch=x86_64
  ostype=linux
fi

version=0.11.1
installurl=https://github.com/kudobuilder/kuttl/releases/download/v${version}/kubectl-kuttl_${version}_${ostype}_${arch}

if ! kubectl kuttl; then
  echo Download from: $installurl
  sudo curl -Lo /usr/local/bin/kubectl-kuttl $installurl
  sudo chmod +x /usr/local/bin/kubectl-kuttl
fi
