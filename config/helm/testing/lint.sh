#!/usr/bin/env bash
cd chart/default || exit 5
echo "Linting helm chart"
if ! helm template --debug .; then
  echo "could not parse template. something is wrong with template files"
  exit 10
fi
if ! helm lint --debug .; then
  echo "linter returned with error. check yaml formatting in files"
  exit 15
fi
