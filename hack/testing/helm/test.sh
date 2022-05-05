#!/usr/bin/env bash
echo "Unit-testing helm chart"
if ! helm unittest --helm3 -f './tests/*/*/*.yaml' -f './tests/*/*.yaml' -f './tests/*.yaml' ./config/helm/chart/default; then
  echo "some tests failed"
  exit 10
fi
