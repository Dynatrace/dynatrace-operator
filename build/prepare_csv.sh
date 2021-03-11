#!/bin/bash

set -eu

VERSION=$(echo "$TRAVIS_TAG" | sed 's/v//')

# Copy over the latest existing version of the CSV for K8s, generate the CSV and move it back to the K8s folder
export IMG="$OAO_IMAGE_QUAY"
make bundle

# Copy over the latest existing version of the CSV for OCP, generate the CSV and move it back to the OCP folder
export PLATFORM="openshift"
export IMG="$OAO_IMAGE_RHCC"
make bundle

# Prepare files in a separate branch and push them to github
echo -n "$GITHUB_KEY" | base64 -d >~/.ssh/id_rsa
chmod 400 ~/.ssh/id_rsa

cd /tmp
git clone git@github.com:Dynatrace/dynatrace-operator.git
cd ./dynatrace-operator

cp -r "$TRAVIS_BUILD_DIR"/config/olm/kubernetes/"$VERSION" ./config/olm/kubernetes/
cp -r "$TRAVIS_BUILD_DIR"/config/olm/openshift/"$VERSION" ./config/olm/openshift/

git config user.email "cloudplatform@dynatrace.com"
git config user.name "Dynatrace Bot"

git checkout -b "csv/${TRAVIS_TAG}"
git add .
git commit -m "New CSV file for version ${TRAVIS_TAG}"
git push --set-upstream origin "csv/${TRAVIS_TAG}"
