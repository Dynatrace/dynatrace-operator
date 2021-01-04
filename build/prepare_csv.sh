#!/bin/bash

set -eu

VERSION=$(echo "$TRAVIS_TAG" | sed 's/v//')

# Get the latest operator-sdk
OPERATOR_SDK_VERSION="v0.15.2"
OPERATOR_SDK="/usr/local/bin/operator-sdk"

if [ ! -f "/usr/local/bin/operator-sdk" ]; then
  curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu
  chmod +x operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu
  sudo mkdir -p /usr/local/bin/
  sudo mv operator-sdk-${OPERATOR_SDK_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk
fi

LATEST_OPERATOR_RELEASE=$(ls -d ./config/olm/kubernetes/*/ | sort -r | head -n 1 | xargs -n 1 basename)

mkdir -p ./config/olm-catalog/dynatrace-monitoring/"${LATEST_OPERATOR_RELEASE}"
mkdir -p ./config/olm/kubernetes/"${VERSION}"
mkdir -p ./config/olm/openshift/"${VERSION}"

# Copy over the latest existing version of the CSV for K8s, generate the CSV and move it back to the K8s folder
cp -r ./config/olm/kubernetes/"${LATEST_OPERATOR_RELEASE}" ./config/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version "$VERSION" --csv-config=./config/olm/config_k8s.yaml --from-version "$LATEST_OPERATOR_RELEASE" --operator-name dynatrace-monitoring
sed -i "i/dynatrace-operator:v${LATEST_OPERATOR_RELEASE}/dynatrace-operator:v${VERSION}" ./config/olm-catalog/dynatrace-monitoring/"${VERSION}"
mv ./config/olm-catalog/dynatrace-monitoring/"${VERSION}" ./config/olm/kubernetes/
rm -rf ./config/olm-catalog/dynatrace-monitoring/"${LATEST_OPERATOR_RELEASE}"

# Copy over the latest existing version of the CSV for OCP, generate the CSV and move it back to the OCP folder
cp -r ./config/olm/openshift/"${LATEST_OPERATOR_RELEASE}" ./config/olm-catalog/dynatrace-monitoring/
$OPERATOR_SDK generate csv --csv-channel alpha --csv-version "$VERSION" --csv-config=./config/olm/config_ocp.yaml --from-version "$LATEST_OPERATOR_RELEASE" --operator-name dynatrace-monitoring
sed -i "i/dynatrace-operator:v${LATEST_OPERATOR_RELEASE}/dynatrace-operator:v${VERSION}" ./config/olm-catalog/dynatrace-monitoring/"${VERSION}"
mv ./config/olm-catalog/dynatrace-monitoring/"${VERSION}" ./config/olm/openshift/
rm -rf ./config/olm-catalog/dynatrace-monitoring/"${LATEST_OPERATOR_RELEASE}"

# Remove the created folder
rm -rf ./config/olm-catalog/

# Copy CRDs to new CSV folders
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./config/olm/kubernetes/"${VERSION}"/oneagents.dynatrace.com.crd.yaml
cp ./deploy/crds/dynatrace.com_oneagents_crd.yaml ./config/olm/openshift/"${VERSION}"/oneagents.dynatrace.com.crd.yaml

# Prepare files in a separate branch and push them to github
echo -n "$GITHUB_KEY" | base64 -d >~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa

cd /tmp
git clone git@github.com:Dynatrace/dynatrace-operator.git
cd ./dynatrace-operator

cp -r "$TRAVIS_BUILD_DIR"/config/olm/kubernetes/"$VERSION" ./config/olm/kubernetes/
cp -r "$TRAVIS_BUILD_DIR"/config/olm/openshift/"$VERSION" ./config/olm/openshift/
cat "$TRAVIS_BUILD_DIR"/config/olm/openshift/oneagent.package.yaml | sed "s/${LATEST_OPERATOR_RELEASE}/${VERSION}/" >./config/olm/openshift/oneagent.package.yaml
cat "$TRAVIS_BUILD_DIR"/config/olm/kubernetes/oneagent.package.yaml | sed "s/${LATEST_OPERATOR_RELEASE}/${VERSION}/" >./config/olm/kubernetes/oneagent.package.yaml

git config user.email "cloudplatform@dynatrace.com"
git config user.name "Dynatrace Bot"

git checkout -b "csv/${TRAVIS_TAG}"
git add .
git commit -m "New CSV file for version ${TRAVIS_TAG}"
git push --set-upstream origin "csv/${TRAVIS_TAG}"
