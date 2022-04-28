#!/usr/bin/env bash

# create dynakube
cat <<EOF | kubectl apply -f -
  apiVersion: dynatrace.com/v1beta1
  kind: DynaKube
  metadata:
    name: dynakube
    namespace: dynatrace
  spec:
    apiUrl: ${APIURL}
EOF

