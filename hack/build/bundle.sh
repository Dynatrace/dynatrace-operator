#!/bin/bash
set -e

PLATFORM="${1:-openshift}"
VERSION="${2:-0.0.1}"
BUNDLE_CHANNELS="${3:-}"
BUNDLE_DEFAULT_CHANNEL="${4:-}"

if [ -z "$OLM_IMAGE" ]; then
  OLM_IMAGE="registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}"
  if [ "${PLATFORM}" == "kubernetes" ]; then
    OLM_IMAGE="public.ecr.aws/dynatrace/dynatrace-operator:v${VERSION}"
  fi
fi
echo "OLM image: ${OLM_IMAGE}"

KUSTOMIZE="$(hack/build/command.sh kustomize 2>/dev/null)"
if [ -z "${KUSTOMIZE}" ]; then
  echo "'kustomize' command not found"
  exit 2
fi

OPERATOR_SDK="$(hack/build/command.sh operator-sdk 2>/dev/null)"
if [ -z "${OPERATOR_SDK}" ]; then
  echo "'operator-sdk' command not found"
  exit 2
fi

SDK_PARAMS=(
--extra-service-accounts dynatrace-dynakube-oneagent
--extra-service-accounts dynatrace-kubernetes-monitoring
--extra-service-accounts dynatrace-activegate
--extra-service-accounts dynatrace-opentelemetry-collector
--extra-service-accounts dynatrace-edgeconnect
--extra-service-accounts dynatrace-extensions-controller
--extra-service-accounts dynatrace-logmonitoring
--extra-service-accounts dynatrace-node-config-collector
)

if [ -n "${BUNDLE_CHANNELS}" ]; then
    SDK_PARAMS+=("${BUNDLE_CHANNELS}")
fi

if [ -n "${BUNDLE_DEFAULT_CHANNEL}" ]; then
    SDK_PARAMS+=("${BUNDLE_DEFAULT_CHANNEL}")
fi

"${OPERATOR_SDK}" generate kustomize manifests -q --apis-dir ./pkg/api/
(cd "config/deploy/${PLATFORM}" && ${KUSTOMIZE} edit set image quay.io/dynatrace/dynatrace-operator:snapshot="${OLM_IMAGE}")
"${KUSTOMIZE}" build "config/olm/${PLATFORM}" | "${OPERATOR_SDK}" generate bundle --overwrite --version "${VERSION}" "${SDK_PARAMS[@]}"
"${OPERATOR_SDK}" bundle validate ./bundle
