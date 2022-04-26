#!/usr/bin/env bash

kind="${1}"
label="${2}"
condition="${3}"

while ! kubectl -n dynatrace wait "${kind}" --timeout=60s --for="${condition}" -l "${label}" 2> /dev/null ; do
  sleep 1
done

