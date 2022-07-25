#!/usr/bin/env bash
echo "Linting helm chart"
if ! helm template --debug ./config/helm/chart/default/ --set platform=google-marketplace --set installCRD=true --set csidriver.enabled=true; then
  echo "could not parse template. something is wrong with template files"
  exit 10
fi
if ! helm lint --debug ./config/helm/chart/default/ --set platform=google-marketplace --set installCRD=true --set csidriver.enabled=true; then
  echo "linter returned with error. check yaml formatting in files"
  exit 15
fi
