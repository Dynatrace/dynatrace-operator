#!/usr/bin/env bash
cd chart/default || exit 5
echo "Unit-testing helm chart"
if ! helm unittest --helm3 -f './tests/*/*/*.yaml' -f './tests/*/*.yaml' -f './tests/*.yaml' .; then
  echo "some tests failed"
  exit 10
fi
