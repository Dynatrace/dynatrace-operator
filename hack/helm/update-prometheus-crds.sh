#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

VERSION=${1?missing version}

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

_curl() {
    curl --silent --retry-all-errors --fail --location "$@"
}

LICENSE_HEADER=$(_curl "https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/$VERSION/.header" | awk '{ sub(/^\/\//, "# "); print }')

create_crd_file() {
    local download_url=$1
    local crd_file=$2

    cat > "$crd_file" <<EOM
{{- if and (.Values.prometheus).installCRDs (.Values.experimental).enablePrometheus }}
$LICENSE_HEADER

$(_curl "$download_url" | sed '/^---/d')
{{- end }}
EOM
}

FILES=(
  "crd-podmonitors.yaml         :  monitoring.coreos.com_podmonitors.yaml"
  "crd-probes.yaml              :  monitoring.coreos.com_probes.yaml"
  "crd-scrapeconfigs.yaml       :  monitoring.coreos.com_scrapeconfigs.yaml"
  "crd-servicemonitors.yaml     :  monitoring.coreos.com_servicemonitors.yaml"
)

for line in "${FILES[@]}"; do
  DESTINATION=$(echo "${line%%:*}" | xargs)
  SOURCE=$(echo "${line##*:}" | xargs)

  URL="https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/$VERSION/example/prometheus-operator-crd/$SOURCE"

  echo -e "Downloading Prometheus Operator CRD with Version ${VERSION}:\n${URL}\n"

  if ! create_crd_file "${URL}" "${SCRIPT_DIR}/../../config/helm/chart/default/templates/Common/crd/prometheus/${DESTINATION}"; then
    echo -e "Failed to download ${URL}!"
    exit 1
  fi
done
