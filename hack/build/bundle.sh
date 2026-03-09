#!/bin/bash
set -eo pipefail

PLATFORM="${1:-openshift}"
VERSION="${2:-0.0.1}"
BUNDLE_CHANNELS="${3:-}"
BUNDLE_DEFAULT_CHANNEL="${4:-}"
OCP_MIN_VERSION="v4.14"

lookup_cmd() {
    local cmd=$1
    # prioritize local installed binary
    if [[ -e ./bin/$1 ]]; then
        echo "$PWD/bin/$1"
    else
        command -v "$cmd"
    fi
}

if ! KUSTOMIZE=$(lookup_cmd kustomize); then
    echo "'kustomize' command not found"
    exit 2
fi


if ! OPERATOR_SDK=$(lookup_cmd operator-sdk); then
    echo "'operator-sdk' command not found"
    exit 2
fi

SDK_PARAMS=(
    --extra-service-accounts dynatrace-dynakube-oneagent
    --extra-service-accounts dynatrace-activegate
    --extra-service-accounts dynatrace-otel-collector
    --extra-service-accounts dynatrace-edgeconnect
    --extra-service-accounts dynatrace-extension-controller
    --extra-service-accounts dynatrace-sql-ext-exec
    --extra-service-accounts dynatrace-logmonitoring
    --extra-service-accounts dynatrace-node-config-collector
)

if [[ -n ${BUNDLE_CHANNELS} ]]; then
    SDK_PARAMS+=("${BUNDLE_CHANNELS}")
fi

if [[ -n ${BUNDLE_DEFAULT_CHANNEL} ]]; then
    SDK_PARAMS+=("${BUNDLE_DEFAULT_CHANNEL}")
fi

# Clear previous manifests to ensure local builds don't keep stale data
rm -rf manifests

"${OPERATOR_SDK}" generate kustomize manifests -q --apis-dir pkg/api/
"${KUSTOMIZE}" build "config/olm/${PLATFORM}" | "${OPERATOR_SDK}" generate bundle --overwrite --output-dir . --version "${VERSION}" "${SDK_PARAMS[@]}"
# Add missing aggregated ClusterRole and binding. This fixes the issue of missing cluster permissions in the CSV.
# operator-sdk looks at the RBAC on disk, but aggregated roles are rendered in the cluster.
# https://github.com/operator-framework/operator-lifecycle-manager/issues/2757
yq 'select(.kind=="ClusterRole")|select(.metadata.name=="dynatrace-kubernetes-monitoring")|.' "config/deploy/${PLATFORM}/${PLATFORM}.yaml" > manifests/dynatrace-kubernetes-monitoring_rbac.authorization.k8s.io_v1_clusterrole.yaml
yq 'select(.kind=="ClusterRoleBinding")|select(.metadata.name=="dynatrace-kubernetes-monitoring")|.' "config/deploy/${PLATFORM}/${PLATFORM}.yaml" > manifests/dynatrace-kubernetes-monitoring_rbac.authorization.k8s.io_v1_clusterrolebinding.yaml

# Set important metadata
yq -i ".metadata.annotations[\"olm.skipRange\"] = \"<${VERSION}\"" manifests/dynatrace-operator.clusterserviceversion.yaml
yq -i ".metadata.annotations.createdAt = now" manifests/dynatrace-operator.clusterserviceversion.yaml
if [[ ${PLATFORM} = "openshift" ]]; then
    {
        echo "LABEL com.redhat.openshift.versions=\"${OCP_MIN_VERSION}\""
        echo 'LABEL com.redhat.delivery.operator.bundle=true'
        echo 'LABEL com.redhat.delivery.backport=true'
    } >> bundle.Dockerfile

    yq -i ".annotations[\"com.redhat.openshift.versions\"] = \"${OCP_MIN_VERSION}\"" metadata/annotations.yaml
fi

# Move to version specific directory. It's important that the previous directory is fully deleted to prevent including stale manifests.
BUNDLE_DIR=config/olm/${PLATFORM}/${VERSION}
rm -rf "${BUNDLE_DIR}"
mkdir -p "${BUNDLE_DIR}"
mv bundle.Dockerfile metadata manifests "${BUNDLE_DIR}/"

"${OPERATOR_SDK}" bundle validate "${BUNDLE_DIR}"
