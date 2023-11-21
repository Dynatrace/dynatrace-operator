#!/bin/bash
set -e

cmd="${1}"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
if [ "$(go env GOBIN)" = "" ]; then
  GOBIN="$(go env GOPATH)/bin"
else
  GOBIN="$(go env GOBIN)"
fi

if [ -x "${GOBIN}/${cmd}" ]; then
  echo "${GOBIN}/${cmd}"
  exit 0
fi

if which "${cmd}" &>/dev/null ; then
  which "${cmd}" 2>/dev/null
  exit 0
fi

# The command hasn't been found
exit 1
