#!/bin/bash
set -e

PLATFORM="${1:-openshift}"
VERSION="${2:-0.0.1}"
OLM_IMAGE="${3:-registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}}"
BUNDLE_CHANNELS="${4:-}"
BUNDLE_DEFAULT_CHANNEL="${5:-}"

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

SERVICE_ACCOUNTS=(
--extra-service-accounts dynatrace-dynakube-oneagent
--extra-service-accounts dynatrace-dynakube-oneagent-unprivileged
--extra-service-accounts dynatrace-kubernetes-monitoring
--extra-service-accounts dynatrace-activegate
)

function splitOperatorSdkOptions() {
    # shellcheck disable=SC2034
    OPERATOR_SDK_OPTIONS=("$@")
}

eval splitOperatorSdkOptions "${SERVICE_ACCOUNTS[@]}" "${BUNDLE_CHANNELS}" "${BUNDLE_DEFAULT_CHANNEL}"

"${OPERATOR_SDK}" generate kustomize manifests -q --apis-dir ./src/api/
(cd "config/deploy/${PLATFORM}" && ${KUSTOMIZE} edit set image quay.io/dynatrace/dynatrace-operator:snapshot="${OLM_IMAGE}")
"${KUSTOMIZE}" build "config/olm/${PLATFORM}" | "${OPERATOR_SDK}" generate bundle --overwrite --version "${VERSION}" "${OPERATOR_SDK_OPTIONS[@]}"
"${OPERATOR_SDK}" bundle validate ./bundle

rm -rf "./config/olm/${PLATFORM}/${VERSION}"
mkdir -p "./config/olm/${PLATFORM}/${VERSION}"
mv ./bundle/* "./config/olm/${PLATFORM}/${VERSION}"
mv "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.clusterserviceversion.yaml" "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml"
mv "./bundle.Dockerfile" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
grep -v 'scorecard' "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile" > "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output"
mv "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
sed "s/bundle/${VERSION}/" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile" > "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output"
mv "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
awk '/operators.operatorframework.io.metrics.project_layout/ { print; print "  operators.operatorframework.io.bundle.channel.default.v1: alpha"; next }1' "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml" >  "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output"
mv "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output" "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml"
awk '/operators.operatorframework.io.${VERSION}.mediatype.v1/ { print "LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha"; print; next }1' "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile" > "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output"
mv "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
grep -v '# Labels for testing.' "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile" > "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output"
mv "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile.output" "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
if [ "${PLATFORM}" = "openshift" ]; then
  # shellcheck disable=SC2129
	echo 'LABEL com.redhat.openshift.versions="v4.7,v4.8,v4.9"' >> "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
	echo 'LABEL com.redhat.delivery.operator.bundle=true' >> "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
	echo 'LABEL com.redhat.delivery.backport=true' >> "./config/olm/${PLATFORM}/bundle-${VERSION}.Dockerfile"
	sed 's/\bkubectl\b/oc/g' "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml" > "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml.output"
	mv "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml.output" "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml"
	echo '  com.redhat.openshift.versions: v4.7-v4.9' >> "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml"
fi
grep -v 'scorecard' "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml" > "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output"
grep -v '  # Annotations for testing.' "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output" > "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml"
rm "./config/olm/${PLATFORM}/${VERSION}/metadata/annotations.yaml.output"
mv "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.v${VERSION}.clusterserviceversion.yaml" "./config/olm/${PLATFORM}/${VERSION}/manifests/dynatrace-operator.clusterserviceversion.yaml"
