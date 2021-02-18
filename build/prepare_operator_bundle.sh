#!/bin/bash

set -eu

bundle_image="./config/olm/openshift/bundle-current.Dockerfile"

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build ./config/olm/openshift -f "$bundle_image" -t "$OUT_IMAGE"
else
  docker build ./config/olm/openshift -f "$bundle_image" -t "$OUT_IMAGE" --label "$LABEL"
fi

docker push "$OUT_IMAGE"

mkdir /tmp/opm_bundle
cd /tmp/opm_bundle

curl -LO https://github.com/operator-framework/operator-registry/releases/download/v1.16.1/linux-amd64-opm
mv linux-amd64-opm opm
chmod +x opm

./opm index add --bundles "$OUT_IMAGE" --generate

if [[ $TRAVIS_BRANCH == "master" ]]; then
  docker build . -f index.Dockerfile -t "${OUT_IMAGE}"_opm
else
  docker build . -f index.Dockerfile --label "$LABEL" -t "${OUT_IMAGE}"_opm
fi

docker push "${OUT_IMAGE}"_opm
