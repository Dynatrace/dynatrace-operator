#!/bin/bash

VERSION=$1
NAMESPACE=${2:-dynatrace}
REGISTRY=quay.io
OPERATOR=$REGISTRY/$NAMESPACE/dynatrace-operator:v$VERSION
BUNDLE=$REGISTRY/$NAMESPACE/dynatrace-operator-bundle:v$VERSION
INDEX=$REGISTRY/$NAMESPACE/operator-index:v$(date +%Y%m%d)-$(date +%H | sed 's/^0//')
PREFIX_REGEXP='^v'
SEMVER_REGEXP='^([0-9]+\.){2}(\*|[0-9]+)(-.*)?$'

if [[ -z $VERSION ]]; then
	echo "ERROR: Not Enough Arguments."
	echo "Usage:"
	echo "  $(basename $0) <version> [namespace]"
	exit 1
fi

if [[ $VERSION =~ $PREFIX_REGEXP ]]; then
	echo "INFO: Stripping prefix 'v' from version argument..."
	VERSION=$(echo -ne $VERSION | sed -e 's/^v//')
fi

if [[ $VERSION =~ $SEMVER_REGEXP ]]; then
	continue
else
	echo "ERROR: Version argument must confirm to SemVer 2.0 specification. Exiting..."
	echo -e "See \033[36mhttps://semver.org\033[00m for more info, but please note that build"
	echo "tags (+suffix) are not supported in Kubernetes due to DNS naming constraints."
	exit 2
fi

if [[ -n "$(pwd | grep config$)" ]] && [[ -n "$(dirname $pwd | grep dynatrace-operator$)" ]]; then
	cd ..
elif [[ -z "$(pwd | grep dynatrace-operator$)" ]] || [[ ! -f ./Makefile ]]; then
	echo "ERROR: Must run from the operator project root. Exiting..."
	exit 4
fi

OLM_IMAGE=$OPERATOR PLATFORM=openshift VERSION=$VERSION make bundle
BUNDLE_IMG=$BUNDLE PLATFORM=openshift VERSION=$VERSION make bundle/build
podman login quay.io
podman push $BUNDLE
opm index add --bundles $BUNDLE -t $INDEX
podman push $INDEX

cat > ${NAMESPACE}-operators.v${VERSION}.catalogsource.yaml << EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${NAMESPACE}-operators
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: $INDEX
  displayName: ${NAMESPACE^} Operators
  publisher:
  updateStrategy:
    registryPoll:
      interval: 5m
EOF

echo "Created ./${NAMESPACE}-operators.v${VERSION}.catalogsource.yaml"
if [[ -n "$(oc whoami > /dev/null 2>&1 | grep admin)" ]]; then
	oc apply -f "${NAMESPACE}-operators.v${VERSION}.catalogsource.yaml"
else
	echo "WARNING: You don't appear to be logged into OpenShift as an admin."
	echo "You must login to OpenShift and run the following command to populate OperatorHub:"
	echo
	echo "  oc apply -f ${NAMESPACE}-operators.v${VERSION}.catalogsource.yaml"
fi
