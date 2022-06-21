#!/bin/bash
set -e


CONTAINER_CLI=docker
if ! command -v docker &> /dev/null
then
    echo "docker could not be found, using podman instead"
    CONTAINER_CLI=podman
fi

KUBERNETES_CLI=oc
if ! command -v oc &> /dev/null
then
    echo "oc could not be found, using kubectl instead"
    KUBERNETES_CLI=kubectl
fi

NAMESPACE=openshift-operators
if [ "${PLATFORM}" == "kubernetes" ]; then
    NAMESPACE=operators
fi


pushd config/olm/"${PLATFORM}"

$CONTAINER_CLI build -f bundle-"${VERSION}".Dockerfile . -t quay.io/dynatrace/olm_catalog_tests:"${TAG}"
$CONTAINER_CLI push quay.io/dynatrace/olm_catalog_tests:"${TAG}"
opm index add --container-tool $CONTAINER_CLI --bundles quay.io/dynatrace/olm_catalog_tests:"${TAG}" --tag quay.io/dynatrace/olm_index_tests:"${TAG}"
# if you want to add an existing index, append:  --from-index quay.io/dynatrace/olm_index_tests:prev-tag
$CONTAINER_CLI push quay.io/dynatrace/olm_index_tests:"${TAG}"


cat <<EOF | $KUBERNETES_CLI apply -f -
    apiVersion: operators.coreos.com/v1alpha1
    kind: CatalogSource
    metadata:
        name: dynatrace-catalog
        namespace: $NAMESPACE
        labels:
          app.kubernetes.io/name: dynatrace-operator
    spec:
        sourceType: grpc
        image: quay.io/dynatrace/olm_index_tests:${TAG}
EOF

if [ "${CREATE_SUBSCRIPTION}" == true ]; then

$KUBERNETES_CLI wait catalogsource \
  -n $NAMESPACE \
  --for="jsonpath={.status.connectionState.lastObservedState}=READY" \
  --selector=app.kubernetes.io/name=dynatrace-operator \
  --timeout=300s

cat <<EOF | $KUBERNETES_CLI apply -f -
    apiVersion: operators.coreos.com/v1alpha1
    kind: Subscription
    metadata:
        name: dynatrace-subscription
        namespace: $NAMESPACE
    spec:
        channel: alpha
        name: dynatrace-operator
        source: dynatrace-catalog
        sourceNamespace: $NAMESPACE
EOF

fi

