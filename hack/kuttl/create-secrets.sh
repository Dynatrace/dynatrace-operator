#!/usr/bin/env bash
missingVars=0
if [[ -z "${APIURL}" ]]; then
  echo "environment variable APIURL not set"
  missingVars=1
fi
if [[ -z "${APITOKEN}" ]]; then
  echo "environment variable APITOKEN not set"
  missingVars=1
fi
if [[ -z "${PAASTOKEN}" ]]; then
  echo "environment variable PAASTOKEN not set"
  missingVars=1
fi
if [[ $missingVars = 1 ]]; then
  exit 1
fi

kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=${APITOKEN}" --from-literal="paasToken=${PAASTOKEN}" --dry-run=client -o yaml | kubectl apply -f -
