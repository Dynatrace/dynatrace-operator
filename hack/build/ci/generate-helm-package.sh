#!/bin/bash

mkdir -p ~/.gnupg
echo "${SECRING}" | base64 -d >~/.gnupg/secring.gpg
echo "${PASSPHRASE}" >~/.gnupg/passphrase

cleanup() {
    rm -f ~/.gnupg/secring.gpg
    rm -f ~/.gnupg/passphrase
    [[ -n "${CHART_NAME_OVERRIDE}" ]] && git checkout -- config/helm/chart/default/Chart.yaml
    [[ -n "${IMAGE_TAG_OVERRIDE}" ]] && git checkout -- config/helm/chart/default/values.yaml
}

trap 'cleanup' ERR

if [[ -n "${CHART_NAME_OVERRIDE}" ]]; then
    yq -i ".name = \"${CHART_NAME_OVERRIDE}\"" config/helm/chart/default/Chart.yaml
fi

if [[ -n "${IMAGE_TAG_OVERRIDE}" ]]; then
    yq -i ".imageRef.tag = \"${IMAGE_TAG_OVERRIDE}\"" config/helm/chart/default/values.yaml
fi

helm package \
    "./config/helm/chart/default/" \
    -d "${OUTPUT_DIR}" \
    --app-version "${VERSION_WITHOUT_PREFIX}" \
    --version "${VERSION_WITHOUT_PREFIX}" \
    --sign \
    --key "Dynatrace LLC" \
    --keyring ~/.gnupg/secring.gpg \
    --passphrase-file ~/.gnupg/passphrase

cleanup
