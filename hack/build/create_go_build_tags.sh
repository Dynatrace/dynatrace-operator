#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <needs-e2e-tag>"
    exit 1
fi

needs_e2e_tag=$1

go_build_tags=(
    "containers_image_openpgp"
    "osusergo"
    "netgo"
    "sqlite_omit_load_extension"
    "containers_image_storage_stub"
    "containers_image_docker_daemon_stub"
)

if "${needs_e2e_tag}"; then
    go_build_tags+=("e2e")
fi

printf "%s," "${go_build_tags[@]}"
