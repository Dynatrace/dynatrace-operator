#!/bin/sh
set -e


CONTAINER_CLI=podman
if ! command -v podman &> /dev/null
then
    echo "podman could not be found, using docker instead"
    CONTAINER_CLI=docker
fi

pushd config/olm/"${PLATFORM}"

$CONTAINER_CLI build -f bundle-"${VERSION}".Dockerfile . -t quay.io/dynatrace/olm_catalog_tests:"${TAG}"
$CONTAINER_CLI push quay.io/dynatrace/olm_catalog_tests:"${TAG}"
opm index add --container-tool $CONTAINER_CLI --bundles quay.io/dynatrace/olm_catalog_tests:"${TAG}" --tag quay.io/dynatrace/olm_index_tests:"${TAG}"
# if you want to add an existing index, append:  --from-index quay.io/dynatrace/olm_index_tests:prev-tag
$CONTAINER_CLI push quay.io/dynatrace/olm_index_tests:"${TAG}"


cat <<EOF | oc apply -f -
    apiVersion: operators.coreos.com/v1alpha1
    kind: CatalogSource
    metadata:
        name: dynatrace-catalog
        namespace: openshift-operators
    spec:
        sourceType: grpc
        image: quay.io/dynatrace/olm_index_tests:"${TAG}"
EOF


if [ "${CREATE_SUBSCRIPTION}" == true ]; then

sleep 30
cat <<EOF | oc apply -f -
    apiVersion: operators.coreos.com/v1alpha1
    kind: Subscription
    metadata:
        name: dynatrace-subscription
        namespace: openshift-operators
    spec:
        channel: alpha
        name: dynatrace-operator
        source: dynatrace-catalog
        sourceNamespace: openshift-operators
EOF

fi

