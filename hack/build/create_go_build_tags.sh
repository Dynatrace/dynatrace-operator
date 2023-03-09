#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <needs-e2e-tag>"
    exit 1
fi

needs_e2e_tag=$1

go_build_tags=(
    # Needed for the image library(https://github.com/containers/image), to use the open go implementation of gpgme. creating new signatures is not possible but we never do that
    "containers_image_openpgp"

     # If CGO is enabled, certain standard libraries will also use CGO, these explicitly disallow that
    "osusergo"
    "netgo"

    # Disables the ability to add load extensions for sqlite3, needed to statically build when using the sqlite library. We never use extensions so its good to disable it.
    "sqlite_omit_load_extension"

    # Removes the containers-storage/docker-daemon parts of the image library(https://github.com/containers/image) as we are not using it.
    # More info about the disabled parts:
    # - https://github.com/containers/image/blob/main/docs/containers-transports.5.md#containers-storagestorage-specifierimage-iddocker-referenceimage-id
    # - https://github.com/containers/image/blob/main/docs/containers-transports.5.md#docker-daemondocker-referencealgodigest
    "containers_image_storage_stub"
    "containers_image_docker_daemon_stub"
)

if "${needs_e2e_tag}"; then
    # Used for enabling e2e testing code
    go_build_tags+=("e2e")
fi

printf "%s," "${go_build_tags[@]}"
