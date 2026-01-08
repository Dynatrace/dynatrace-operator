#!/bin/bash

mkdir -p ~/.gnupg
echo "${SECRING}" | base64 -d >~/.gnupg/secring.gpg
echo "${PASSPHRASE}" >~/.gnupg/passphrase

cleanup() {
    rm -f ~/.gnupg/secring.gpg
    rm -f ~/.gnupg/passphrase
}

trap 'cleanup' ERR

helm package \
    "./config/helm/chart/default/" \
    -d "${OUTPUT_DIR}" \
    --app-version "${VERSION_WITHOUT_PREFIX}" \
    --version "${VERSION_WITHOUT_PREFIX}" \
    --sign \
    --key "Dynatrace LLC" \
    --keyring ~/.gnupg/secring.gpg \
    --passphrase-file ~/.gnupg/passphrase
