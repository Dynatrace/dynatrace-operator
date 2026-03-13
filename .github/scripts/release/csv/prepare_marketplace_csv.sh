#!/usr/bin/env bash
# Resolves the image digest for the given marketplace and applies all CSV field
# updates in a single yq pass instead of one call per field.
#
# Required env vars: VERSION, MARKETPLACE, FORK_REPO_DIR
# Additional env vars for non-community marketplaces: RHCC_USERNAME, RHCC_PASSWORD
set -euo pipefail

case "${MARKETPLACE}" in
  community)
    image_digest=$(skopeo inspect --override-os linux --override-arch amd64 "docker://docker.io/dynatrace/dynatrace-operator:v${VERSION}" | jq -r '.Digest')
    export IMAGE="docker.io/dynatrace/dynatrace-operator@${image_digest}"
    ;;
  community-prod|certified|redhat)
    image_digest=$(skopeo inspect --override-os linux --override-arch amd64 \
      --username "${RHCC_USERNAME}" --password "${RHCC_PASSWORD}" \
      "docker://registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}" | jq -r '.Digest')
    export IMAGE="registry.connect.redhat.com/dynatrace/dynatrace-operator@${image_digest}"
    ;;
  *)
    echo "Unknown marketplace: ${MARKETPLACE}"
    exit 1
    ;;
esac

if [ "${MARKETPLACE}" = "redhat" ]; then
  csv_file="${FORK_REPO_DIR}/operators/dynatrace-operator-rhmp/${VERSION}/manifests/dynatrace-operator.clusterserviceversion.yaml"
else
  csv_file="${FORK_REPO_DIR}/operators/dynatrace-operator/${VERSION}/manifests/dynatrace-operator.clusterserviceversion.yaml"
fi

# Apply all standard CSV annotations and image references in a single yq pass.
# strenv() reads a variable from the process environment, avoiding shell interpolation
# inside the yq expression and keeping the expression readable.
yq -i '
  .metadata.annotations.containerImage = strenv(IMAGE) |
  .spec.install.spec.deployments[].spec.template.spec.containers[].image = strenv(IMAGE) |
  (.spec.install.spec.deployments[] | select(.name == "dynatrace-operator") | .spec.template.metadata.labels."dynatrace.com/install-source") = "operatorhub-" + strenv(MARKETPLACE) |
  .metadata.annotations."operators.openshift.io/valid-subscription" = "[\"Dynatrace Platform Subscription (DPS)\",\"Dynatrace Classic License\"]" |
  .metadata.annotations."features.operators.openshift.io/disconnected" = "true" |
  .metadata.annotations."features.operators.openshift.io/proxy-aware" = "true" |
  .metadata.annotations."features.operators.openshift.io/fips-compliant" = "false" |
  .metadata.annotations."features.operators.openshift.io/tls-profiles" = "false" |
  .metadata.annotations."features.operators.openshift.io/token-auth-aws" = "false" |
  .metadata.annotations."features.operators.openshift.io/token-auth-azure" = "false" |
  .metadata.annotations."features.operators.openshift.io/token-auth-gcp" = "false" |
  .spec.relatedImages = [{"name": "dynatrace-operator", "image": strenv(IMAGE)}]
' "${csv_file}"

# Redhat marketplace requires two additional marketplace.openshift.io annotations
if [ "${MARKETPLACE}" = "redhat" ]; then
  yq -i '
    .metadata.annotations."marketplace.openshift.io/remote-workflow" = "https://marketplace.redhat.com/en-us/operators/dynatrace-operator-rhmp/pricing?utm_source=openshift_console" |
    .metadata.annotations."marketplace.openshift.io/support-workflow" = "https://marketplace.redhat.com/en-us/operators/dynatrace-operator-rhmp/support?utm_source=openshift_console"
  ' "${csv_file}"
fi
