#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

VERSION=${1?missing version}

create_crd_file() {
    local download_url=$1
    local crd_file=$2

    cat > "$crd_file" <<EOM
{{- if and (.Values.prometheus).installCRDs (.Values.experimental).enablePrometheus }}
# Source: $download_url
# Licensed under the Apache License, Version 2.0 (https://www.apache.org/licenses/LICENSE-2.0)
#
# CoreOS Project
# Copyright 2015 CoreOS, Inc
# This product includes software developed at CoreOS, Inc. (http://www.coreos.com/).
#
# NOTICE: This file has been modified from its original form by Dynatrace LLC.
# Modification: added helm template.
EOM

    curl --silent --retry-all-errors --fail --location "$download_url" | sed '
    /---/d;
    s/\(^  name: .*\)/\1\n  namespace: {{ .Release.Namespace }}\n  labels: {{- include \"dynatrace-operator.commonLabels\" . | nindent 4 }}/' >> "$crd_file"

    echo "{{- end }}" >> "$crd_file"
}

FILES=(
  "monitoring.coreos.com_podmonitors.yaml"
  "monitoring.coreos.com_probes.yaml"
  "monitoring.coreos.com_scrapeconfigs.yaml"
  "monitoring.coreos.com_servicemonitors.yaml"
)

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
CRD_BASE_DIR=${SCRIPT_DIR}/../../config/helm/chart/default/templates/Common/crd/prometheus
mkdir -p "${CRD_BASE_DIR}"

for file in "${FILES[@]}"; do
    URL="https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/${VERSION}/example/prometheus-operator-crd/${file}"
    if grep -q "${VERSION}" "${CRD_BASE_DIR}/${file}" 2>/dev/null; then
        echo "Prometheus Operator CRD ${file} is up-to-date"
        continue
    fi

    echo -e "Downloading Prometheus Operator CRD with Version ${VERSION}:\n${URL}\n"

    if ! create_crd_file "${URL}" "${CRD_BASE_DIR}/${file}"; then
        echo "Failed to download ${URL}!"
        rm -f "${CRD_BASE_DIR}/${file}"
        exit 1
    fi
done
