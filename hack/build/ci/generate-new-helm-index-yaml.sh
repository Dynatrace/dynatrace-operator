#!/bin/bash

cp "config/helm/repos/stable/index.yaml" "config/helm/repos/stable/index.yaml.previous"

helm repo index "${OUTPUT_DIR}" \
    --url "https://github.com/Dynatrace/dynatrace-operator/releases/download/v${VERSION_WITHOUT_PREFIX}" \
    --merge "./config/helm/repos/stable/index.yaml"

mv -v "${OUTPUT_DIR}"/index.yaml ./config/helm/repos/stable/index.yaml

# Fix quotes in place to minimize the diff
sed -i'' -e "s/\"/'/g" ./config/helm/repos/stable/index.yaml

rm -rf "${OUTPUT_DIR}"
