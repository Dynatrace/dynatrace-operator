#!/usr/bin/env bash

kind=${1}
label=${2}
condition=${3}

while true; do
  lines=$(kubectl -n dynatrace get "${kind}" --ignore-not-found -l "${label}" | wc -l)

  if [ "$lines" -gt 0 ]; then
    kubectl -n dynatrace wait "${kind}" --timeout=60s --for="${condition}" -l "${label}" 2> /dev/null
    ret=$?

    if [ $ret -eq 0 ];
    then
      exit 0
    else
      echo -n "."
    fi
  fi

  sleep 1
done
