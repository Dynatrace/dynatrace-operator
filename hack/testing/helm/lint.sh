#!/usr/bin/env bash
cd chart/default || exit 5
echo "Linting helm chart"
if ! helm template --debug --set apiUrl="test-url",apiToken="test-token",paasToken="test-token" ../../../config/helm/chart/default/; then
  echo "could not parse template. something is wrong with template files"
  exit 10
fi
if ! helm lint --debug --set apiUrl="test-url",apiToken="test-token",paasToken="test-token" ../../../config/helm/chart/default/; then
  echo "linter returned with error. check yaml formatting in files"
  exit 15
fi
