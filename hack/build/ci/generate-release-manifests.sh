#!/bin/bash

if [ $# -ne 3 ]; then
    echo "Usage: $0 <newline_separated_platforms> <newline_separated_images> <version>"
    exit 1
fi

newline_separated_platforms="$1"
newline_separated_images="$2"
version="$3"

IFS=$'\n' read -rd '' -a platforms <<<"$newline_separated_platforms"
IFS=$'\n' read -rd '' -a images <<<"$newline_separated_images"

if [ "${#platforms[@]}" -ne "${#images[@]}" ]; then
    echo "Arrays must have the same length"
    exit 1
fi

version_without_prefix="${version#v}"

make manifests/crd/release \
    CHART_VERSION="${version_without_prefix}" \

for ((i=0; i<${#platforms[@]}; i++)); do
    make manifests/"${platforms[$i]}" \
        IMAGE="${images[$i]}" \
        TAG="${version}" \
        OLM_IMAGE="${images[$i]}:${version}"
done
