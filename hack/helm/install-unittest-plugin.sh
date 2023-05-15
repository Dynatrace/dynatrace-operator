#!/usr/bin/env bash

helm plugin uninstall unittest || true
helm plugin install https://github.com/helm-unittest/helm-unittest.git --version v0.3.2
