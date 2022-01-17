#!/usr/bin/env bash

for directory in "$(pwd)"; do

  if [[ -d "$directory/chart/default" ]]; then
    cd "$directory/chart/default" || exit 5
    if [ -f "Chart.yml" ] || [ -f "Chart.yaml" ]; then
      echo "Linting $directory"
      if ! helm template --debug --set apiUrl="test-url",apiToken="test-token",paasToken="test-token" .; then
        echo "could not parse template. something is wrong with template files of directory '$directory'"
        exit 10
      fi
      if ! helm lint --debug --set apiUrl="test-url",apiToken="test-token",paasToken="test-token" .; then
        echo "linter returned with error. check yaml formatting in files of directory '$directory'."
        exit 15
      fi
    else
      echo "$directory does not contain Chart file. skipping..."
    fi
  fi
done
