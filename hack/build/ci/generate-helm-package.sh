#!/bin/bash

if [ -z "$3" ]
then
  echo "Usage: $0 <secring> <passphrase> <output-dir> <version_without_prefix>"
  exit 1
fi

readonly secring="${1}"
readonly passphrase="${2}"
readonly output_dir="${3}"
readonly version_without_prefix="${4}"

mkdir -p ~/.gnupg
echo "${secring}" | base64 -d >~/.gnupg/secring.gpg
echo "${passphrase}" >~/.gnupg/passphrase

cleanup() {
    rm -f ~/.gnupg/secring.gpg
    rm -f ~/.gnupg/passphrase
}

trap 'cleanup' ERR

helm package \
    "./config/helm/chart/default/" \
    -d "${output_dir}" \
    --app-version "${version_without_prefix}" \
    --version "${version_without_prefix}" \
    --sign \
    --key "Dynatrace LLC" \
    --keyring ~/.gnupg/secring.gpg \
    --passphrase-file ~/.gnupg/passphrase
