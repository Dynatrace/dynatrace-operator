#!/bin/bash

desired_version="0.3.2"

if ! helm plugin list | grep -q "^unittest.*$desired_version"; then
  helm plugin install https://github.com/helm-unittest/helm-unittest.git --version v$desired_version
fi
