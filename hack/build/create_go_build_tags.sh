#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: $0 <needs-e2e-tag>"
    exit 1
fi

needs_e2e_tag=$1

go_build_tags=(
     # If CGO is enabled, certain standard libraries will also use CGO, these explicitly disallow that
    "osusergo"
    "netgo"
)

if "${needs_e2e_tag}"; then
    # Used for enabling e2e testing code
    go_build_tags+=("e2e")
fi

printf "%s," "${go_build_tags[@]}"
