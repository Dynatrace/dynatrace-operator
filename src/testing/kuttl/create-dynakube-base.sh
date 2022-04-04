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

kubectl get ns dynatrace || kubectl create ns dynatrace

kubectl get secret -n dynatrace dynakube || kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=${APITOKEN}" --from-literal="paasToken=${PAASTOKEN}"

# create dynakube
cat <<EOF | kubectl apply -f -
  apiVersion: dynatrace.com/v1beta1
  kind: DynaKube
  metadata:
    name: dynakube
    namespace: dynatrace
  spec:
    apiUrl: ${APIURL}
    oneAgent:
      hostMonitoring: null
      classicFullStack: null
      applicationMonitoring: null
      cloudNativeFullStack: null
EOF
