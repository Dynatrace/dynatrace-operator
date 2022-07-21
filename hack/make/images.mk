MASTER_IMAGE ?= quay.io/dynatrace/dynatrace-operator:snapshot

# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
SNAPSHOT_SUFFIX = $(shell git branch --show-current | sed "s/[^a-zA-Z0-9_-]/-/g")
BRANCH_IMAGE ?= quay.io/dynatrace/dynatrace-operator:snapshot-${SNAPSHOT_SUFFIX}
OLM_IMAGE ?= registry.connect.redhat.com/dynatrace/dynatrace-operator:v${VERSION}

# Image URL to use all building/pushing image targets
# If the IMG variable is undefined
ifeq ($(origin IMG),undefined)
	# If the current branch is not a release branch
	ifneq ($(shell git branch --show-current | grep "^release-"),)
		# then the MASTER_IMAGE points to quay.io and has a snapshot-<branch-name> tag
		MASTER_IMAGE=$(BRANCH_IMAGE)
	# Otherwise, if the current branch is the master branch
	else ifeq ($(shell git branch --show-current), master)
		# the branch image has the same value as the master branch which has the snapshot tag
		BRANCH_IMAGE=$(MASTER_IMAGE)
	endif
endif

## Builds and pushes an Operator image with a given TAG
images/push:
	./hack/build/push_image.sh

## Builds and pushes an Operator image with a snapshot tag
images/push/tagged: export TAG=snapshot-${SNAPSHOT_SUFFIX}
images/push/tagged: images/push

## Builds and pushes the deployer image for the Google marketplace to the development environment on GCR
images/gcr/deployer:
	./hack/gcr/deployer-image.sh ":snapshot-${SNAPSHOT_SUFFIX}"
