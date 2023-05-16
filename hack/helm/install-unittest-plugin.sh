#!/usr/bin/env bash

if ! helm plugin list | grep -q "^unittest"; then
  echo "Installing unittest plugin..."
  helm plugin install https://github.com/helm-unittest/helm-unittest.git --version v0.3.2
fi

