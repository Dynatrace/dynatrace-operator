#!/usr/bin/env bash

for directory in "$(pwd)"; do
  echo $directory
  if [[ -d "$directory/chart/default" ]]; then
    cd "$directory/chart/default" || exit 5
    if [ -f "Chart.yml" ] || [ -f "Chart.yaml" ]; then
      echo "Unit-testing $directory"
      if ! helm unittest --helm3 -f './tests/*/*/*.yaml' -f './tests/*/*.yaml' -f './tests/*.yaml' .; then
        echo "some tests failed in directory '$directory'"
        exit 10
      fi
    else
      echo "$directory does not contain Chart file. skipping..."
    fi
  fi
done
