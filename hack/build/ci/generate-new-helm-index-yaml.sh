#!/bin/bash

if [ -z "$2" ]
then
  echo "Usage: $0 <output-dir> <version_without_prefix>"
  exit 1
fi

readonly output_dir="${1}"
readonly version_without_prefix="${2}"

cleanup() {
  echo "cleanup"
}

trap 'cleanup' ERR

cp "config/helm/repos/stable/index.yaml" "config/helm/repos/stable/index.yaml.previous"

helm repo index "${output_dir}" \
    --url "https://github.com/Dynatrace/dynatrace-operator/releases/download/v${version_without_prefix}" \
    --merge "./config/helm/repos/stable/index.yaml"

mv -v "${output_dir}"/index.yaml ./config/helm/repos/stable/index.yaml

# Fix quotes in place to minimize the diff
sed -i'' -e "s/\"/'/g" ./config/helm/repos/stable/index.yaml

rm -rf "${output_dir}"
